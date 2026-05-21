package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func TestCLIConnectionCreateGeneratesConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	app, stdout, stderr := newCLIApp(t, "", root)

	err := app.Run(context.Background(), []string{
		"create", "connection", "prod",
		"--yes",
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

func TestCLIConnectionCreateProxySSHGeneratesConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	app, stdout, stderr := newCLIApp(t, "", root)

	err := app.Run(context.Background(), []string{
		"create", "connection", "prod_proxy",
		"--yes",
		"--mode", "proxy-ssh",
		"--host", "10.0.1.20",
		"--port", "3306",
		"--user", "root",
		"--password-env", "MYSQL_PROD_PASSWORD",
		"--proxy-url", "socks5://proxy_user:proxy_password@127.0.0.1:1080",
		"--ssh-host", "bastion.example.com",
		"--ssh-port", "22",
		"--ssh-user", "ubuntu",
		"--ssh-private-key", "~/.ssh/id_rsa",
		"--config-dir", root,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	store := config.NewStore(root)
	cfg, err := store.LoadConnection("prod_proxy")
	if err != nil {
		t.Fatalf("LoadConnection returned error: %v", err)
	}
	if cfg.Mode != "proxy-ssh" || cfg.Proxy == nil || cfg.Proxy.URL != "socks5://proxy_user:proxy_password@127.0.0.1:1080" {
		t.Fatalf("unexpected proxy config: %+v", cfg)
	}
	if cfg.SSH == nil || cfg.SSH.Host != "bastion.example.com" {
		t.Fatalf("unexpected ssh config: %+v", cfg.SSH)
	}
	if !strings.Contains(stdout.String(), filepath.Join(root, "prod_proxy", "config.json")) {
		t.Fatalf("stdout missing saved path: %q", stdout.String())
	}
}

func TestCLIConnectionCreateProxyGeneratesConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	app, stdout, stderr := newCLIApp(t, "", root)

	err := app.Run(context.Background(), []string{
		"create", "connection", "prod_proxy",
		"--yes",
		"--mode", "proxy",
		"--host", "10.0.1.20",
		"--port", "3306",
		"--user", "root",
		"--password-env", "MYSQL_PROD_PASSWORD",
		"--proxy-url", "socks5://proxy_user:proxy_password@127.0.0.1:1080",
		"--config-dir", root,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	store := config.NewStore(root)
	cfg, err := store.LoadConnection("prod_proxy")
	if err != nil {
		t.Fatalf("LoadConnection returned error: %v", err)
	}
	if cfg.Mode != "proxy" || cfg.Proxy == nil || cfg.Proxy.URL != "socks5://proxy_user:proxy_password@127.0.0.1:1080" {
		t.Fatalf("unexpected proxy config: %+v", cfg)
	}
	if cfg.SSH != nil {
		t.Fatalf("proxy mode should not save ssh config: %+v", cfg.SSH)
	}
	if !strings.Contains(stdout.String(), filepath.Join(root, "prod_proxy", "config.json")) {
		t.Fatalf("stdout missing saved path: %q", stdout.String())
	}
}

func TestCLIConnectionCreateSavesWhenTestFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: failingConnector{openErr: errors.New("mysql error: ping database: ssh error: complete SSH handshake with 39.108.126.24:22: ssh: handshake failed")},
	})

	err := app.Run(context.Background(), []string{
		"create", "connection", "prod",
		"--yes",
		"--mode", "direct",
		"--host", "127.0.0.1",
		"--port", "3306",
		"--user", "root",
		"--password", "secret123",
		"--test",
		"--format", "json",
		"--config-dir", root,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	store := config.NewStore(root)
	if !store.ConnectionExists("prod") {
		t.Fatalf("expected saved connection after failed test")
	}
	if !strings.Contains(stderr.String(), "Connection test failed:") {
		t.Fatalf("stderr missing warning: %q", stderr.String())
	}

	var result ConnectionCreateResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if !result.OK || !result.Saved {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.TestOK == nil || *result.TestOK {
		t.Fatalf("expected test_ok false, got %+v", result.TestOK)
	}
	if result.Warning != "connection test failed" {
		t.Fatalf("warning = %q", result.Warning)
	}
	if result.OverwriteCommand != "create connection prod --overwrite" {
		t.Fatalf("overwrite command = %q", result.OverwriteCommand)
	}
	if strings.Contains(stdout.String(), "secret123") || strings.Contains(stderr.String(), "secret123") {
		t.Fatalf("secret leaked in output")
	}
}

func TestCLIConnectionEditCommandRemoved(t *testing.T) {
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
		"edit", "connection", "prod",
		"--yes",
		"--host", "10.0.1.30",
		"--query-timeout", "60",
		"--config-dir", root,
	})
	if err == nil {
		t.Fatalf("expected removed command failure")
	}
	_ = stderr
}

func TestCLIConnectionTestCommandRemoved(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &spyConnector{}
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: connector,
	})
	err := app.Run(context.Background(), []string{"test", "connection", "prod", "--config-dir", root})
	if err == nil {
		t.Fatalf("expected removed command failure")
	}
	if connector.openCalls != 0 || connector.lastName != "" {
		t.Fatalf("unexpected connector usage: calls=%d name=%q", connector.openCalls, connector.lastName)
	}
	_ = stdout
	_ = stderr
}

func TestCLIConnectionTestVerboseCommandRemoved(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &spyConnector{}
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: connector,
	})

	err := app.Run(context.Background(), []string{"test", "connection", "prod", "--verbose", "--format", "json", "--config-dir", root})
	if err == nil {
		t.Fatalf("expected removed command failure")
	}
	_ = stdout
	_ = stderr
}

func TestCLIConnectionCreateValidationFailureStillFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	app, _, stderr := newCLIApp(t, "", root)

	err := app.Run(context.Background(), []string{
		"create", "connection", "prod",
		"--yes",
		"--mode", "direct",
		"--user", "root",
		"--config-dir", root,
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if config.NewStore(root).ConnectionExists("prod") {
		t.Fatalf("did not expect saved config on validation failure")
	}
	if app.ExitStatus() == 0 {
		t.Fatalf("expected non-zero exit status")
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestCLIConnectionCreateWriteFailureStillFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "prod"), []byte("not-a-directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	app, _, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{
		"create", "connection", "prod",
		"--yes",
		"--mode", "direct",
		"--host", "127.0.0.1",
		"--port", "3306",
		"--user", "root",
		"--password", "secret123",
		"--config-dir", root,
	})
	if err == nil {
		t.Fatalf("expected write error")
	}
	if app.ExitStatus() == 0 {
		t.Fatalf("expected non-zero exit status")
	}
	if strings.Contains(stderr.String(), "secret123") {
		t.Fatalf("stderr leaked secret: %q", stderr.String())
	}
}

func TestCLIConnectionCreateOverwriteProtectedWithoutForce(t *testing.T) {
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
		"create", "connection", "prod",
		"--mode", "direct",
		"--host", "127.0.0.1",
		"--port", "3306",
		"--user", "root",
		"--password", "secret123",
		"--config-dir", root,
	})
	if err == nil {
		t.Fatalf("expected overwrite protection error")
	}
	if app.ExitStatus() == 0 {
		t.Fatalf("expected non-zero exit status")
	}
	if strings.Contains(stderr.String(), "secret123") {
		t.Fatalf("stderr leaked secret: %q", stderr.String())
	}
}

func TestCLIDoctorConnectionJSONOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	cfg := sampleConnection("prod_proxy")
	cfg.Mode = "proxy-ssh"
	cfg.Proxy = &config.ProxyConfig{
		URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080",
	}
	cfg.SSH = &config.SSHConfig{
		Host:       "bastion.example.com",
		Port:       22,
		User:       "ubuntu",
		PrivateKey: "~/.ssh/id_rsa",
	}
	if err := store.SaveConnection(cfg); err != nil {
		t.Fatal(err)
	}

	connector := &spyConnector{
		openErr: util.WrapLayer("proxy", "dial socks5://proxy_user:***@127.0.0.1:1080", errors.New("connection refused")),
	}
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: connector,
	})
	err := app.Run(context.Background(), []string{"doctor", "--connection", "prod_proxy", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected json mode to keep stderr empty, got %q", stderr.String())
	}

	var result DoctorResult
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &result); unmarshalErr != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", unmarshalErr, stdout.String())
	}
	if result.Connection != "prod_proxy" {
		t.Fatalf("unexpected doctor result: %+v", result)
	}
}

func TestCLIConnectionShowJSONRedactsProxyModeSecrets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	cfg := sampleConnection("prod_proxy")
	cfg.Mode = "proxy"
	cfg.Proxy = &config.ProxyConfig{
		URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080",
	}
	if err := store.SaveConnection(cfg); err != nil {
		t.Fatal(err)
	}

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"show", "connection", "prod_proxy", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "proxy_password") {
		t.Fatalf("json output leaked proxy password: %s", stdout.String())
	}

	var result RedactedConnection
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Mode != "proxy" || result.Proxy == nil || result.Proxy.URL != "socks5://proxy_user:***@127.0.0.1:1080" {
		t.Fatalf("unexpected proxy redaction: %+v", result)
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

	app, _, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"drop", "connection", "prod", "--yes", "--config-dir", root})
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
	if !strings.Contains(err.Error(), "multiple templates matched command \"create database\" at global scope") {
		t.Fatalf("unexpected error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(err.Error(), "aaa") || !strings.Contains(err.Error(), "bbb") {
		t.Fatalf("missing candidate list in error: %v", err)
	}
	if !strings.Contains(err.Error(), "run template <name>") {
		t.Fatalf("missing explicit template hint in error: %v", err)
	}
}

func TestCLIUnknownShowTargetUsesShowContext(t *testing.T) {
	t.Parallel()

	app, _, stderr := newCLIApp(t, "", t.TempDir())
	err := app.Run(context.Background(), []string{"show", "indexes"})
	if err == nil {
		t.Fatalf("expected error")
	}
	message := err.Error()
	if !strings.Contains(message, `unknown show target "indexes"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(message, `Did you mean show?`) {
		t.Fatalf("unexpected root-level suggestion: %v", err)
	}
	if !strings.Contains(message, "available show targets:") {
		t.Fatalf("missing show targets list: %v", err)
	}
	if !strings.Contains(message, "\n  users") || !strings.Contains(message, "\n  user") {
		t.Fatalf("missing show user targets: %v", err)
	}
	if strings.Contains(message, "\n  connect\n") || strings.Contains(message, "\n  create\n") {
		t.Fatalf("unexpected root commands in error: %v", err)
	}
	_ = stderr
}

func TestCLIUnknownShowTargetSuggestsClosestMatch(t *testing.T) {
	t.Parallel()

	app, _, _ := newCLIApp(t, "", t.TempDir())
	err := app.Run(context.Background(), []string{"show", "tabl"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), `did you mean "table"?`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCLIRootUnknownCommandStillSuggestsRootMatch(t *testing.T) {
	t.Parallel()

	app, _, _ := newCLIApp(t, "", t.TempDir())
	err := app.Run(context.Background(), []string{"sho"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), `unknown command "sho"`) || !strings.Contains(err.Error(), "Did you mean show?") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCLIUnknownRunTargetSuggestsClosestMatch(t *testing.T) {
	t.Parallel()

	app, _, _ := newCLIApp(t, "", t.TempDir())
	err := app.Run(context.Background(), []string{"run", "sq"})
	if err == nil {
		t.Fatalf("expected error")
	}
	message := err.Error()
	if !strings.Contains(message, `unknown run target "sq"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(message, `did you mean "sql"?`) {
		t.Fatalf("missing contextual suggestion: %v", err)
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
	cfg.Mode = "proxy-ssh"
	cfg.Proxy = &config.ProxyConfig{
		URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080",
	}
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
	err := app.Run(context.Background(), []string{"show", "connection", "prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "super-secret") || strings.Contains(stdout.String(), "ssh-secret") || strings.Contains(stdout.String(), "proxy_password") {
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
	if result.Proxy == nil || result.Proxy.URL != "socks5://proxy_user:***@127.0.0.1:1080" {
		t.Fatalf("unexpected proxy redaction: %+v", result.Proxy)
	}
}

func TestCLIShowContextParsesGlobalFlagsAfterCommand(t *testing.T) {
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
	err := app.Run(context.Background(), []string{"show", "context", "--connection", "prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result ContextResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Connection != "prod" || result.Mode == "" {
		t.Fatalf("unexpected status result: %+v", result)
	}
}

func TestCLIShowContextIncludesDatabaseFlag(t *testing.T) {
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
		Connector: &databaseSelectionConnector{databases: []string{"app_prod", "app_demo"}},
	})
	err := app.Run(context.Background(), []string{"show", "context", "--connection", "prod", "--database", "app_prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result ContextResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if result.Database != "app_prod" {
		t.Fatalf("database = %q, want app_prod", result.Database)
	}
}

func TestCLIShowDatabasesCanonical(t *testing.T) {
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
	err := app.Run(context.Background(), []string{"show", "databases", "--connection", "prod", "--template", "builtin_list_databases", "--dry-run", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"command": "show databases"`) {
		t.Fatalf("stdout missing canonical command: %q", stdout.String())
	}
}

func TestCLIShowDatabasesDryRunJSON(t *testing.T) {
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
	err := app.Run(context.Background(), []string{"show", "databases", "--connection", "prod", "--template", "builtin_list_databases", "--dry-run", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"command": "show databases"`) {
		t.Fatalf("stdout missing canonical command: %q", stdout.String())
	}
}

func TestCLICreateDatabaseAllowsHyphenName(t *testing.T) {
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
	err := app.Run(context.Background(), []string{
		"--connection", "prod",
		"--dry-run",
		"--format", "json",
		"--config-dir", root,
		"create", "database", "greenhn-dev",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "`greenhn-dev`") {
		t.Fatalf("stdout missing quoted database name: %q", stdout.String())
	}
}

func TestCLICreateUserDryRunRedactsPassword(t *testing.T) {
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
	err := app.Run(context.Background(), []string{
		"--connection", "prod",
		"--database", "app_prod",
		"--dry-run",
		"--format", "json",
		"create", "user", "analytics-ro",
		"--password", "secret123",
		"--grant", "readonly",
		"--config-dir", root,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "secret123") {
		t.Fatalf("stdout leaked password: %s", stdout.String())
	}

	var result PlanExecutionResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if !result.DryRun || len(result.Actions) == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.Contains(result.Actions[0].SQL, "***") {
		t.Fatalf("expected redacted SQL, got %q", result.Actions[0].SQL)
	}
	if !strings.Contains(result.Actions[len(result.Actions)-1].SQL, "GRANT SELECT ON `app_prod`.*") {
		t.Fatalf("missing readonly grant SQL: %+v", result.Actions)
	}
}

func TestCLICreateUserRequiresYesUnlessDryRun(t *testing.T) {
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
		"--connection", "prod",
		"create", "user", "analytics-ro",
		"--password", "secret123",
		"--config-dir", root,
	})
	if err == nil {
		t.Fatalf("expected confirmation error")
	}
	if app.ExitStatus() == 0 {
		t.Fatalf("expected non-zero exit status")
	}
	if !strings.Contains(err.Error(), "confirmation required") {
		t.Fatalf("unexpected error: %v\nstderr=%s", err, stderr.String())
	}
}

func TestCLIDropUserDryRunDoesNotRequireYes(t *testing.T) {
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
	err := app.Run(context.Background(), []string{
		"--connection", "prod",
		"--dry-run",
		"--format", "json",
		"drop", "user", "analytics-ro",
		"--config-dir", root,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"command": "drop user"`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestCLIDropDatabaseAllowsHyphenName(t *testing.T) {
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
	err := app.Run(context.Background(), []string{
		"--connection", "prod",
		"--dry-run",
		"--format", "json",
		"--config-dir", root,
		"drop", "database", "greenhn-dev",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "`greenhn-dev`") {
		t.Fatalf("stdout missing quoted database name: %q", stdout.String())
	}
}

func TestCLIMutatingCommandRequiresYes(t *testing.T) {
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
	err := app.Run(context.Background(), []string{
		"--connection", "prod",
		"--format", "json",
		"--config-dir", root,
		"drop", "database", "greenhn-dev",
	})
	if err == nil {
		t.Fatalf("expected confirmation error")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected handled json error, got stderr=%q", stderr.String())
	}

	var envelope ErrorEnvelope
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &envelope); unmarshalErr != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", unmarshalErr, stdout.String())
	}
	if envelope.Error == nil || envelope.Error.Code != "CONFIRMATION_REQUIRED" {
		t.Fatalf("unexpected error envelope: %+v", envelope)
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

func TestCLIHelpRootCommandHasOutput(t *testing.T) {
	t.Parallel()

	app, stdout, stderr := newCLIApp(t, "", t.TempDir())
	err := app.Run(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	output := stdout.String() + stderr.String()
	if !strings.Contains(output, "Usage:") || !strings.Contains(output, "Available Commands:") {
		t.Fatalf("help output missing expected sections: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCLIJSONErrorIncludesCode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	err := app.Run(context.Background(), []string{"show", "connection", "missing", "--format", "json", "--config-dir", root})
	if err == nil {
		t.Fatalf("expected error")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for handled json error: %q", stderr.String())
	}

	var envelope ErrorEnvelope
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &envelope); unmarshalErr != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", unmarshalErr, stdout.String())
	}
	if envelope.Error == nil || envelope.Error.Code != "CONFIG_NOT_FOUND" {
		t.Fatalf("unexpected error envelope: %+v", envelope)
	}
}

func newCLIApp(t *testing.T, stdin string, _ string) (*cmd.App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	return newCLIAppWithOptions(t, stdin, Options{})
}

func newCLIAppWithOptions(t *testing.T, stdin string, options Options) (*cmd.App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := newCommandAppWithOptions(strings.NewReader(stdin), &stdout, &stderr, options)
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
