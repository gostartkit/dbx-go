package app

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestHandleLineCreateUserParsesName(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader("\n3\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}
	app.dryRun = true

	exit, err := app.handleLine(context.Background(), "create user analytics-ro")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "[DRY-RUN] Create MySQL user 'analytics-ro'@'%'") {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if strings.Contains(out.String(), "Confirm execution?") {
		t.Fatalf("dry-run should not ask confirmation: %q", out.String())
	}
}

func TestHandleLineDropUserParsesName(t *testing.T) {
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

	exit, err := app.handleLine(context.Background(), "drop user analytics-ro")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "[DRY-RUN] Drop MySQL user 'analytics-ro'@'%'") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestCreateUserUsesCurrentDatabaseForGrant(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader("\n3\ny\n2\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}
	app.dryRun = true

	if err := app.handleCreateUser(context.Background(), "analytics-ro"); err != nil {
		t.Fatalf("handleCreateUser returned error: %v", err)
	}
	if !strings.Contains(out.String(), "GRANT SELECT ON `app_prod`.* TO 'analytics-ro'@'%'") {
		t.Fatalf("missing grant SQL in preview: %q", out.String())
	}
}

func TestCreateUserClearsUserCompletionCache(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader("\n3\nn\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}
	app.dryRun = true
	app.completionUsers = []string{"analytics-ro", "app_user"}
	app.completionUsersConn = "prod"

	if err := app.handleCreateUser(context.Background(), "analytics-ro"); err != nil {
		t.Fatalf("handleCreateUser returned error: %v", err)
	}
	if len(app.completionUsers) != 0 || app.completionUsersConn != "" {
		t.Fatalf("user completion cache not cleared: %+v", app)
	}
}

func TestDropUserClearsUserCompletionCache(t *testing.T) {
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
	app.completionUsers = []string{"analytics-ro", "app_user"}
	app.completionUsersConn = "prod"

	if err := app.handleDropUser(context.Background(), "analytics-ro"); err != nil {
		t.Fatalf("handleDropUser returned error: %v", err)
	}
	if len(app.completionUsers) != 0 || app.completionUsersConn != "" {
		t.Fatalf("user completion cache not cleared: %+v", app)
	}
}
