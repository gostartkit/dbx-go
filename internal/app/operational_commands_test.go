package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
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
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "users") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowDatabasesParsesCanonicalCommand(t *testing.T) {
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
		Connector: &readOnlyConnector{queryStrings: []string{"app_prod", "app_demo"}},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show databases")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "app_prod") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineListDatabasesParsesAlias(t *testing.T) {
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
		Connector: &readOnlyConnector{queryStrings: []string{"app_prod", "app_demo"}},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "list databases")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "app_prod") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineDescribeParsesAlias(t *testing.T) {
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

	exit, err := app.handleLine(context.Background(), "describe users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "id") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowGrantsParsesCommand(t *testing.T) {
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
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show grants analytics-ro")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "GRANT SELECT") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowIndexesParsesCommand(t *testing.T) {
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
		Connector: &databaseSelectionConnector{
			indexes: []driver.TableIndex{
				{Name: "idx_email", Column: "email", Type: "BTREE", SeqInIndex: 1},
				{Name: "PRIMARY", Column: "id", Type: "BTREE", SeqInIndex: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show indexes users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "PRIMARY") || !strings.Contains(out.String(), "idx_email") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowIndexesOnParsesCommand(t *testing.T) {
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

	exit, err := app.handleLine(context.Background(), "show indexes on users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "PRIMARY") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowProcesslistParsesCommand(t *testing.T) {
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
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show processlist")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "app_user") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineShowVariablesParsesCommand(t *testing.T) {
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
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "show variables max_connections")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "max_connections") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineContextAlias(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"

	exit, err := app.handleLine(context.Background(), "ctx")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if !strings.Contains(out.String(), "Connection: prod") || !strings.Contains(out.String(), "Database: app_prod") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestShowTablesRequiresDatabaseContext(t *testing.T) {
	t.Parallel()

	app := &Application{
		connector: &databaseSelectionConnector{},
		session:   &Session{Connection: sampleConnection("prod"), DB: &sql.DB{}},
		store:     config.NewStore(t.TempDir()),
	}

	err := app.handleShowTables(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no database selected; use: use <database>") {
		t.Fatalf("handleShowTables error = %v", err)
	}
}

func TestShowIndexesRequiresDatabaseContext(t *testing.T) {
	t.Parallel()

	app := &Application{
		connector: &databaseSelectionConnector{},
		session:   &Session{Connection: sampleConnection("prod"), DB: &sql.DB{}},
		store:     config.NewStore(t.TempDir()),
	}

	err := app.handleShowIndexes(context.Background(), "users")
	if err == nil || !strings.Contains(err.Error(), "no database selected; use: use <database>") {
		t.Fatalf("handleShowIndexes error = %v", err)
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
	err := app.Run(context.Background(), []string{"show", "tables", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root})
	if err != nil {
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

func TestCLIDescribeJSON(t *testing.T) {
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
	err := app.Run(context.Background(), []string{"describe", "users", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result TableDescriptionResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Table != "users" || len(result.Columns) == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIShowGrantsJSON(t *testing.T) {
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
		Connector: &databaseSelectionConnector{},
	})
	err := app.Run(context.Background(), []string{"show", "grants", "analytics-ro", "--connection", "prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result GrantsResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.User != "analytics-ro" || result.Host != "%" || len(result.Grants) == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIShowIndexesJSON(t *testing.T) {
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
			indexes: []driver.TableIndex{
				{Name: "idx_email", Column: "email", Type: "BTREE", SeqInIndex: 1},
				{Name: "PRIMARY", Column: "id", Type: "BTREE", SeqInIndex: 1},
			},
		},
	})
	err := app.Run(context.Background(), []string{"show", "indexes", "users", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result TableIndexesResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Table != "users" || len(result.Indexes) != 2 || result.Indexes[0].Name != "PRIMARY" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIShowProcesslistJSON(t *testing.T) {
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
			processes: []driver.Process{
				{ID: 18, User: "root", Host: "localhost", Command: "Sleep", TimeSeconds: 12},
				{ID: 12, User: "app_user", Host: "10.0.0.2", Command: "Query", TimeSeconds: 2, Info: "SELECT 1"},
			},
		},
	})
	err := app.Run(context.Background(), []string{"show", "processlist", "--connection", "prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result ProcesslistResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(result.Processes) != 2 || result.Processes[0].ID != 12 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCLIShowVariablesJSON(t *testing.T) {
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
			variables: []driver.SystemVariable{
				{Name: "wait_timeout", Value: "28800"},
				{Name: "max_connections", Value: "500"},
			},
		},
	})
	err := app.Run(context.Background(), []string{"show", "variables", "innodb%", "--connection", "prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result VariablesResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Pattern != "innodb%" || len(result.Variables) != 2 || result.Variables[0].Name != "max_connections" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestShowProcesslistTruncatesLongQuery(t *testing.T) {
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
		Connector: &databaseSelectionConnector{
			processes: []driver.Process{{
				ID:          12,
				User:        "app_user",
				Host:        "10.0.0.2",
				Command:     "Query",
				TimeSeconds: 2,
				Info:        strings.Repeat("x", processInfoPreviewLimit+10),
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.DB = &sql.DB{}

	if err := app.handleShowProcesslist(context.Background()); err != nil {
		t.Fatalf("handleShowProcesslist returned error: %v", err)
	}
	if !strings.Contains(out.String(), "...") {
		t.Fatalf("expected truncated query output: %q", out.String())
	}
}

func TestShowIndexesOutputOrderingDeterministic(t *testing.T) {
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
		Connector: &databaseSelectionConnector{
			indexes: []driver.TableIndex{
				{Name: "idx_org", Column: "organization_id", Type: "BTREE", SeqInIndex: 1},
				{Name: "PRIMARY", Column: "id", Type: "BTREE", SeqInIndex: 1},
				{Name: "idx_email", Column: "email", Type: "BTREE", SeqInIndex: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	if err := app.handleShowIndexes(context.Background(), "users"); err != nil {
		t.Fatalf("handleShowIndexes returned error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if !strings.Contains(lines[0], "PRIMARY") || !strings.Contains(lines[1], "idx_email") || !strings.Contains(lines[2], "idx_org") {
		t.Fatalf("unexpected ordering: %q", out.String())
	}
}

func TestCLIContextJSON(t *testing.T) {
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
	err := app.Run(context.Background(), []string{"context", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root})
	if err != nil {
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

func TestHelpIncludesOperationalCommands(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := printHelpTopic(simplePrinter{writer: &out}, "show indexes"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	if !strings.Contains(out.String(), "show indexes") {
		t.Fatalf("unexpected help output: %q", out.String())
	}
}

func TestHelpListDatabasesShowsCanonicalTopic(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := printHelpTopic(simplePrinter{writer: &out}, "list databases"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	if !strings.Contains(out.String(), "show databases") {
		t.Fatalf("expected canonical help topic: %q", out.String())
	}
	if strings.Contains(out.String(), "title: list databases") {
		t.Fatalf("unexpected legacy help title: %q", out.String())
	}
}

type simplePrinter struct {
	writer *bytes.Buffer
}

func (p simplePrinter) Println(args ...any) {
	_, _ = fmt.Fprintln(p.writer, args...)
}
