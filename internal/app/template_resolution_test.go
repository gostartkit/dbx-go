package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestResolveTemplateForActionREPLChoosesConnectionLayerOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.ConnectionTemplatesDir("prod"), 0o755); err != nil {
		t.Fatal(err)
	}

	writeTemplate(t, filepath.Join(store.ConnectionTemplatesDir("prod"), "a.json"), `{
  "name": "conn_primary",
  "category": "database",
  "description": "connection primary workflow",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "A", "sql": "A"}]
}`)
	writeTemplate(t, filepath.Join(store.ConnectionTemplatesDir("prod"), "b.json"), `{
  "name": "conn_secondary",
  "category": "database",
  "description": "connection secondary workflow",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "B", "sql": "B"}]
}`)
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "global.json"), `{
  "name": "global_fallback",
  "category": "database",
  "description": "global fallback workflow",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "GLOBAL", "sql": "GLOBAL"}]
}`)

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("2\n"), &out, &out, Options{ConfigDir: root, Connector: &readOnlyConnector{}})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")

	selected, err := app.resolveTemplateForAction(context.Background(), "create database", app.session.Connection)
	if err != nil {
		t.Fatalf("resolveTemplateForAction returned error: %v", err)
	}
	if selected.Name != "conn_secondary" {
		t.Fatalf("selected template = %q, want conn_secondary", selected.Name)
	}

	output := out.String()
	if !strings.Contains(output, "conn_primary") || !strings.Contains(output, "conn_secondary") {
		t.Fatalf("missing connection candidates: %q", output)
	}
	if strings.Contains(output, "global_fallback") || strings.Contains(output, "builtin_create_database") {
		t.Fatalf("unexpected lower-priority candidates in output: %q", output)
	}
	if !strings.Contains(output, "scope=connection") || !strings.Contains(output, "category=database") || !strings.Contains(output, "description=connection secondary workflow") {
		t.Fatalf("candidate details missing from output: %q", output)
	}
}

func TestResolveTemplateForActionREPLChoosesGlobalLayerOnlyWhenConnectionHasNoMatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "a.json"), `{
  "name": "global_primary",
  "category": "database",
  "description": "global primary workflow",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "A", "sql": "A"}]
}`)
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "b.json"), `{
  "name": "global_secondary",
  "category": "database",
  "description": "global secondary workflow",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "B", "sql": "B"}]
}`)

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("1\n"), &out, &out, Options{ConfigDir: root, Connector: &readOnlyConnector{}})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")

	selected, err := app.resolveTemplateForAction(context.Background(), "create database", app.session.Connection)
	if err != nil {
		t.Fatalf("resolveTemplateForAction returned error: %v", err)
	}
	if selected.Name != "global_primary" {
		t.Fatalf("selected template = %q, want global_primary", selected.Name)
	}

	output := out.String()
	if !strings.Contains(output, "global_primary") || !strings.Contains(output, "global_secondary") {
		t.Fatalf("missing global candidates: %q", output)
	}
	if strings.Contains(output, "builtin_create_database") {
		t.Fatalf("unexpected builtin candidate in output: %q", output)
	}
	if !strings.Contains(output, "scope=global") {
		t.Fatalf("expected global scope in output: %q", output)
	}
}

func TestResolveTemplateForActionREPLDoesNotDefaultFirstCandidate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "a.json"), `{
  "name": "global_primary",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "A", "sql": "A"}]
}`)
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "b.json"), `{
  "name": "global_secondary",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "B", "sql": "B"}]
}`)

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader("\n2\n"), &out, &out, Options{ConfigDir: root, Connector: &readOnlyConnector{}})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	app.session.Connection = sampleConnection("prod")

	selected, err := app.resolveTemplateForAction(context.Background(), "create database", app.session.Connection)
	if err != nil {
		t.Fatalf("resolveTemplateForAction returned error: %v", err)
	}
	if selected.Name != "global_secondary" {
		t.Fatalf("selected template = %q, want global_secondary", selected.Name)
	}
	if !strings.Contains(out.String(), "Please choose one of the listed options.") {
		t.Fatalf("expected explicit selection prompt, got %q", out.String())
	}
}

func TestCLICreateUserAmbiguousTemplateFailsBeforePasswordOrConfirmation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "a.json"), `{
  "name": "readonly_a",
  "description": "readonly variant a",
  "match": {"command": "create user", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "A", "sql": "A"}]
}`)
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "b.json"), `{
  "name": "readonly_b",
  "description": "readonly variant b",
  "match": {"command": "create user", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "B", "sql": "B"}]
}`)

	app, _, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"create", "user", "analytics_ro",
	})
	if err == nil {
		t.Fatalf("expected ambiguous template error")
	}
	if !strings.Contains(err.Error(), "multiple templates matched command \"create user\" at global scope") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "readonly_a") || !strings.Contains(err.Error(), "readonly_b") {
		t.Fatalf("candidate list missing from error: %v", err)
	}
	if !strings.Contains(err.Error(), "run template <name>") || !strings.Contains(err.Error(), "--template <name>") {
		t.Fatalf("missing explicit selection hint: %v", err)
	}
	if strings.Contains(stderr.String(), "Password:") || strings.Contains(stderr.String(), "confirmation required") {
		t.Fatalf("CLI should fail before password or confirmation prompts: %q", stderr.String())
	}
}
