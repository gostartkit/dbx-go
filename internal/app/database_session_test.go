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

type databaseSelectionConnector struct {
	databases []string
	openCalls int
	listCalls int
}

func (c *databaseSelectionConnector) Open(context.Context, *config.ConnectionConfig) (*sql.DB, error) {
	c.openCalls++
	return nil, nil
}

func (c *databaseSelectionConnector) Diagnose(context.Context, *config.ConnectionConfig) (*driver.DiagnosticTrace, error) {
	return nil, nil
}

func (c *databaseSelectionConnector) Ping(context.Context, *config.ConnectionConfig, *sql.DB) error {
	return nil
}

func (c *databaseSelectionConnector) ListDatabases(context.Context, *config.ConnectionConfig, *sql.DB) ([]string, error) {
	c.listCalls++
	return append([]string(nil), c.databases...), nil
}

func (c *databaseSelectionConnector) QueryStrings(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return nil, nil
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

	if err := app.activateConnection(context.Background(), sampleConnection("prod"), true); err != nil {
		t.Fatalf("activateConnection returned error: %v", err)
	}
	if app.session.Database != "" {
		t.Fatalf("database = %q, want cleared", app.session.Database)
	}

	sessionFile, err := app.store.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}
	if sessionFile.CurrentConnection != "prod" || sessionFile.CurrentDatabase != "" {
		t.Fatalf("unexpected session file: %+v", sessionFile)
	}
}
