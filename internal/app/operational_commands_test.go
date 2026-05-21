package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestHandleLineShowTablesParsesCommand(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{tables: []string{"users", "orders"}},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show tables")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "users") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowContextParsesCommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"

	exit, err := app.handleLine(context.Background(), "show context")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "Connection: prod") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowUsersParsesCommand(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{queryStrings: []string{"root@localhost", "analytics_ro@%"}},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "analytics_ro@%") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowUserParsesCommand(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{queryStrings: []string{"analytics_ro@%", "analytics_ro@localhost"}},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show user analytics_ro")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "User analytics_ro:") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestCLIShowTablesJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{databases: []string{"app_prod"}, tables: []string{"users", "orders"}},
	})
	if err := app.Run(context.Background(), []string{"show", "tables", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result TablesResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Database != "app_prod" || len(result.Tables) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIShowContextJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{databases: []string{"app_prod"}},
	})
	if err := app.Run(context.Background(), []string{"show", "context", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result ContextResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Connection != "prod" || result.Database != "app_prod" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIShowUsersJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{queryStrings: []string{"root@localhost", "analytics_ro@%"}},
	})
	if err := app.Run(context.Background(), []string{"show", "users", "--connection", "prod", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result UsersResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Connection != "prod" || len(result.Users) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIShowUserJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{queryStrings: []string{"analytics_ro@%", "analytics_ro@localhost"}},
	})
	if err := app.Run(context.Background(), []string{"show", "user", "analytics_ro", "--connection", "prod", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result UsersResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Connection != "prod" || result.User != "analytics_ro" || len(result.Users) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
