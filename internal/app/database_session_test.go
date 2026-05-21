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
	databases    []string
	tables       []string
	columns      []driver.SchemaColumn
	rowSet       *driver.RowSet
	createDDL    string
	statuses     []driver.TableStatus
	queryStrings []string
	dbErr        error
	tableErr     error
	userErr      error
	openCalls    int
	listCalls    int
	tableCalls   int
	peekLimit    int
	sampleLimit  int
}

func (c *databaseSelectionConnector) Open(context.Context, *config.ConnectionConfig) (*sql.DB, error) {
	c.openCalls++
	return sql.Open("mysql", "root@tcp(127.0.0.1:3306)/mysql")
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

func (c *databaseSelectionConnector) ShowColumns(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.SchemaColumn, error) {
	if len(c.columns) == 0 {
		return []driver.SchemaColumn{
			{Name: "id", Type: "bigint unsigned", Nullable: false, Key: "PRI", Extra: "auto_increment"},
			{Name: "email", Type: "varchar(255)", Nullable: false, Key: "UNI"},
		}, nil
	}
	return append([]driver.SchemaColumn(nil), c.columns...), nil
}

func (c *databaseSelectionConnector) PeekRows(_ context.Context, _ *config.ConnectionConfig, _ *sql.DB, _ string, _ string, limit int) (*driver.RowSet, error) {
	c.peekLimit = limit
	if c.rowSet != nil {
		return cloneRowSet(c.rowSet), nil
	}
	return &driver.RowSet{
		Columns: []string{"id", "email", "created_at"},
		Rows: [][]any{
			{1, "a@example.com", "2026-01-01 12:00:00"},
			{2, "b@example.com", "2026-01-02 12:00:00"},
		},
	}, nil
}

func (c *databaseSelectionConnector) SampleRows(_ context.Context, _ *config.ConnectionConfig, _ *sql.DB, _ string, _ string, limit int) (*driver.RowSet, error) {
	c.sampleLimit = limit
	if c.rowSet != nil {
		return cloneRowSet(c.rowSet), nil
	}
	return &driver.RowSet{
		Columns: []string{"id", "email", "created_at"},
		Rows: [][]any{
			{2, "b@example.com", "2026-01-02 12:00:00"},
		},
	}, nil
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

func (c *databaseSelectionConnector) QueryStrings(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	if c.userErr != nil {
		return nil, c.userErr
	}
	return append([]string(nil), c.queryStrings...), nil
}

func cloneRowSet(value *driver.RowSet) *driver.RowSet {
	if value == nil {
		return nil
	}
	result := &driver.RowSet{
		Columns: append([]string(nil), value.Columns...),
		Rows:    make([][]any, 0, len(value.Rows)),
	}
	for _, row := range value.Rows {
		result.Rows = append(result.Rows, append([]any(nil), row...))
	}
	return result
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

	exit, err := app.handleLine(context.Background(), "use database app_prod")
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

	app.dryRun = true
	if got := app.promptLabel(); got != "dbx[dry-run]> " {
		t.Fatalf("promptLabel() with dry-run = %q", got)
	}

	app.dryRun = false
	app.session.Connection = sampleConnection("prod")
	if got := app.promptLabel(); got != "dbx[prod][disconnected]> " {
		t.Fatalf("promptLabel() with connection = %q", got)
	}

	app.session.DB = &sql.DB{}
	app.session.Database = "app_prod"
	if got := app.promptLabel(); got != "dbx[prod/app_prod]> " {
		t.Fatalf("promptLabel() with database = %q", got)
	}

	app.session.DB = nil
	app.dryRun = true
	if got := app.promptLabel(); got != "dbx[prod/app_prod][disconnected][dry-run]> " {
		t.Fatalf("promptLabel() with disconnected dry-run = %q", got)
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
