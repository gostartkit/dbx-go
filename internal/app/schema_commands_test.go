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

func TestHandleLineShowColumnsParsesCommand(t *testing.T) {
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
		Connector: &databaseSelectionConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show columns users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "auto_increment") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestCLIShowColumnsJSON(t *testing.T) {
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
	if err := app.Run(context.Background(), []string{"show", "columns", "users", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result ColumnsResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Table != "users" || len(result.Columns) == 0 || result.Columns[0].Name != "id" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
