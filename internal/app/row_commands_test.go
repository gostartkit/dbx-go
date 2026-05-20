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

func TestHandleLineCountRowsParsesCommand(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: root, Connector: &databaseSelectionConnector{rowCount: 12813}})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "count rows users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "users: 12813 rows") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineCountAliasParsesCommand(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: root, Connector: &databaseSelectionConnector{rowCount: 12813}})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	exit, err := app.handleLine(context.Background(), "count users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "12813 rows") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLinePeekRowsParsesCommand(t *testing.T) {
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

	exit, err := app.handleLine(context.Background(), "peek rows users 20")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "a@example.com") || !strings.Contains(out.String(), "created_at") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLinePeekAliasParsesCommand(t *testing.T) {
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

	exit, err := app.handleLine(context.Background(), "peek users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "a@example.com") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineSampleRowsParsesCommand(t *testing.T) {
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

	exit, err := app.handleLine(context.Background(), "sample rows users 10")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "b@example.com") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHandleLineSampleAliasParsesCommand(t *testing.T) {
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

	exit, err := app.handleLine(context.Background(), "sample users")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit || !strings.Contains(out.String(), "b@example.com") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRowInspectionCommandsRequireDatabaseContext(t *testing.T) {
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
		{name: "count rows", run: func() error { return app.handleCountRows(context.Background(), "users") }},
		{name: "peek rows", run: func() error { return app.handlePeekRows(context.Background(), "users", "") }},
		{name: "sample rows", run: func() error { return app.handleSampleRows(context.Background(), "users", "") }},
	}

	for _, tc := range cases {
		if err := tc.run(); err == nil || !strings.Contains(err.Error(), "no database selected; use: use <database>") {
			t.Fatalf("%s error = %v", tc.name, err)
		}
	}
}

func TestRowInspectionRejectsInvalidTableName(t *testing.T) {
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

	if err := app.handleCountRows(context.Background(), "users;drop"); err == nil || !strings.Contains(err.Error(), "invalid table name") {
		t.Fatalf("unexpected count rows error: %v", err)
	}
}

func TestRowInspectionLimitDefaultAndMax(t *testing.T) {
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
	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root, Connector: connector})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}

	if err := app.handlePeekRows(context.Background(), "users", ""); err != nil {
		t.Fatalf("handlePeekRows returned error: %v", err)
	}
	if connector.peekLimit != defaultRowInspectionLimit {
		t.Fatalf("peek default limit = %d", connector.peekLimit)
	}

	if err := app.handleSampleRows(context.Background(), "users", "500"); err != nil {
		t.Fatalf("handleSampleRows returned error: %v", err)
	}
	if connector.sampleLimit != maxRowInspectionLimit {
		t.Fatalf("sample max limit = %d", connector.sampleLimit)
	}
}

func TestRowInspectionRejectsZeroAndNegativeLimit(t *testing.T) {
	t.Parallel()

	if _, err := parseRowInspectionLimit("0"); err == nil {
		t.Fatalf("expected zero limit error")
	}
	if _, err := parseRowInspectionLimit("-1"); err == nil {
		t.Fatalf("expected negative limit error")
	}
	if _, err := normalizeRowInspectionLimit(0); err == nil {
		t.Fatalf("expected cli zero limit error")
	}
}

func TestRowInspectionDryRunQuotesHyphenTableName(t *testing.T) {
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

	if err := app.handlePeekRows(context.Background(), "tmp-users", "10"); err != nil {
		t.Fatalf("handlePeekRows returned error: %v", err)
	}
	if !strings.Contains(out.String(), "SELECT * FROM `tmp-users` LIMIT 10") {
		t.Fatalf("expected quoted preview SQL: %q", out.String())
	}
}

func TestCLIRowInspectionJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	countApp, countOut, countErr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{databases: []string{"app_prod"}, rowCount: 12813},
	})
	if err := countApp.Run(context.Background(), []string{"count", "rows", "users", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("count Run returned error: %v\nstderr=%s", err, countErr.String())
	}
	var countResult RowCountResult
	if err := json.Unmarshal(countOut.Bytes(), &countResult); err != nil {
		t.Fatalf("count Unmarshal returned error: %v", err)
	}
	if countResult.Rows != 12813 || countResult.Table != "users" {
		t.Fatalf("unexpected count result: %+v", countResult)
	}

	peekApp, peekOut, peekErr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{databases: []string{"app_prod"}},
	})
	if err := peekApp.Run(context.Background(), []string{"peek", "rows", "users", "--connection", "prod", "--database", "app_prod", "--limit", "20", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("peek Run returned error: %v\nstderr=%s", err, peekErr.String())
	}
	var previewResult RowPreviewResult
	if err := json.Unmarshal(peekOut.Bytes(), &previewResult); err != nil {
		t.Fatalf("peek Unmarshal returned error: %v", err)
	}
	if previewResult.Limit != 20 || len(previewResult.Columns) != 3 || len(previewResult.Rows) == 0 {
		t.Fatalf("unexpected peek result: %+v", previewResult)
	}

	sampleApp, sampleOut, sampleErr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{databases: []string{"app_prod"}},
	})
	if err := sampleApp.Run(context.Background(), []string{"sample", "rows", "users", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("sample Run returned error: %v\nstderr=%s", err, sampleErr.String())
	}
	if err := json.Unmarshal(sampleOut.Bytes(), &previewResult); err != nil {
		t.Fatalf("sample Unmarshal returned error: %v", err)
	}
	if previewResult.Limit != defaultRowInspectionLimit || len(previewResult.Rows) == 0 {
		t.Fatalf("unexpected sample result: %+v", previewResult)
	}
}

func TestCLITextRowRendering(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	rowSet := &driver.RowSet{
		Columns: []string{"id", "email", "bio"},
		Rows: [][]any{
			{1, "a@example.com", nil},
			{2, "b@example.com", strings.Repeat("x", rowPreviewCellLimit+20)},
		},
	}
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: &databaseSelectionConnector{databases: []string{"app_prod"}, rowSet: rowSet},
	})
	if err := app.Run(context.Background(), []string{"peek", "rows", "users", "--connection", "prod", "--database", "app_prod", "--config-dir", root}); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "email") || !strings.Contains(output, "NULL") || !strings.Contains(output, "...") {
		t.Fatalf("unexpected row output: %q", output)
	}
}

func TestCLIRowInspectionMissingDatabaseContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, _, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root, Connector: &databaseSelectionConnector{}})
	err := app.Run(context.Background(), []string{"count", "rows", "users", "--connection", "prod", "--config-dir", root})
	if err == nil || !strings.Contains(err.Error(), "no database selected; use --database <name>") {
		t.Fatalf("expected missing database error, got %v\nstderr=%s", err, stderr.String())
	}
}
