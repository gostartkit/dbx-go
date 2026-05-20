package app

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
)

type readOnlyConnector struct {
	queryStrings []string
}

func (c *readOnlyConnector) Open(context.Context, *config.ConnectionConfig) (*sql.DB, error) {
	return nil, nil
}

func (c *readOnlyConnector) Diagnose(context.Context, *config.ConnectionConfig) (*driver.DiagnosticTrace, error) {
	return &driver.DiagnosticTrace{
		Steps: []driver.DiagnosticStep{
			{Name: "config", Status: "ok"},
			{Name: "mysql", Status: "ok"},
		},
	}, nil
}

func (c *readOnlyConnector) Ping(context.Context, *config.ConnectionConfig, *sql.DB) error {
	return nil
}

func (c *readOnlyConnector) ListDatabases(context.Context, *config.ConnectionConfig, *sql.DB) ([]string, error) {
	return []string{"app_prod", "analytics_v2"}, nil
}

func (c *readOnlyConnector) ListTables(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return []string{"users", "orders"}, nil
}

func (c *readOnlyConnector) DescribeTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableColumn, error) {
	return []driver.TableColumn{
		{Name: "id", Type: "bigint"},
		{Name: "email", Type: "varchar(255)"},
	}, nil
}

func (c *readOnlyConnector) ShowIndexes(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableIndex, error) {
	return []driver.TableIndex{{Name: "PRIMARY", Column: "id", Type: "BTREE", SeqInIndex: 1}}, nil
}

func (c *readOnlyConnector) ShowGrants(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]string, error) {
	return []string{"GRANT SELECT ON `app_prod`.* TO 'analytics-ro'@'%'"}, nil
}

func (c *readOnlyConnector) ShowProcesslist(context.Context, *config.ConnectionConfig, *sql.DB) ([]driver.Process, error) {
	return []driver.Process{{ID: 12, User: "app_user", Host: "10.0.0.2", Command: "Query", TimeSeconds: 2, Info: "SELECT 1"}}, nil
}

func (c *readOnlyConnector) ShowVariables(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]driver.SystemVariable, error) {
	return []driver.SystemVariable{{Name: "max_connections", Value: "500"}}, nil
}

func (c *readOnlyConnector) QueryStrings(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return append([]string(nil), c.queryStrings...), nil
}

func newReadOnlyTestApp(t *testing.T, root string, connector connectorClient) (*Application, *bytes.Buffer) {
	t.Helper()

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
	return app, &out
}

func TestReadOnlyCommandsDoNotAskConfirmation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		run     func(app *Application) error
		wantOut string
	}{
		{
			name: "show databases",
			run: func(app *Application) error {
				return app.handleShowDatabases(context.Background())
			},
			wantOut: "Databases:",
		},
		{
			name: "status",
			run: func(app *Application) error {
				return app.handleStatus(context.Background())
			},
			wantOut: "Connection: prod",
		},
		{
			name: "connection show",
			run: func(app *Application) error {
				return app.handleConnectionShow(context.Background(), "prod")
			},
			wantOut: "Name: prod",
		},
		{
			name: "connection test",
			run: func(app *Application) error {
				return app.handleConnectionTest(context.Background(), "prod", false)
			},
			wantOut: "[OK] mysql",
		},
		{
			name: "connection doctor",
			run: func(app *Application) error {
				return app.handleConnectionDoctor(context.Background(), "prod")
			},
			wantOut: "Connection doctor: prod",
		},
		{
			name: "show users",
			run: func(app *Application) error {
				return app.handleShowUsers(context.Background())
			},
			wantOut: "Users:",
		},
		{
			name: "show tables",
			run: func(app *Application) error {
				app.session.Database = "app_prod"
				return app.handleShowTables(context.Background())
			},
			wantOut: "Tables:",
		},
		{
			name: "describe",
			run: func(app *Application) error {
				app.session.Database = "app_prod"
				return app.handleDescribeTable(context.Background(), "users")
			},
			wantOut: "id",
		},
		{
			name: "show grants",
			run: func(app *Application) error {
				return app.handleShowGrants(context.Background(), "analytics-ro", "%")
			},
			wantOut: "GRANT SELECT",
		},
		{
			name: "show indexes",
			run: func(app *Application) error {
				app.session.Database = "app_prod"
				return app.handleShowIndexes(context.Background(), "users")
			},
			wantOut: "PRIMARY",
		},
		{
			name: "show processlist",
			run: func(app *Application) error {
				return app.handleShowProcesslist(context.Background())
			},
			wantOut: "app_user",
		},
		{
			name: "show variables",
			run: func(app *Application) error {
				return app.handleShowVariables(context.Background(), "max_connections")
			},
			wantOut: "max_connections",
		},
		{
			name: "context",
			run: func(app *Application) error {
				app.session.Database = "app_prod"
				return app.handleContext(context.Background())
			},
			wantOut: "Connection: prod",
		},
		{
			name: "connections",
			run: func(app *Application) error {
				return app.handleConnections(context.Background())
			},
			wantOut: "Configured connections:",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{queryStrings: []string{"app_prod", "analytics_v2"}})
			if err := tc.run(app); err != nil {
				t.Fatalf("command returned error: %v", err)
			}
			if strings.Contains(out.String(), "Confirm execution?") {
				t.Fatalf("unexpected confirmation prompt in output: %q", out.String())
			}
			if !strings.Contains(out.String(), tc.wantOut) {
				t.Fatalf("output missing %q: %q", tc.wantOut, out.String())
			}
		})
	}
}

func TestDropDatabaseDryRunDoesNotAskConfirmation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("analytics_v2\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{queryStrings: []string{"app_prod", "analytics_v2"}},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}
	app.dryRun = true

	if err := app.handleDropDatabase(context.Background()); err != nil {
		t.Fatalf("handleDropDatabase returned error: %v", err)
	}
	if strings.Contains(out.String(), "Confirm execution?") {
		t.Fatalf("unexpected confirmation prompt: %q", out.String())
	}
	if !strings.Contains(out.String(), "[DRY-RUN]") {
		t.Fatalf("expected dry-run output: %q", out.String())
	}
}

func TestCreateDatabaseConfirmsWhenNotDryRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("greenhn-dev\nutf8mb4\nutf8mb4_unicode_ci\nn\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}

	if err := app.handleCreateDatabase(context.Background()); err != nil {
		t.Fatalf("handleCreateDatabase returned error: %v", err)
	}
	if !strings.Contains(out.String(), "Confirm execution?") {
		t.Fatalf("expected confirmation prompt: %q", out.String())
	}
	if !strings.Contains(out.String(), "Cancelled.") {
		t.Fatalf("expected cancel path: %q", out.String())
	}
}

func TestCreateDatabaseClearsDatabaseCompletionCache(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("greenhn-dev\nutf8mb4\nutf8mb4_unicode_ci\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}
	app.dryRun = true
	app.completionDBs = []string{"app_prod", "analytics_v2"}
	app.completionDBsConn = "prod"

	if err := app.handleCreateDatabase(context.Background()); err != nil {
		t.Fatalf("handleCreateDatabase returned error: %v", err)
	}
	if len(app.completionDBs) != 0 || app.completionDBsConn != "" {
		t.Fatalf("database completion cache not cleared: %+v", app)
	}
}

func TestDropDatabaseClearsDatabaseCompletionCache(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("analytics_v2\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{queryStrings: []string{"app_prod", "analytics_v2"}},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}
	app.dryRun = true
	app.completionDBs = []string{"app_prod", "analytics_v2"}
	app.completionDBsConn = "prod"

	if err := app.handleDropDatabase(context.Background()); err != nil {
		t.Fatalf("handleDropDatabase returned error: %v", err)
	}
	if len(app.completionDBs) != 0 || app.completionDBsConn != "" {
		t.Fatalf("database completion cache not cleared: %+v", app)
	}
}

func TestCreateUserConfirmsWhenNotDryRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("\n3\nn\nn\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}

	if err := app.handleCreateUser(context.Background(), "analytics-ro"); err != nil {
		t.Fatalf("handleCreateUser returned error: %v", err)
	}
	if !strings.Contains(out.String(), "Confirm execution?") {
		t.Fatalf("expected confirmation prompt: %q", out.String())
	}
	if !strings.Contains(out.String(), "Cancelled.") {
		t.Fatalf("expected cancel path: %q", out.String())
	}
}

func TestDropUserDryRunDoesNotAskConfirmation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}
	app.dryRun = true

	if err := app.handleDropUser(context.Background(), "analytics-ro"); err != nil {
		t.Fatalf("handleDropUser returned error: %v", err)
	}
	if strings.Contains(out.String(), "Confirm execution?") {
		t.Fatalf("unexpected confirmation prompt: %q", out.String())
	}
	if !strings.Contains(out.String(), "[DRY-RUN]") {
		t.Fatalf("expected dry-run output: %q", out.String())
	}
}
