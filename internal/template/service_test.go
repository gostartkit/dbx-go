package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestResolveTemplatePrecedence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout returned error: %v", err)
	}

	cfg := &config.ConnectionConfig{
		Name:   "prod",
		Driver: "mysql",
		Mode:   "direct",
		Host:   "127.0.0.1",
		Port:   3306,
		User:   "root",
	}

	if err := os.MkdirAll(store.ConnectionTemplatesDir(cfg.Name), 0o755); err != nil {
		t.Fatalf("MkdirAll connection templates: %v", err)
	}

	globalTemplate := `{
  "name": "global_create_database",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "global", "sql": "GLOBAL"}]
}`
	connectionTemplate := `{
  "name": "connection_create_database",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "connection", "sql": "CONNECTION"}]
}`

	if err := os.WriteFile(filepath.Join(store.GlobalTemplatesDir(), "global.json"), []byte(globalTemplate), 0o644); err != nil {
		t.Fatalf("WriteFile global: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.ConnectionTemplatesDir(cfg.Name), "connection.json"), []byte(connectionTemplate), 0o644); err != nil {
		t.Fatalf("WriteFile connection: %v", err)
	}

	service := NewService(store)
	tpl, err := service.Resolve("create database", cfg)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if tpl.Layer != "connection" {
		t.Fatalf("Resolve layer = %q, want connection", tpl.Layer)
	}
	if tpl.Name != "connection_create_database" {
		t.Fatalf("Resolve name = %q", tpl.Name)
	}
}

func TestResolveFallsBackToBuiltin(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout returned error: %v", err)
	}

	cfg := &config.ConnectionConfig{
		Name:   "dev",
		Driver: "mysql",
		Mode:   "direct",
		Host:   "127.0.0.1",
		Port:   3306,
		User:   "root",
	}

	service := NewService(store)
	tpl, err := service.Resolve("show databases", cfg)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if tpl.Layer != "builtin" {
		t.Fatalf("Resolve layer = %q, want builtin", tpl.Layer)
	}
}

func TestListResolvedPrefersHigherScopeByName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout returned error: %v", err)
	}

	cfg := &config.ConnectionConfig{
		Name:   "prod",
		Driver: "mysql",
	}

	if err := os.MkdirAll(store.ConnectionTemplatesDir(cfg.Name), 0o755); err != nil {
		t.Fatalf("MkdirAll connection templates: %v", err)
	}

	writeTemplate := func(path string, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	writeTemplate(filepath.Join(store.GlobalTemplatesDir(), "shared.json"), `{
  "name": "shared_workflow",
  "description": "global",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "global", "sql": "GLOBAL"}]
}`)
	writeTemplate(filepath.Join(store.ConnectionTemplatesDir(cfg.Name), "shared.json"), `{
  "name": "shared_workflow",
  "description": "connection",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "connection", "sql": "CONNECTION"}]
}`)

	service := NewService(store)
	templates, err := service.ListResolved(cfg)
	if err != nil {
		t.Fatalf("ListResolved returned error: %v", err)
	}

	found := false
	for _, candidate := range templates {
		if candidate.Name != "shared_workflow" {
			continue
		}
		found = true
		if candidate.Layer != "connection" {
			t.Fatalf("Layer = %q, want connection", candidate.Layer)
		}
		if candidate.Description != "connection" {
			t.Fatalf("Description = %q, want connection", candidate.Description)
		}
	}
	if !found {
		t.Fatalf("shared_workflow not found in resolved templates")
	}
}

func TestResolveNamedAnyRejectsDuplicateNamesAtSameScope(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout returned error: %v", err)
	}

	writeTemplate := func(path string, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	writeTemplate(filepath.Join(store.GlobalTemplatesDir(), "a.json"), `{
  "name": "duplicate",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "a", "sql": "A"}]
}`)
	writeTemplate(filepath.Join(store.GlobalTemplatesDir(), "b.json"), `{
  "name": "duplicate",
  "match": {"command": "drop database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "b", "sql": "B"}]
}`)

	service := NewService(store)
	_, err := service.ResolveNamedAny(&config.ConnectionConfig{Driver: "mysql"}, "duplicate")
	if err == nil || !strings.Contains(err.Error(), "multiple templates named") {
		t.Fatalf("ResolveNamedAny error = %v", err)
	}
}
