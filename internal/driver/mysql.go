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
	xproxy "golang.org/x/net/proxy"

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

	if cfg.Mode == "proxy" {
		dsn.Net, err = registerProxyDialer(cfg)
		if err != nil {
			return nil, util.WrapLayer("proxy", "prepare proxy dialer", err)
		}
	}
	if cfg.UsesSSH() {
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

func DiagnoseMySQL(ctx context.Context, cfg *config.ConnectionConfig) (*DiagnosticTrace, error) {
	trace := &DiagnosticTrace{
		Steps: make([]DiagnosticStep, 0, 3),
	}

	if cfg.Mode == "proxy" || cfg.Mode == "proxy-ssh" {
		target := cfg.Address()
		if cfg.Mode == "proxy-ssh" && cfg.SSH != nil {
			target = fmt.Sprintf("%s:%d", cfg.SSH.Host, cfg.SSH.Port)
		}
		step, err := diagnoseProxyStep(ctx, cfg, target)
		trace.Steps = append(trace.Steps, step)
		if err != nil {
			return trace, err
		}
	}

	if cfg.Mode == "ssh" || cfg.Mode == "proxy-ssh" {
		step, err := diagnoseSSHStep(ctx, cfg)
		trace.Steps = append(trace.Steps, step)
		if err != nil {
			return trace, err
		}
	}

	step, err := diagnoseMySQLStep(ctx, cfg)
	trace.Steps = append(trace.Steps, step)
	if err != nil {
		return trace, err
	}

	return trace, nil
}

func ListDatabases(ctx context.Context, db *sql.DB) ([]string, error) {
	return QueryStrings(ctx, db, "SHOW DATABASES")
}

type TableColumn struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Null    string `json:"null,omitempty"`
	Key     string `json:"key,omitempty"`
	Default string `json:"default,omitempty"`
	Extra   string `json:"extra,omitempty"`
}

type TableIndex struct {
	Name       string `json:"name"`
	Column     string `json:"column"`
	Type       string `json:"type"`
	SeqInIndex int    `json:"seq_in_index,omitempty"`
}

type Process struct {
	ID          int64  `json:"id"`
	User        string `json:"user"`
	Host        string `json:"host"`
	Database    string `json:"database,omitempty"`
	Command     string `json:"command"`
	TimeSeconds int64  `json:"time_seconds"`
	State       string `json:"state,omitempty"`
	Info        string `json:"info,omitempty"`
}

type SystemVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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

func ListTables(ctx context.Context, db *sql.DB, database string) ([]string, error) {
	query := "SHOW TABLES FROM " + util.QuoteMySQLIdentifier(database)
	return QueryStrings(ctx, db, query)
}

func DescribeTable(ctx context.Context, db *sql.DB, database string, table string) ([]TableColumn, error) {
	query := fmt.Sprintf(
		"DESCRIBE %s.%s",
		util.QuoteMySQLIdentifier(database),
		util.QuoteMySQLIdentifier(table),
	)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.WrapLayer("sql execution", "run query", err)
	}
	defer rows.Close()

	columns := make([]TableColumn, 0)
	for rows.Next() {
		var (
			field      string
			typ        string
			nullValue  sql.NullString
			keyValue   sql.NullString
			defaultVal sql.NullString
			extraValue sql.NullString
		)
		if err := rows.Scan(&field, &typ, &nullValue, &keyValue, &defaultVal, &extraValue); err != nil {
			return nil, util.WrapLayer("sql execution", "scan query result", err)
		}
		columns = append(columns, TableColumn{
			Name:    field,
			Type:    typ,
			Null:    nullValue.String,
			Key:     keyValue.String,
			Default: defaultVal.String,
			Extra:   extraValue.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "read query rows", err)
	}
	return columns, nil
}

func ShowIndexes(ctx context.Context, db *sql.DB, database string, table string) ([]TableIndex, error) {
	query := fmt.Sprintf(
		"SHOW INDEXES FROM %s FROM %s",
		util.QuoteMySQLIdentifier(table),
		util.QuoteMySQLIdentifier(database),
	)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
			return nil, util.WrapLayer("validation", "show indexes", fmt.Errorf("table not found: %s", table))
		}
		return nil, util.WrapLayer("sql execution", "run query", err)
	}
	defer rows.Close()

	indexes := make([]TableIndex, 0)
	for rows.Next() {
		var (
			tableName    string
			nonUnique    sql.NullInt64
			keyName      string
			seqInIndex   int
			columnName   string
			collation    sql.NullString
			cardinality  sql.NullInt64
			subPart      sql.NullInt64
			packed       sql.NullString
			nullValue    sql.NullString
			indexType    sql.NullString
			comment      sql.NullString
			indexComment sql.NullString
			visible      sql.NullString
			expression   sql.NullString
		)
		if err := rows.Scan(
			&tableName,
			&nonUnique,
			&keyName,
			&seqInIndex,
			&columnName,
			&collation,
			&cardinality,
			&subPart,
			&packed,
			&nullValue,
			&indexType,
			&comment,
			&indexComment,
			&visible,
			&expression,
		); err != nil {
			return nil, util.WrapLayer("sql execution", "scan query result", err)
		}
		indexes = append(indexes, TableIndex{
			Name:       keyName,
			Column:     columnName,
			Type:       indexType.String,
			SeqInIndex: seqInIndex,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "read query rows", err)
	}
	return indexes, nil
}

func ShowGrants(ctx context.Context, db *sql.DB, user string, host string) ([]string, error) {
	query := fmt.Sprintf(
		"SHOW GRANTS FOR '%s'@'%s'",
		util.EscapeMySQLString(user),
		util.EscapeMySQLString(host),
	)
	return QueryStrings(ctx, db, query)
}

func ShowProcesslist(ctx context.Context, db *sql.DB) ([]Process, error) {
	rows, err := db.QueryContext(ctx, "SHOW PROCESSLIST")
	if err != nil {
		return nil, util.WrapLayer("sql execution", "run query", err)
	}
	defer rows.Close()

	processes := make([]Process, 0)
	for rows.Next() {
		var (
			id      int64
			user    sql.NullString
			host    sql.NullString
			dbName  sql.NullString
			command sql.NullString
			timeVal sql.NullInt64
			state   sql.NullString
			info    sql.NullString
		)
		if err := rows.Scan(&id, &user, &host, &dbName, &command, &timeVal, &state, &info); err != nil {
			return nil, util.WrapLayer("sql execution", "scan query result", err)
		}
		processes = append(processes, Process{
			ID:          id,
			User:        user.String,
			Host:        host.String,
			Database:    dbName.String,
			Command:     command.String,
			TimeSeconds: timeVal.Int64,
			State:       state.String,
			Info:        info.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "read query rows", err)
	}
	return processes, nil
}

func ShowVariables(ctx context.Context, db *sql.DB, pattern string) ([]SystemVariable, error) {
	query := "SHOW VARIABLES"
	if strings.TrimSpace(pattern) != "" {
		query += " LIKE '" + util.EscapeMySQLString(pattern) + "'"
	}

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.WrapLayer("sql execution", "run query", err)
	}
	defer rows.Close()

	variables := make([]SystemVariable, 0)
	for rows.Next() {
		var (
			name  string
			value sql.NullString
		)
		if err := rows.Scan(&name, &value); err != nil {
			return nil, util.WrapLayer("sql execution", "scan query result", err)
		}
		variables = append(variables, SystemVariable{
			Name:  name,
			Value: value.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "read query rows", err)
	}
	return variables, nil
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

func diagnoseProxyStep(ctx context.Context, cfg *config.ConnectionConfig, targetAddr string) (DiagnosticStep, error) {
	details := map[string]any{
		"url":    config.RedactProxyURL(cfg.Proxy.URL),
		"target": targetAddr,
	}

	startedAt := time.Now()
	conn, err := openProxyConn(ctx, cfg, targetAddr)
	details["duration_ms"] = time.Since(startedAt).Milliseconds()
	if err != nil {
		return DiagnosticStep{
			Name:    "proxy",
			Status:  "fail",
			Error:   diagnosticErrorText(err),
			Details: details,
		}, err
	}
	if conn != nil {
		_ = conn.Close()
	}

	return DiagnosticStep{
		Name:    "proxy",
		Status:  "ok",
		Details: details,
	}, nil
}

func diagnoseSSHStep(ctx context.Context, cfg *config.ConnectionConfig) (DiagnosticStep, error) {
	sshAddr := fmt.Sprintf("%s:%d", cfg.SSH.Host, cfg.SSH.Port)
	details := map[string]any{
		"host": sshAddr,
		"user": cfg.SSH.User,
	}

	startedAt := time.Now()
	authMethods, err := sshAuthMethods(cfg.SSH)
	if err != nil {
		details["duration_ms"] = time.Since(startedAt).Milliseconds()
		return DiagnosticStep{
			Name:    "ssh",
			Status:  "fail",
			Error:   diagnosticErrorText(err),
			Details: details,
		}, err
	}

	hostKeyCallback, err := sshHostKeyCallback()
	if err != nil {
		details["duration_ms"] = time.Since(startedAt).Milliseconds()
		return DiagnosticStep{
			Name:    "ssh",
			Status:  "fail",
			Error:   diagnosticErrorText(err),
			Details: details,
		}, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.SSH.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         cfg.ConnectTimeout(),
	}

	baseConn, err := dialSSHBaseConn(ctx, cfg, sshAddr)
	if err != nil {
		details["duration_ms"] = time.Since(startedAt).Milliseconds()
		return DiagnosticStep{
			Name:    "ssh",
			Status:  "fail",
			Error:   diagnosticErrorText(err),
			Details: details,
		}, err
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(baseConn, sshAddr, clientConfig)
	if err != nil {
		_ = baseConn.Close()
		details["duration_ms"] = time.Since(startedAt).Milliseconds()
		wrapped := util.WrapLayer("ssh", "complete SSH handshake with "+sshAddr, err)
		return DiagnosticStep{
			Name:    "ssh",
			Status:  "fail",
			Error:   diagnosticErrorText(wrapped),
			Details: details,
		}, wrapped
	}

	client := ssh.NewClient(clientConn, chans, reqs)
	_ = client.Close()
	details["duration_ms"] = time.Since(startedAt).Milliseconds()
	return DiagnosticStep{
		Name:    "ssh",
		Status:  "ok",
		Details: details,
	}, nil
}

func diagnoseMySQLStep(ctx context.Context, cfg *config.ConnectionConfig) (DiagnosticStep, error) {
	details := map[string]any{
		"target": cfg.Address(),
	}

	startedAt := time.Now()
	db, err := OpenMySQL(ctx, cfg)
	details["duration_ms"] = time.Since(startedAt).Milliseconds()
	if err != nil {
		return DiagnosticStep{
			Name:    "mysql",
			Status:  "fail",
			Error:   diagnosticErrorText(err),
			Details: details,
		}, err
	}
	if db != nil {
		_ = db.Close()
	}

	return DiagnosticStep{
		Name:    "mysql",
		Status:  "ok",
		Details: details,
	}, nil
}

func diagnosticErrorText(err error) string {
	current := err
	for {
		layerErr, ok := current.(*util.LayerError)
		if !ok || layerErr.Err == nil {
			break
		}
		current = layerErr.Err
	}
	if current == nil {
		return ""
	}
	return current.Error()
}

func registerProxyDialer(cfg *config.ConnectionConfig) (string, error) {
	network := "dbx+proxy+" + proxyDialerID(cfg)
	if _, loaded := registeredDialers.LoadOrStore(network, struct{}{}); loaded {
		return network, nil
	}

	mysql.RegisterDialContext(network, func(ctx context.Context, addr string) (net.Conn, error) {
		return openProxyConn(ctx, cfg, addr)
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
		proxyURLForHash(cfg),
	}, "|")))
	return hex.EncodeToString(sum[:8])
}

func proxyURLForHash(cfg *config.ConnectionConfig) string {
	if cfg == nil || cfg.Proxy == nil {
		return ""
	}
	return cfg.Proxy.URL
}

func proxyDialerID(cfg *config.ConnectionConfig) string {
	sum := sha1.Sum([]byte(strings.Join([]string{
		cfg.Name,
		cfg.Host,
		fmt.Sprintf("%d", cfg.Port),
		cfg.Mode,
		cfg.User,
		cfg.Driver,
		proxyURLForHash(cfg),
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
	baseConn, err := dialSSHBaseConn(ctx, cfg, sshAddr)
	if err != nil {
		return nil, err
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

func openProxyConn(ctx context.Context, cfg *config.ConnectionConfig, targetAddr string) (net.Conn, error) {
	settings, err := proxyDialerSettings(cfg)
	if err != nil {
		return nil, err
	}

	dialer, err := xproxy.SOCKS5("tcp", settings.Address, settings.Auth, xproxy.Direct)
	if err != nil {
		return nil, util.WrapLayer("proxy", "create SOCKS5 dialer for "+settings.RedactedURL, err)
	}

	conn, err := dialProxyWithContext(ctx, dialer, "tcp", targetAddr)
	if err != nil {
		return nil, util.WrapLayer("proxy", "dial "+settings.RedactedURL, err)
	}
	return conn, nil
}

type proxyDialSettings struct {
	Address     string
	Auth        *xproxy.Auth
	RedactedURL string
}

func dialSSHBaseConn(ctx context.Context, cfg *config.ConnectionConfig, sshAddr string) (net.Conn, error) {
	if cfg.Mode != "proxy-ssh" {
		conn, err := (&net.Dialer{Timeout: cfg.ConnectTimeout()}).DialContext(ctx, "tcp", sshAddr)
		if err != nil {
			return nil, util.WrapLayer("ssh", "dial SSH server "+sshAddr, err)
		}
		return conn, nil
	}

	settings, err := proxyDialerSettings(cfg)
	if err != nil {
		return nil, err
	}

	dialer, err := xproxy.SOCKS5("tcp", settings.Address, settings.Auth, xproxy.Direct)
	if err != nil {
		return nil, util.WrapLayer("proxy", "create SOCKS5 dialer for "+settings.RedactedURL, err)
	}

	conn, err := dialProxyWithContext(ctx, dialer, "tcp", sshAddr)
	if err != nil {
		return nil, util.WrapLayer("proxy", "dial "+settings.RedactedURL, err)
	}
	return conn, nil
}

func proxyDialerSettings(cfg *config.ConnectionConfig) (*proxyDialSettings, error) {
	if cfg == nil || cfg.Proxy == nil {
		return nil, util.WrapLayer("config", "read proxy settings", fmt.Errorf("proxy settings are required"))
	}

	parsed, err := config.ParseProxyURL(cfg.Proxy.URL)
	if err != nil {
		return nil, util.WrapLayer("validation", "parse proxy URL", err)
	}

	settings := &proxyDialSettings{
		Address:     parsed.Address,
		RedactedURL: config.RedactProxyURL(cfg.Proxy.URL),
	}
	if parsed.Username != "" {
		settings.Auth = &xproxy.Auth{
			User:     parsed.Username,
			Password: parsed.Password,
		}
	}
	return settings, nil
}

func dialProxyWithContext(ctx context.Context, dialer xproxy.Dialer, network string, address string) (net.Conn, error) {
	type dialResult struct {
		conn net.Conn
		err  error
	}

	resultCh := make(chan dialResult, 1)
	go func() {
		conn, err := dialer.Dial(network, address)
		select {
		case resultCh <- dialResult{conn: conn, err: err}:
		case <-ctx.Done():
			if conn != nil {
				conn.Close()
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		return result.conn, result.err
	}
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
