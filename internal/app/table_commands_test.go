package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
)

func TestHandleLineShowCreateTableParsesCommand(t *testing.T) {
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
		Connector: &databaseSelectionConnector{createDDL: "CREATE TABLE `users` (\n  `id` bigint NOT NULL\n)"},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show create table users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "CREATE TABLE `users`") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowTableStatusParsesCommands(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &databaseSelectionConnector{
		statuses: []driver.TableStatus{
			{Name: "users", Engine: "InnoDB", Rows: 12813, DataLength: 44040192, IndexLength: 12582912, Collation: "utf8mb4_unicode_ci"},
			{Name: "orders", Engine: "InnoDB", Rows: 99231, DataLength: 125829120, IndexLength: 42991616, Collation: "utf8mb4_unicode_ci"},
		},
	}
	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: root, Connector: connector})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show table status")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "users") || !strings.Contains(out.String(), "orders") {
		t.Fatalf("unexpected output: %q", out.String())
	}

	out.Reset()
	connector.statuses = []driver.TableStatus{{Name: "users", Engine: "InnoDB", Rows: 12813, DataLength: 44040192, IndexLength: 12582912, Collation: "utf8mb4_unicode_ci"}}
	exit, err = app.handleLine(context.Background(), "show table status users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "Name: users") || !strings.Contains(out.String(), "Data Size: 42MB") {
		t.Fatalf("unexpected detail output: %q", out.String())
	}
}

func TestHandleLineTruncateTableParsesCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &databaseSelectionConnector{}
	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("users\n"), &out, &out, Options{ConfigDir: root, Connector: connector})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "truncate table users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if connector.truncateCalls != 1 || !strings.Contains(out.String(), "Type the table name to confirm") {
		t.Fatalf("unexpected truncate behavior: calls=%d output=%q", connector.truncateCalls, out.String())
	}
}

func TestHandleLineRenameTableParsesCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &databaseSelectionConnector{}
	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("y\n"), &out, &out, Options{ConfigDir: root, Connector: connector})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "rename table users_tmp users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if connector.renameCalls != 1 || !strings.Contains(out.String(), "[OK] Rename table users_tmp -> users") {
		t.Fatalf("unexpected rename behavior: calls=%d output=%q", connector.renameCalls, out.String())
	}
}

func TestTableCommandsRequireDatabaseContext(t *testing.T) {
	t.Parallel()

	app := &Application{
		connector: &databaseSelectionConnector{},
		session:   &Session{Connection: sampleConnection("prod"), DB: &sql.DB{}},
		store:     config.NewStore(t.TempDir()),
	}

	cases := []struct {
		name string
		run  func() error
	}{
		{name: "show create table", run: func() error { return app.handleShowCreateTable(context.Background(), "users") }},
		{name: "show table status", run: func() error { return app.handleShowTableStatus(context.Background(), "") }},
		{name: "truncate table", run: func() error { return app.handleTruncateTable(context.Background(), "users") }},
		{name: "rename table", run: func() error { return app.handleRenameTable(context.Background(), "users_tmp", "users") }},
	}

	for _, tc := range cases {
		if err := tc.run(); err == nil || !strings.Contains(err.Error(), "no database selected; use: use <database>") {
			t.Fatalf("%s error = %v", tc.name, err)
		}
	}
}

func TestTableCommandRejectsInvalidTableName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root, Connector: &databaseSelectionConnector{}})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	if err := app.handleShowCreateTable(context.Background(), "foo bar"); err == nil || !strings.Contains(err.Error(), "invalid table name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowCreateTableDryRunQuotesHyphenatedName(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: root, Connector: &databaseSelectionConnector{}})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}
	app.dryRun = true

	if err := app.handleShowCreateTable(context.Background(), "tmp-users"); err != nil {
		t.Fatalf("handleShowCreateTable returned error: %v", err)
	}
	if !strings.Contains(out.String(), "SHOW CREATE TABLE `tmp-users`") {
		t.Fatalf("expected quoted table name: %q", out.String())
	}
}

func TestTruncateTableRequiresTypedConfirmation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &databaseSelectionConnector{}
	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("wrong\n"), &out, &out, Options{ConfigDir: root, Connector: connector})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	if err := app.handleTruncateTable(context.Background(), "users"); err != nil {
		t.Fatalf("handleTruncateTable returned error: %v", err)
	}
	if connector.truncateCalls != 0 || !strings.Contains(out.String(), "Cancelled.") {
		t.Fatalf("unexpected truncate confirmation behavior: calls=%d output=%q", connector.truncateCalls, out.String())
	}
}

func TestTruncateAndRenameClearTableCompletion(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &databaseSelectionConnector{}
	app, err := NewWithOptions(strings.NewReader("users\ny\n"), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root, Connector: connector})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}
	app.completionTables = []string{"users", "users_tmp"}
	app.completionTablesConn = "prod"
	app.completionTablesDB = "app_prod"

	if err := app.handleTruncateTable(context.Background(), "users"); err != nil {
		t.Fatalf("handleTruncateTable returned error: %v", err)
	}
	if len(app.completionTables) != 0 {
		t.Fatalf("expected truncate to clear table completion: %+v", app.completionTables)
	}

	app.completionTables = []string{"users", "users_tmp"}
	app.completionTablesConn = "prod"
	app.completionTablesDB = "app_prod"
	if err := app.handleRenameTable(context.Background(), "users_tmp", "users"); err != nil {
		t.Fatalf("handleRenameTable returned error: %v", err)
	}
	if len(app.completionTables) != 0 {
		t.Fatalf("expected rename to clear table completion: %+v", app.completionTables)
	}
}

func TestCLIShowCreateTableJSON(t *testing.T) {
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
		Connector: &databaseSelectionConnector{databases: []string{"app_prod"}, createDDL: "CREATE TABLE `users` (\n  `id` bigint NOT NULL\n)"},
	})
	err := app.Run(context.Background(), []string{"show", "create", "table", "users", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result CreateTableResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Table != "users" || !strings.Contains(result.CreateTable, "CREATE TABLE `users`") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIShowTableStatusJSON(t *testing.T) {
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
		Connector: &databaseSelectionConnector{
			databases: []string{"app_prod"},
			statuses:  []driver.TableStatus{{Name: "users", Engine: "InnoDB", Rows: 12813, DataLength: 44040192, IndexLength: 12582912, Collation: "utf8mb4_unicode_ci"}},
		},
	})
	err := app.Run(context.Background(), []string{"show", "table", "status", "users", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result TableStatusResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(result.Tables) != 1 || result.Tables[0].Name != "users" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLITruncateTableJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &databaseSelectionConnector{databases: []string{"app_prod"}}
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root, Connector: connector})
	err := app.Run(context.Background(), []string{"truncate", "table", "auth_sessions", "--connection", "prod", "--database", "app_prod", "--yes", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if connector.truncateCalls != 1 {
		t.Fatalf("truncate calls = %d", connector.truncateCalls)
	}

	var result TableMutationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Action != "truncate" || result.Table != "auth_sessions" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIRenameTableJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &databaseSelectionConnector{databases: []string{"app_prod"}}
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root, Connector: connector})
	err := app.Run(context.Background(), []string{"rename", "table", "users_tmp", "users", "--connection", "prod", "--database", "app_prod", "--yes", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if connector.renameCalls != 1 {
		t.Fatalf("rename calls = %d", connector.renameCalls)
	}

	var result TableMutationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Action != "rename" || result.From != "users_tmp" || result.To != "users" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLITableMutationsRequireYesUnlessDryRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	truncateApp, _, truncateErr := newCLIAppWithOptions(t, "", Options{ConfigDir: root, Connector: &databaseSelectionConnector{databases: []string{"app_prod"}}})
	err := truncateApp.Run(context.Background(), []string{"truncate", "table", "users", "--connection", "prod", "--database", "app_prod", "--config-dir", root})
	if err == nil || !strings.Contains(err.Error(), "confirmation required") {
		t.Fatalf("expected truncate confirmation error, got %v", err)
	}
	if truncateErr.String() != "" {
		t.Fatalf("unexpected stderr: %q", truncateErr.String())
	}

	renameApp, _, _ := newCLIAppWithOptions(t, "", Options{ConfigDir: root, Connector: &databaseSelectionConnector{databases: []string{"app_prod"}}})
	err = renameApp.Run(context.Background(), []string{"rename", "table", "users_tmp", "users", "--connection", "prod", "--database", "app_prod", "--config-dir", root})
	if err == nil || !strings.Contains(err.Error(), "confirmation required") {
		t.Fatalf("expected rename confirmation error, got %v", err)
	}
}

func TestCLITableMutationsDryRunDoNotExecute(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	truncateConnector := &databaseSelectionConnector{databases: []string{"app_prod"}}
	truncateApp, truncateOut, truncateErr := newCLIAppWithOptions(t, "", Options{ConfigDir: root, Connector: truncateConnector})
	err := truncateApp.Run(context.Background(), []string{"truncate", "table", "tmp-users", "--connection", "prod", "--database", "app_prod", "--dry-run", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("truncate dry-run returned error: %v\nstderr=%s", err, truncateErr.String())
	}
	if truncateConnector.truncateCalls != 0 || !strings.Contains(truncateOut.String(), "TRUNCATE TABLE `tmp-users`") {
		t.Fatalf("unexpected truncate dry-run output: calls=%d output=%q", truncateConnector.truncateCalls, truncateOut.String())
	}

	renameConnector := &databaseSelectionConnector{databases: []string{"app_prod"}}
	renameApp, renameOut, renameErr := newCLIAppWithOptions(t, "", Options{ConfigDir: root, Connector: renameConnector})
	err = renameApp.Run(context.Background(), []string{"rename", "table", "tmp-users", "users", "--connection", "prod", "--database", "app_prod", "--dry-run", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("rename dry-run returned error: %v\nstderr=%s", err, renameErr.String())
	}
	if renameConnector.renameCalls != 0 || !strings.Contains(renameOut.String(), "RENAME TABLE `tmp-users` TO `users`") {
		t.Fatalf("unexpected rename dry-run output: calls=%d output=%q", renameConnector.renameCalls, renameOut.String())
	}
}

func TestCLITableCommandsRequireDatabaseContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, _, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{},
	})
	err := app.Run(context.Background(), []string{"show", "create", "table", "users", "--connection", "prod", "--config-dir", root})
	if err == nil || !strings.Contains(err.Error(), "no database selected; use --database <name>") {
		t.Fatalf("expected missing database error, got %v\nstderr=%s", err, stderr.String())
	}
}

func TestHelpIncludesNewTableCommands(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := printHelpTopic(simplePrinter{writer: &out}, ""); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	joined := out.String()
	if !strings.Contains(joined, "show create table") || !strings.Contains(joined, "show table status") || !strings.Contains(joined, "truncate table") || !strings.Contains(joined, "rename table") {
		t.Fatalf("unexpected root help output: %q", joined)
	}

	out.Reset()
	if err := printHelpTopic(simplePrinter{writer: &out}, "truncate table"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	joined = out.String()
	if !strings.Contains(joined, "truncate table") || !strings.Contains(joined, "--yes in the CLI") {
		t.Fatalf("unexpected help output: %q", joined)
	}
}
