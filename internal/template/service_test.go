package template

import (
	"os"
	"path/filepath"
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
	tpl, err := service.Resolve("list databases", cfg)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if tpl.Layer != "builtin" {
		t.Fatalf("Resolve layer = %q, want builtin", tpl.Layer)
	}
}
