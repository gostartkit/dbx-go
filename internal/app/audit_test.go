package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestCLIAuditLogJSONAndText(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendAudit(&config.AuditRecord{
		Command:    "status",
		Connection: "prod",
		Mode:       "proxy-ssh",
		Success:    true,
		DurationMS: 12,
	}); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	if err := app.Run(context.Background(), []string{"audit", "log", "--config-dir", root}); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Recent audit entries:") || !strings.Contains(stdout.String(), "status") {
		t.Fatalf("unexpected text output: %q", stdout.String())
	}

	app, stdout, stderr = newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	if err := app.Run(context.Background(), []string{"audit", "log", "--format", "json", "--config-dir", root}); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result AuditLogResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if len(result.Entries) < 1 {
		t.Fatalf("unexpected audit result: %+v", result)
	}
	first := result.Entries[0]
	if first.Mode != "proxy-ssh" || first.Command != "status" {
		t.Fatalf("unexpected audit result: %+v", result)
	}
}

func TestAuditLogRedactsSecretsAndDoesNotFailCommandOnWriteError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "create_database_with_user.json"), `{
  "name": "create_database_with_user",
  "match": {"command": "create database", "driver": "mysql"},
  "inputs": [{"name": "password", "type": "secret", "prompt": "Password"}],
  "actions": [{"type": "sql", "description": "Create database", "sql": "CREATE DATABASE IF NOT EXISTS `+"`{{database}}`"+`"}]
}`)

	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	err := app.Run(context.Background(), []string{
		"--connection", "prod",
		"--dry-run",
		"create", "database", "demo",
		"--template", "create_database_with_user",
		"--input", "password=secret123",
		"--yes",
		"--config-dir", root,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}

	data, err := os.ReadFile(store.AuditLogPath())
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(data), "secret123") {
		t.Fatalf("audit log leaked secret: %s", string(data))
	}
	if !strings.Contains(string(data), `"command":"create database"`) {
		t.Fatalf("audit log missing command: %s", string(data))
	}

	if err := os.Remove(store.AuditLogPath()); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.AuditLogPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	app, stdout, stderr = newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	err = app.Run(context.Background(), []string{"show", "connections", "--config-dir", root})
	if err != nil {
		t.Fatalf("audit write failure should not fail command: %v\nstderr=%s", err, stderr.String())
	}
}
