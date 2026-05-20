package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
)

type databaseSelectionConnector struct {
	databases     []string
	tables        []string
	users         []string
	columns       []driver.SchemaColumn
	indexes       []driver.TableIndex
	foreignKeys   []driver.ForeignKey
	createDDL     string
	statuses      []driver.TableStatus
	triggers      []driver.Trigger
	views         []string
	processes     []driver.Process
	variables     []driver.SystemVariable
	dbErr         error
	tableErr      error
	userErr       error
	openCalls     int
	listCalls     int
	tableCalls    int
	userCalls     int
	truncateCalls int
	renameCalls   int
}

func (c *databaseSelectionConnector) Open(context.Context, *config.ConnectionConfig) (*sql.DB, error) {
	c.openCalls++
	return sql.Open("mysql", "root@tcp(127.0.0.1:3306)/mysql")
}

func (c *databaseSelectionConnector) Diagnose(context.Context, *config.ConnectionConfig) (*driver.DiagnosticTrace, error) {
	return nil, nil
}

func (c *databaseSelectionConnector) Ping(context.Context, *config.ConnectionConfig, *sql.DB) error {
	return nil
}

func (c *databaseSelectionConnector) ListDatabases(context.Context, *config.ConnectionConfig, *sql.DB) ([]string, error) {
	c.listCalls++
	if c.dbErr != nil {
		return nil, c.dbErr
	}
	return append([]string(nil), c.databases...), nil
}

func (c *databaseSelectionConnector) ListTables(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	c.tableCalls++
	if c.tableErr != nil {
		return nil, c.tableErr
	}
	if len(c.tables) == 0 {
		return []string{"users", "orders"}, nil
	}
	return append([]string(nil), c.tables...), nil
}

func (c *databaseSelectionConnector) DescribeTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableColumn, error) {
	return []driver.TableColumn{{Name: "id", Type: "bigint"}}, nil
}

func (c *databaseSelectionConnector) ShowColumns(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.SchemaColumn, error) {
	if len(c.columns) == 0 {
		return []driver.SchemaColumn{
			{Name: "id", Type: "bigint unsigned", Nullable: false, Key: "PRI", Extra: "auto_increment"},
			{Name: "email", Type: "varchar(255)", Nullable: false, Key: "UNI"},
		}, nil
	}
	return append([]driver.SchemaColumn(nil), c.columns...), nil
}

func (c *databaseSelectionConnector) ShowIndexes(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableIndex, error) {
	if len(c.indexes) == 0 {
		return []driver.TableIndex{{Name: "PRIMARY", Column: "id", Type: "BTREE", SeqInIndex: 1}}, nil
	}
	return append([]driver.TableIndex(nil), c.indexes...), nil
}

func (c *databaseSelectionConnector) ShowForeignKeys(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.ForeignKey, error) {
	if len(c.foreignKeys) == 0 {
		return []driver.ForeignKey{{Constraint: "fk_members_org", Column: "organization_id", ReferencedTable: "organizations", ReferencedColumn: "id"}}, nil
	}
	return append([]driver.ForeignKey(nil), c.foreignKeys...), nil
}

func (c *databaseSelectionConnector) ShowCreateTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) (string, error) {
	if c.createDDL == "" {
		return "CREATE TABLE `users` (\n  `id` bigint NOT NULL\n)", nil
	}
	return c.createDDL, nil
}

func (c *databaseSelectionConnector) ShowTableStatus(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableStatus, error) {
	if len(c.statuses) == 0 {
		return []driver.TableStatus{{Name: "users", Engine: "InnoDB", Rows: 12813, DataLength: 44040192, IndexLength: 12582912, Collation: "utf8mb4_unicode_ci"}}, nil
	}
	return append([]driver.TableStatus(nil), c.statuses...), nil
}

func (c *databaseSelectionConnector) ShowTriggers(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]driver.Trigger, error) {
	if len(c.triggers) == 0 {
		return []driver.Trigger{{Name: "users_before_insert", Timing: "BEFORE", Event: "INSERT", Table: "users"}}, nil
	}
	return append([]driver.Trigger(nil), c.triggers...), nil
}

func (c *databaseSelectionConnector) ShowViews(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	if len(c.views) == 0 {
		return []string{"active_users", "monthly_reports"}, nil
	}
	return append([]string(nil), c.views...), nil
}

func (c *databaseSelectionConnector) ShowGrants(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]string, error) {
	return []string{"GRANT SELECT ON *.* TO 'analytics-ro'@'%'"}, nil
}

func (c *databaseSelectionConnector) ShowProcesslist(context.Context, *config.ConnectionConfig, *sql.DB) ([]driver.Process, error) {
	if len(c.processes) == 0 {
		return []driver.Process{{ID: 12, User: "app_user", Host: "10.0.0.2", Command: "Query", TimeSeconds: 2, Info: "SELECT 1"}}, nil
	}
	return append([]driver.Process(nil), c.processes...), nil
}

func (c *databaseSelectionConnector) ShowVariables(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]driver.SystemVariable, error) {
	if len(c.variables) == 0 {
		return []driver.SystemVariable{{Name: "max_connections", Value: "500"}}, nil
	}
	return append([]driver.SystemVariable(nil), c.variables...), nil
}

func (c *databaseSelectionConnector) TruncateTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) error {
	c.truncateCalls++
	return nil
}

func (c *databaseSelectionConnector) RenameTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string, string) error {
	c.renameCalls++
	return nil
}

func (c *databaseSelectionConnector) QueryStrings(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	c.userCalls++
	if c.userErr != nil {
		return nil, c.userErr
	}
	return append([]string(nil), c.users...), nil
}

func TestHandleLineUseParsesDatabase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	connector := &databaseSelectionConnector{databases: []string{"app_prod"}}
	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{
		ConfigDir: root,
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "use app_prod")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if app.session.Database != "app_prod" {
		t.Fatalf("database = %q", app.session.Database)
	}
}

func TestRestoreSessionDatabaseRestoresSelection(t *testing.T) {
	t.Parallel()

	connector := &databaseSelectionConnector{databases: []string{"app_prod", "app_demo"}}
	app := &Application{
		connector:         connector,
		session:           &Session{Connection: sampleConnection("prod"), DB: &sql.DB{}},
		reconnectDatabase: "app_prod",
	}

	if err := app.restoreSessionDatabase(context.Background()); err != nil {
		t.Fatalf("restoreSessionDatabase returned error: %v", err)
	}
	if app.session.Database != "app_prod" {
		t.Fatalf("database = %q", app.session.Database)
	}
}

func TestRestoreSessionDatabaseClearsStaleSelection(t *testing.T) {
	t.Parallel()

	connector := &databaseSelectionConnector{databases: []string{"app_demo"}}
	app := &Application{
		connector:         connector,
		session:           &Session{Connection: sampleConnection("prod"), DB: &sql.DB{}},
		reconnectDatabase: "app_prod",
	}

	err := app.restoreSessionDatabase(context.Background())
	if err == nil || !strings.Contains(err.Error(), "database \"app_prod\" no longer exists") {
		t.Fatalf("restoreSessionDatabase error = %v", err)
	}
	if app.session.Database != "" {
		t.Fatalf("database = %q, want cleared", app.session.Database)
	}
}

func TestPromptLabelFormatsDatabaseSelection(t *testing.T) {
	t.Parallel()

	app := &Application{session: &Session{}}
	if got := app.promptLabel(); got != "dbx> " {
		t.Fatalf("promptLabel() = %q", got)
	}

	app.session.Connection = sampleConnection("prod")
	if got := app.promptLabel(); got != "dbx(prod)> " {
		t.Fatalf("promptLabel() with connection = %q", got)
	}

	app.session.Database = "app_prod"
	if got := app.promptLabel(); got != "dbx(prod/app_prod)> " {
		t.Fatalf("promptLabel() with database = %q", got)
	}
}

func TestStatusIncludesDatabase(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("y\n"), &out, &out, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"

	if err := app.handleStatus(context.Background()); err != nil {
		t.Fatalf("handleStatus returned error: %v", err)
	}
	if !strings.Contains(out.String(), "Database: app_prod") {
		t.Fatalf("status output missing database: %q", out.String())
	}
}

func TestInvalidDatabaseSelectionFails(t *testing.T) {
	t.Parallel()

	connector := &databaseSelectionConnector{databases: []string{"app_demo"}}
	app := &Application{
		connector: connector,
		session:   &Session{Connection: sampleConnection("prod"), DB: &sql.DB{}},
		store:     config.NewStore(t.TempDir()),
	}

	err := app.handleUseDatabase(context.Background(), "app_prod")
	if err == nil || err.Error() != "Database not found: app_prod" {
		t.Fatalf("handleUseDatabase error = %v", err)
	}
}

func TestActivateConnectionClearsDatabaseSelection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	connector := &databaseSelectionConnector{}
	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{
		ConfigDir: root,
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("old")
	app.session.Database = "old_db"
	app.completionDBs = []string{"old_db"}
	app.completionDBsConn = "old"
	app.completionTables = []string{"users"}
	app.completionTablesConn = "old"
	app.completionTablesDB = "old_db"
	app.completionUsers = []string{"app_user"}
	app.completionUsersConn = "old"

	if err := app.activateConnection(context.Background(), sampleConnection("prod"), true); err != nil {
		t.Fatalf("activateConnection returned error: %v", err)
	}
	if app.session.Database != "" {
		t.Fatalf("database = %q, want cleared", app.session.Database)
	}
	if len(app.completionDBs) != 0 || app.completionDBsConn != "" {
		t.Fatalf("database completion cache not cleared: %+v", app)
	}
	if len(app.completionTables) != 0 || app.completionTablesConn != "" || app.completionTablesDB != "" {
		t.Fatalf("table completion cache not cleared: %+v", app)
	}
	if len(app.completionUsers) != 0 || app.completionUsersConn != "" {
		t.Fatalf("user completion cache not cleared: %+v", app)
	}

	sessionFile, err := app.store.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}
	if sessionFile.CurrentConnection != "prod" || sessionFile.CurrentDatabase != "" {
		t.Fatalf("unexpected session file: %+v", sessionFile)
	}
}

func TestCurrentCompletionTablesCachesByConnectionAndDatabase(t *testing.T) {
	t.Parallel()

	connector := &databaseSelectionConnector{tables: []string{"users", "orders"}}
	app := &Application{
		connector: connector,
		session: &Session{
			Connection: sampleConnection("prod"),
			Database:   "app_prod",
			DB:         &sql.DB{},
		},
	}

	first := app.currentCompletionTables()
	second := app.currentCompletionTables()
	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("unexpected tables: %#v %#v", first, second)
	}
	if connector.tableCalls != 1 {
		t.Fatalf("tableCalls = %d, want 1", connector.tableCalls)
	}
}

func TestCurrentCompletionTablesClearsOnDatabaseSelectionReset(t *testing.T) {
	t.Parallel()

	app := &Application{
		session:              &Session{Connection: sampleConnection("prod"), Database: "app_prod"},
		completionTables:     []string{"users"},
		completionTablesConn: "prod",
		completionTablesDB:   "app_prod",
	}

	app.clearDatabaseSelection()
	if len(app.completionTables) != 0 || app.completionTablesConn != "" || app.completionTablesDB != "" {
		t.Fatalf("table cache not cleared: %+v", app)
	}
}

func TestCompletionFailuresFallBackToEmptySuggestions(t *testing.T) {
	t.Parallel()

	app := &Application{
		connector: &databaseSelectionConnector{
			dbErr:    errors.New("db failure"),
			tableErr: errors.New("table failure"),
			userErr:  errors.New("user failure"),
		},
		session: &Session{
			Connection: sampleConnection("prod"),
			Database:   "app_prod",
			DB:         &sql.DB{},
		},
	}

	if got := app.currentCompletionDatabases(); got != nil {
		t.Fatalf("currentCompletionDatabases() = %#v, want nil", got)
	}
	if got := app.currentCompletionTables(); got != nil {
		t.Fatalf("currentCompletionTables() = %#v, want nil", got)
	}
	if got := app.currentCompletionUsers(); got != nil {
		t.Fatalf("currentCompletionUsers() = %#v, want nil", got)
	}
}
