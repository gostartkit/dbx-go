package driver

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

var registeredDialers sync.Map

func OpenMySQL(ctx context.Context, cfg *config.ConnectionConfig) (*sql.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, util.WrapLayer("config", "invalid MySQL connection config", err)
	}

	password, err := cfg.PasswordValue()
	if err != nil {
		return nil, util.WrapLayer("config", "read MySQL password", err)
	}

	dsn := mysql.NewConfig()
	dsn.User = cfg.User
	dsn.Passwd = password
	dsn.Addr = cfg.Address()
	dsn.Net = "tcp"
	dsn.AllowNativePasswords = true
	dsn.Params = map[string]string{
		"charset": "utf8mb4",
	}

	if cfg.Mode == "ssh" {
		dsn.Net, err = registerSSHDialer(cfg)
		if err != nil {
			return nil, util.WrapLayer("ssh", "prepare SSH tunnel", err)
		}
	}

	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return nil, util.WrapLayer("mysql", "create MySQL client", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout())
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, util.WrapLayer("mysql", "ping database", err)
	}

	return db, nil
}

func ListDatabases(ctx context.Context, db *sql.DB) ([]string, error) {
	return QueryStrings(ctx, db, "SHOW DATABASES")
}

func Ping(ctx context.Context, db *sql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return util.WrapLayer("mysql", "ping database", err)
	}
	return nil
}

func QueryStrings(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.WrapLayer("sql execution", "run query", err)
	}
	defer rows.Close()

	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, util.WrapLayer("sql execution", "scan query result", err)
		}
		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "read query rows", err)
	}

	return values, nil
}

func ExecStatement(ctx context.Context, db *sql.DB, statement string) error {
	if _, err := db.ExecContext(ctx, statement); err != nil {
		return util.WrapLayer("sql execution", "execute statement", err)
	}
	return nil
}

func registerSSHDialer(cfg *config.ConnectionConfig) (string, error) {
	network := "dbx+ssh+" + sshDialerID(cfg)
	if _, loaded := registeredDialers.LoadOrStore(network, struct{}{}); loaded {
		return network, nil
	}

	mysql.RegisterDialContext(network, func(ctx context.Context, addr string) (net.Conn, error) {
		return openSSHTunnel(ctx, cfg, addr)
	})

	return network, nil
}

func sshDialerID(cfg *config.ConnectionConfig) string {
	sum := sha1.Sum([]byte(strings.Join([]string{
		cfg.Name,
		cfg.Host,
		fmt.Sprintf("%d", cfg.Port),
		cfg.Mode,
		cfg.User,
		cfg.Driver,
		cfg.SSH.Host,
		fmt.Sprintf("%d", cfg.SSH.Port),
		cfg.SSH.User,
		cfg.SSH.PrivateKey,
		cfg.SSH.PasswordEnv,
	}, "|")))
	return hex.EncodeToString(sum[:8])
}

func openSSHTunnel(ctx context.Context, cfg *config.ConnectionConfig, targetAddr string) (net.Conn, error) {
	authMethods, err := sshAuthMethods(cfg.SSH)
	if err != nil {
		return nil, util.WrapLayer("ssh", "build auth methods", err)
	}

	hostKeyCallback, err := sshHostKeyCallback()
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.SSH.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         cfg.ConnectTimeout(),
	}

	sshAddr := fmt.Sprintf("%s:%d", cfg.SSH.Host, cfg.SSH.Port)
	baseConn, err := (&net.Dialer{Timeout: cfg.ConnectTimeout()}).DialContext(ctx, "tcp", sshAddr)
	if err != nil {
		return nil, util.WrapLayer("ssh", "dial SSH server "+sshAddr, err)
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(baseConn, sshAddr, clientConfig)
	if err != nil {
		baseConn.Close()
		return nil, util.WrapLayer("ssh", "complete SSH handshake with "+sshAddr, err)
	}

	client := ssh.NewClient(clientConn, chans, reqs)
	conn, err := client.Dial("tcp", targetAddr)
	if err != nil {
		client.Close()
		return nil, util.WrapLayer("ssh", "dial database target "+targetAddr+" through tunnel", err)
	}

	return &sshBackedConn{
		Conn:   conn,
		client: client,
	}, nil
}

func sshAuthMethods(cfg *config.SSHConfig) ([]ssh.AuthMethod, error) {
	methods := make([]ssh.AuthMethod, 0, 2)

	if strings.TrimSpace(cfg.PrivateKey) != "" {
		privateKeyPath, err := cfg.PrivateKeyPath()
		if err != nil {
			return nil, util.WrapLayer("config", "expand SSH private key path", err)
		}

		keyData, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return nil, util.WrapLayer("ssh", "read SSH private key", err)
		}

		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, util.WrapLayer("ssh", "parse SSH private key", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	if strings.TrimSpace(cfg.PasswordEnv) != "" {
		password := os.Getenv(cfg.PasswordEnv)
		if password == "" {
			return nil, util.WrapLayer("config", "read SSH password", fmt.Errorf("environment variable %s is empty", cfg.PasswordEnv))
		}
		methods = append(methods, ssh.Password(password))
	}
	if strings.TrimSpace(cfg.Password) != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("SSH auth requires private_key or password_env or password")
	}

	return methods, nil
}

func sshHostKeyCallback() (ssh.HostKeyCallback, error) {
	paths, err := knownHostsPaths()
	if err != nil {
		return nil, util.WrapLayer("config", "resolve known_hosts paths", err)
	}
	if len(paths) == 0 {
		return nil, util.WrapLayer("ssh", "verify host key", fmt.Errorf("known_hosts file not found; create ~/.ssh/known_hosts with ssh-keyscan -H <host> >> ~/.ssh/known_hosts or set DBX_KNOWN_HOSTS"))
	}

	callback, err := knownhosts.New(paths...)
	if err != nil {
		return nil, util.WrapLayer("ssh", "load known_hosts", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := callback(hostname, remote, key); err != nil {
			return knownHostsError(paths, hostname, err)
		}
		return nil
	}, nil
}

func knownHostsPaths() ([]string, error) {
	if value := strings.TrimSpace(os.Getenv("DBX_KNOWN_HOSTS")); value != "" {
		rawPaths := filepath.SplitList(value)
		paths := make([]string, 0, len(rawPaths))
		for _, rawPath := range rawPaths {
			expanded, err := util.ExpandHome(rawPath)
			if err != nil {
				return nil, err
			}
			if fileExists(expanded) {
				paths = append(paths, expanded)
			}
		}
		if len(paths) > 0 {
			return paths, nil
		}
		return nil, nil
	}

	candidates := []string{"~/.ssh/known_hosts", "~/.ssh/known_hosts2"}
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		expanded, err := util.ExpandHome(candidate)
		if err != nil {
			return nil, err
		}
		if fileExists(expanded) {
			paths = append(paths, expanded)
		}
	}
	return paths, nil
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func knownHostsError(paths []string, hostname string, err error) error {
	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		hostLabel := stripPort(hostname)
		if len(keyErr.Want) == 0 {
			return util.WrapLayer("ssh", "verify host key", fmt.Errorf("host %s is not in known_hosts (%s); add it with ssh-keyscan -H %s >> %s", hostLabel, strings.Join(paths, ", "), hostLabel, paths[0]))
		}
		return util.WrapLayer("ssh", "verify host key", fmt.Errorf("host key mismatch for %s in %s", hostLabel, strings.Join(paths, ", ")))
	}
	return util.WrapLayer("ssh", "verify host key", err)
}

func stripPort(hostname string) string {
	host, _, err := net.SplitHostPort(hostname)
	if err != nil {
		return hostname
	}
	return host
}

type sshBackedConn struct {
	net.Conn
	client *ssh.Client
}

func (c *sshBackedConn) Close() error {
	connErr := c.Conn.Close()
	clientErr := c.client.Close()
	if connErr != nil {
		return connErr
	}
	return clientErr
}
