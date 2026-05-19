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
			name: "list databases",
			run: func(app *Application) error {
				return app.handleListDatabases(context.Background())
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
