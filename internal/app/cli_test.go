package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/config"
)

func TestCLIConnectionCreateGeneratesConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	app, stdout, stderr := newCLIApp(t, "", root)

	err := app.Run(context.Background(), []string{
		"connection", "create", "prod",
		"--mode", "ssh",
		"--host", "10.0.1.20",
		"--port", "3306",
		"--user", "root",
		"--password-env", "MYSQL_PROD_PASSWORD",
		"--ssh-host", "bastion.example.com",
		"--ssh-port", "22",
		"--ssh-user", "ubuntu",
		"--ssh-private-key", "~/.ssh/id_rsa",
		"--connect-timeout", "12",
		"--query-timeout", "45",
		"--config-dir", root,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	store := config.NewStore(root)
	cfg, err := store.LoadConnection("prod")
	if err != nil {
		t.Fatalf("LoadConnection returned error: %v", err)
	}

	if cfg.Mode != "ssh" || cfg.Host != "10.0.1.20" || cfg.User != "root" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Timeout.ConnectSeconds != 12 || cfg.Timeout.QuerySeconds != 45 {
		t.Fatalf("unexpected timeouts: %+v", cfg.Timeout)
	}
	if cfg.SSH == nil || cfg.SSH.Host != "bastion.example.com" || cfg.SSH.PrivateKey != "~/.ssh/id_rsa" {
		t.Fatalf("unexpected ssh config: %+v", cfg.SSH)
	}
	if !strings.Contains(stdout.String(), filepath.Join(root, "prod", "config.json")) {
		t.Fatalf("stdout missing saved path: %q", stdout.String())
	}
}

func TestCLIConnectionEditPreservesUnspecifiedFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, _, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{
		"connection", "edit", "prod",
		"--host", "10.0.1.30",
		"--query-timeout", "60",
		"--config-dir", root,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	cfg, err := store.LoadConnection("prod")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "10.0.1.30" {
		t.Fatalf("host = %q, want updated value", cfg.Host)
	}
	if cfg.User != "root" || cfg.Port != 3306 {
		t.Fatalf("unexpected preserved fields: %+v", cfg)
	}
	if cfg.Timeout.QuerySeconds != 60 || cfg.Timeout.ConnectSeconds != 10 {
		t.Fatalf("unexpected timeouts: %+v", cfg.Timeout)
	}
}

func TestCLIConnectionDeleteConfirmation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, _, stderr := newCLIApp(t, "y\n", root)
	err := app.Run(context.Background(), []string{"connection", "delete", "prod", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if store.ConnectionExists("prod") {
		t.Fatalf("connection still exists after delete")
	}
}

func TestCLICreateDatabaseDryRunRedactsSecrets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	templateJSON := "{\n" +
		`  "name": "create_database_with_user",` + "\n" +
		`  "transaction": true,` + "\n" +
		`  "match": {"command": "create database", "driver": "mysql"},` + "\n" +
		`  "inputs": [{"name": "password", "type": "secret", "prompt": "Password"}],` + "\n" +
		`  "actions": [` + "\n" +
		`    {"type": "sql", "description": "Create database", "sql": "CREATE DATABASE IF NOT EXISTS ` + "`{{database}}`" + `"},` + "\n" +
		`    {"type": "sql", "description": "Create user", "sql": "CREATE USER IF NOT EXISTS '{{database}}'@'%' IDENTIFIED BY '{{password}}'"}` + "\n" +
		`  ]` + "\n" +
		`}`
	if err := os.WriteFile(filepath.Join(store.GlobalTemplatesDir(), "create_database_with_user.json"), []byte(templateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"--dry-run",
		"--format", "json",
		"create", "database", "app_demo",
		"--template", "create_database_with_user",
		"--input", "password=secret123",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result PlanExecutionResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if !result.DryRun || len(result.Actions) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if strings.Contains(stdout.String(), "secret123") {
		t.Fatalf("stdout leaked secret: %s", stdout.String())
	}
	if !strings.Contains(result.Actions[1].SQL, "***") {
		t.Fatalf("expected redacted SQL, got %q", result.Actions[1].SQL)
	}
}

func TestCLIAmbiguousTemplateFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "a.json"), "{\"name\":\"aaa\",\"match\":{\"command\":\"create database\",\"driver\":\"mysql\"},\"actions\":[{\"type\":\"sql\",\"description\":\"A\",\"sql\":\"CREATE DATABASE `{{database}}`\"}]}")
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "b.json"), "{\"name\":\"bbb\",\"match\":{\"command\":\"create database\",\"driver\":\"mysql\"},\"actions\":[{\"type\":\"sql\",\"description\":\"B\",\"sql\":\"CREATE DATABASE `{{database}}`\"}]}")

	app, _, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"--dry-run",
		"create", "database", "app_demo",
	})
	if err == nil {
		t.Fatalf("expected error for ambiguous templates")
	}
	if app.ExitStatus() == 0 {
		t.Fatalf("expected non-zero exit status")
	}
	if !strings.Contains(err.Error(), "multiple templates match") {
		t.Fatalf("unexpected error: %v\nstderr=%s", err, stderr.String())
	}
}

func TestCLIConnectionShowJSONRedactsSecrets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	cfg := sampleConnection("prod")
	cfg.Password = "super-secret"
	cfg.PasswordEnv = ""
	cfg.SSH = &config.SSHConfig{
		Host:       "bastion.example.com",
		Port:       22,
		User:       "ubuntu",
		Password:   "ssh-secret",
		PrivateKey: "",
	}
	if err := store.SaveConnection(cfg); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"connection", "show", "prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "super-secret") || strings.Contains(stdout.String(), "ssh-secret") {
		t.Fatalf("json output leaked secrets: %s", stdout.String())
	}

	var result RedactedConnection
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Password.Mode != "saved" || result.Password.Value != "[redacted]" {
		t.Fatalf("unexpected redaction: %+v", result.Password)
	}
	if result.SSH == nil || result.SSH.PasswordMode != "saved" {
		t.Fatalf("unexpected ssh redaction: %+v", result.SSH)
	}
}

func TestCLIStatusParsesGlobalFlagsAfterCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"status", "--connection", "prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result StatusResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.ConnectionName != "prod" || !result.SelectedByFlag {
		t.Fatalf("unexpected status result: %+v", result)
	}
}

func TestCLIHelpForMultiWordCommand(t *testing.T) {
	t.Parallel()

	app, stdout, stderr := newCLIApp(t, "", t.TempDir())
	err := app.Run(context.Background(), []string{"help", "create", "database"})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Create a database from the resolved template.") {
		t.Fatalf("help output missing expected text: %q", stdout.String())
	}
}

func newCLIApp(t *testing.T, stdin string, _ string) (*cmd.App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := NewCommandApp(strings.NewReader(stdin), &stdout, &stderr)
	return cli, &stdout, &stderr
}

func sampleConnection(name string) *config.ConnectionConfig {
	return &config.ConnectionConfig{
		Name:        name,
		Driver:      "mysql",
		Mode:        "direct",
		Host:        "127.0.0.1",
		Port:        3306,
		User:        "root",
		PasswordEnv: "MYSQL_PASSWORD",
		Timeout: &config.TimeoutConfig{
			ConnectSeconds: 10,
			QuerySeconds:   30,
		},
	}
}

func writeTemplate(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
