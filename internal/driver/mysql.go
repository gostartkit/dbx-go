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

func ListDatabases(ctx context.Context, db *sql.DB) ([]string, error) {
	return QueryStrings(ctx, db, "SHOW DATABASES")
}

type SchemaColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Key      string `json:"key,omitempty"`
	Extra    string `json:"extra,omitempty"`
}

type RowSet struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

type TableStatus struct {
	Name        string `json:"name"`
	Engine      string `json:"engine"`
	Rows        int64  `json:"rows"`
	DataLength  int64  `json:"data_length"`
	IndexLength int64  `json:"index_length"`
	Collation   string `json:"collation,omitempty"`
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

func ShowColumns(ctx context.Context, db *sql.DB, database string, table string) ([]SchemaColumn, error) {
	rows, err := db.QueryContext(ctx, `
SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, EXTRA
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
ORDER BY ORDINAL_POSITION
`, database, table)
	if err != nil {
		return nil, util.WrapLayer("sql execution", "run query", err)
	}
	defer rows.Close()

	columns := make([]SchemaColumn, 0)
	for rows.Next() {
		var (
			name       string
			columnType string
			nullable   string
			key        sql.NullString
			extra      sql.NullString
		)
		if err := rows.Scan(&name, &columnType, &nullable, &key, &extra); err != nil {
			return nil, util.WrapLayer("sql execution", "scan query result", err)
		}
		columns = append(columns, SchemaColumn{
			Name:     name,
			Type:     columnType,
			Nullable: strings.EqualFold(nullable, "YES"),
			Key:      key.String,
			Extra:    extra.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "read query rows", err)
	}
	if len(columns) == 0 {
		exists, err := tableExists(ctx, db, database, table)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, util.WrapLayer("validation", "show columns", fmt.Errorf("table not found: %s", table))
		}
	}
	return columns, nil
}

func PeekRows(ctx context.Context, db *sql.DB, database string, table string, limit int) (*RowSet, error) {
	return queryRows(ctx, db, database, table, limit, false)
}

func SampleRows(ctx context.Context, db *sql.DB, database string, table string, limit int) (*RowSet, error) {
	return queryRows(ctx, db, database, table, limit, true)
}

func ShowCreateTable(ctx context.Context, db *sql.DB, database string, table string) (string, error) {
	var ddl string
	err := withDatabaseConn(ctx, db, database, func(conn *sql.Conn) error {
		rows, err := conn.QueryContext(ctx, "SHOW CREATE TABLE "+util.QuoteMySQLIdentifier(table))
		if err != nil {
			return classifyTableOperationError("show create table", table, err)
		}
		defer rows.Close()

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return util.WrapLayer("sql execution", "read query rows", err)
			}
			return util.WrapLayer("validation", "show create table", fmt.Errorf("table not found: %s", table))
		}

		var tableName string
		if err := rows.Scan(&tableName, &ddl); err != nil {
			return util.WrapLayer("sql execution", "scan query result", err)
		}
		if err := rows.Err(); err != nil {
			return util.WrapLayer("sql execution", "read query rows", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return ddl, nil
}

func ShowTableStatus(ctx context.Context, db *sql.DB, database string, table string) ([]TableStatus, error) {
	statuses := make([]TableStatus, 0)
	err := withDatabaseConn(ctx, db, database, func(conn *sql.Conn) error {
		query := "SHOW TABLE STATUS"
		if strings.TrimSpace(table) != "" {
			query += " LIKE '" + util.EscapeMySQLString(table) + "'"
		}

		rows, err := conn.QueryContext(ctx, query)
		if err != nil {
			return util.WrapLayer("sql execution", "run query", err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				name          string
				engine        sql.NullString
				version       sql.NullInt64
				rowFormat     sql.NullString
				rowsValue     sql.NullInt64
				avgRowLength  sql.NullInt64
				dataLength    sql.NullInt64
				maxDataLength sql.NullInt64
				indexLength   sql.NullInt64
				dataFree      sql.NullInt64
				autoIncrement sql.NullInt64
				createTime    sql.NullTime
				updateTime    sql.NullTime
				checkTime     sql.NullTime
				collation     sql.NullString
				checksum      sql.NullInt64
				createOptions sql.NullString
				comment       sql.NullString
			)
			if err := rows.Scan(
				&name,
				&engine,
				&version,
				&rowFormat,
				&rowsValue,
				&avgRowLength,
				&dataLength,
				&maxDataLength,
				&indexLength,
				&dataFree,
				&autoIncrement,
				&createTime,
				&updateTime,
				&checkTime,
				&collation,
				&checksum,
				&createOptions,
				&comment,
			); err != nil {
				return util.WrapLayer("sql execution", "scan query result", err)
			}
			statuses = append(statuses, TableStatus{
				Name:        name,
				Engine:      engine.String,
				Rows:        rowsValue.Int64,
				DataLength:  dataLength.Int64,
				IndexLength: indexLength.Int64,
				Collation:   collation.String,
			})
		}
		if err := rows.Err(); err != nil {
			return util.WrapLayer("sql execution", "read query rows", err)
		}
		if strings.TrimSpace(table) != "" && len(statuses) == 0 {
			return util.WrapLayer("validation", "show table status", fmt.Errorf("table not found: %s", table))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return statuses, nil
}

func ExecStatement(ctx context.Context, db *sql.DB, statement string) error {
	if _, err := db.ExecContext(ctx, statement); err != nil {
		return util.WrapLayer("sql execution", "execute statement", err)
	}
	return nil
}

func queryRows(ctx context.Context, db *sql.DB, database string, table string, limit int, random bool) (*RowSet, error) {
	result := &RowSet{Columns: []string{}, Rows: [][]any{}}
	err := withDatabaseConn(ctx, db, database, func(conn *sql.Conn) error {
		query := "SELECT * FROM " + util.QuoteMySQLIdentifier(table)
		if random {
			query += " ORDER BY RAND()"
		}
		query += " LIMIT ?"

		rows, err := conn.QueryContext(ctx, query, limit)
		if err != nil {
			return classifyTableOperationError(operationNameForRows(random), table, err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return util.WrapLayer("sql execution", "read query columns", err)
		}
		result.Columns = append(result.Columns, columns...)

		for rows.Next() {
			values := make([]any, len(columns))
			dest := make([]any, len(columns))
			for i := range values {
				dest[i] = &values[i]
			}
			if err := rows.Scan(dest...); err != nil {
				return util.WrapLayer("sql execution", "scan query result", err)
			}
			row := make([]any, 0, len(values))
			for _, value := range values {
				row = append(row, normalizeRowValue(value))
			}
			result.Rows = append(result.Rows, row)
		}
		if err := rows.Err(); err != nil {
			return util.WrapLayer("sql execution", "read query rows", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func tableExists(ctx context.Context, db *sql.DB, database string, table string) (bool, error) {
	row := db.QueryRowContext(ctx, `
SELECT 1
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
LIMIT 1
`, database, table)
	var marker int
	if err := row.Scan(&marker); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, util.WrapLayer("sql execution", "run query", err)
	}
	return true, nil
}

func normalizeRowValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format("2006-01-02 15:04:05")
	default:
		return typed
	}
}

func operationNameForRows(random bool) string {
	if random {
		return "sample rows"
	}
	return "peek rows"
}

func withDatabaseConn(ctx context.Context, db *sql.DB, database string, fn func(conn *sql.Conn) error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return util.WrapLayer("mysql", "open database connection", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "USE "+util.QuoteMySQLIdentifier(database)); err != nil {
		return util.WrapLayer("mysql", "select database", err)
	}
	return fn(conn)
}

func execWithDatabase(ctx context.Context, db *sql.DB, database string, statement string, operation string, table string) error {
	return withDatabaseConn(ctx, db, database, func(conn *sql.Conn) error {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return classifyTableOperationError(operation, table, err)
		}
		return nil
	})
}

func classifyTableOperationError(operation string, table string, err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
		return util.WrapLayer("validation", operation, fmt.Errorf("table not found: %s", table))
	}
	return util.WrapLayer("sql execution", "run query", err)
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
