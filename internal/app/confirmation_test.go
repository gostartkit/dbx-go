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

func (c *readOnlyConnector) Ping(context.Context, *config.ConnectionConfig, *sql.DB) error {
	return nil
}

func (c *readOnlyConnector) ListDatabases(context.Context, *config.ConnectionConfig, *sql.DB) ([]string, error) {
	return []string{"app_prod", "analytics_v2"}, nil
}

func (c *readOnlyConnector) ListTables(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return []string{"users", "orders"}, nil
}

func (c *readOnlyConnector) ShowColumns(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.SchemaColumn, error) {
	return []driver.SchemaColumn{{Name: "id", Type: "bigint unsigned", Nullable: false, Key: "PRI", Extra: "auto_increment"}}, nil
}

func (c *readOnlyConnector) PeekRows(context.Context, *config.ConnectionConfig, *sql.DB, string, string, int) (*driver.RowSet, error) {
	return &driver.RowSet{Columns: []string{"id", "email"}, Rows: [][]any{{1, "a@example.com"}}}, nil
}

func (c *readOnlyConnector) SampleRows(context.Context, *config.ConnectionConfig, *sql.DB, string, string, int) (*driver.RowSet, error) {
	return &driver.RowSet{Columns: []string{"id", "email"}, Rows: [][]any{{2, "b@example.com"}}}, nil
}

func (c *readOnlyConnector) ShowCreateTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) (string, error) {
	return "CREATE TABLE `users` (\n  `id` bigint NOT NULL\n)", nil
}

func (c *readOnlyConnector) ShowTableStatus(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableStatus, error) {
	return []driver.TableStatus{{Name: "users", Engine: "InnoDB", Rows: 12813, DataLength: 44040192, IndexLength: 12582912, Collation: "utf8mb4_unicode_ci"}}, nil
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
			name: "connection show",
			run: func(app *Application) error {
				return app.handleConnectionShow(context.Background(), "prod")
			},
			wantOut: "Name: prod",
		},
		{
			name: "connection doctor",
			run: func(app *Application) error {
				return app.handleConnectionDoctor(context.Background(), "prod")
			},
			wantOut: "Connection doctor: prod",
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
			name: "show users",
			run: func(app *Application) error {
				return app.handleShowUsers(context.Background())
			},
			wantOut: "Users:",
		},
		{
			name: "show columns",
			run: func(app *Application) error {
				app.session.Database = "app_prod"
				return app.handleShowColumns(context.Background(), "users")
			},
			wantOut: "auto_increment",
		},
		{
			name: "show create table",
			run: func(app *Application) error {
				app.session.Database = "app_prod"
				return app.handleShowCreateTable(context.Background(), "users")
			},
			wantOut: "CREATE TABLE `users`",
		},
		{
			name: "show table status",
			run: func(app *Application) error {
				app.session.Database = "app_prod"
				return app.handleShowTableStatus(context.Background(), "users")
			},
			wantOut: "Name: users",
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

			app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{queryStrings: []string{"app_prod", "analytics_v2", "analytics_ro@%", "analytics_ro@localhost"}})
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
