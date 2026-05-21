package app

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/ui"
)

func BenchmarkBuildApp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		app := newCLIBuilder(bytes.NewReader(nil), io.Discard, io.Discard, Options{}).buildApp()
		if app == nil {
			b.Fatal("buildApp returned nil")
		}
	}
}

func BenchmarkRunLineHelp(b *testing.B) {
	app := benchmarkREPLApp(b)
	runBenchRunLine(b, app, "help")
}

func BenchmarkRunLineShowContext(b *testing.B) {
	app := benchmarkREPLApp(b)
	app.session.Connection = sampleConnection("prod")
	runBenchRunLine(b, app, "show context")
}

func BenchmarkCompletionRoot(b *testing.B) {
	app := benchmarkCompletionApp(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		completion := app.completeInput(ui.NewSingleLineCompletionRequest("", 0))
		if len(completion.Suggestions) == 0 {
			b.Fatal("expected root completion suggestions")
		}
	}
}

func BenchmarkCompletionShow(b *testing.B) {
	app := benchmarkCompletionApp(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		completion := app.completeInput(ui.NewSingleLineCompletionRequest("show ", len([]rune("show "))))
		if len(completion.Suggestions) == 0 {
			b.Fatal("expected show completion suggestions")
		}
	}
}

func BenchmarkCommandSpecs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		specs := replCommandSpecs()
		if len(specs) == 0 {
			b.Fatal("expected command specs")
		}
	}
}

func benchmarkREPLApp(b *testing.B) *Application {
	b.Helper()

	root := b.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		b.Fatalf("EnsureLayout returned error: %v", err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		b.Fatalf("SaveConnection returned error: %v", err)
	}

	var out bytes.Buffer
	app, err := NewWithOptions(bytes.NewReader(nil), &out, &out, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		b.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.DB = &sql.DB{}
	return app
}

func benchmarkCompletionApp(b *testing.B) *Application {
	b.Helper()

	root := b.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		b.Fatalf("EnsureLayout returned error: %v", err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		b.Fatalf("SaveConnection returned error: %v", err)
	}
	if err := os.MkdirAll(store.ConnectionTemplatesDir("prod"), 0o755); err != nil {
		b.Fatalf("MkdirAll returned error: %v", err)
	}
	writeBenchmarkTemplate(b, filepath.Join(store.ConnectionTemplatesDir("prod"), "conn.json"), `{
  "name": "prod_create_database",
  "tags": ["tenant", "database"],
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "create", "sql": "CREATE DATABASE `+"`{{database}}`"+`"}]
}`)

	app := benchmarkREPLApp(b)
	app.session.Connection = sampleConnection("prod")
	app.session.Database = "app_prod"
	app.session.DB = &sql.DB{}
	app.completionDBsConn = "prod"
	app.completionDBs = []string{"app_prod", "analytics_v2"}
	app.completionTablesConn = "prod"
	app.completionTablesDB = "app_prod"
	app.completionTables = []string{"users", "orders"}
	app.completionUsersConn = "prod"
	app.completionUsers = []string{"root", "analytics_ro"}
	return app
}

func runBenchRunLine(b *testing.B, app *Application, line string) {
	b.Helper()

	cmdApp := app.replCommandApp()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cmdApp.RunLine(ctx, line); err != nil {
			b.Fatalf("RunLine returned error: %v", err)
		}
	}
}

func writeBenchmarkTemplate(b *testing.B, path string, content string) {
	b.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatalf("WriteFile %s returned error: %v", path, err)
	}
}
