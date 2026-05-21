package template

import (
	"os"
	"path/filepath"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func BenchmarkTemplateResolutionSingleMatch(b *testing.B) {
	service, cfg := benchmarkTemplateService(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tpl, err := service.Resolve("create database", cfg)
		if err != nil {
			b.Fatalf("Resolve returned error: %v", err)
		}
		if tpl == nil {
			b.Fatal("expected resolved template")
		}
	}
}

func BenchmarkTemplateResolutionAmbiguousLayer(b *testing.B) {
	service, cfg := benchmarkTemplateService(b, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.Resolve("create database", cfg)
		if err == nil {
			b.Fatal("expected ambiguous resolution error")
		}
	}
}

func benchmarkTemplateService(b *testing.B, ambiguous bool) (*Service, *config.ConnectionConfig) {
	b.Helper()

	root := b.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		b.Fatalf("EnsureLayout returned error: %v", err)
	}

	cfg := &config.ConnectionConfig{
		Name:   "prod",
		Driver: "mysql",
		Mode:   "direct",
	}

	if err := os.MkdirAll(store.ConnectionTemplatesDir(cfg.Name), 0o755); err != nil {
		b.Fatalf("MkdirAll returned error: %v", err)
	}

	writeBenchmarkTemplate(b, filepath.Join(store.ConnectionTemplatesDir(cfg.Name), "conn.json"), `{
  "name": "conn_primary",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "connection", "sql": "CREATE DATABASE `+"`{{database}}`"+`"}]
}`)
	if ambiguous {
		writeBenchmarkTemplate(b, filepath.Join(store.ConnectionTemplatesDir(cfg.Name), "conn_b.json"), `{
  "name": "conn_secondary",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "connection", "sql": "CREATE DATABASE `+"`{{database}}`"+`"}]
}`)
	}
	writeBenchmarkTemplate(b, filepath.Join(store.GlobalTemplatesDir(), "global.json"), `{
  "name": "global_fallback",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "global", "sql": "CREATE DATABASE `+"`{{database}}`"+`"}]
}`)

	return NewService(store), cfg
}

func writeBenchmarkTemplate(b *testing.B, path string, content string) {
	b.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatalf("WriteFile %s returned error: %v", path, err)
	}
}
