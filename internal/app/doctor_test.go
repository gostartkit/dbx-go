package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestHandleLineConnectionDoctorParsesName(t *testing.T) {
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
	connector := &diagnosticConnector{}
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{
		ConfigDir: root,
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	exit, err := app.handleLine(context.Background(), "doctor connection prod")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if connector.openCalls != 0 {
		t.Fatalf("doctor should not open network connections")
	}
	if !strings.Contains(out.String(), "Connection doctor: prod") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestHelpConnectionIncludesDoctor(t *testing.T) {
	t.Parallel()

	entry := helpEntries["connection"].body
	if !strings.Contains(entry, "doctor connection") {
		t.Fatalf("connection help missing doctor command: %q", entry)
	}
}

func TestDoctorConnectionCommonChecks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "direct",
  "host": "127.0.0.1",
  "port": 3306,
  "user": "root",
  "password_env": "MISSING_DB_PASSWORD",
  "timeout": {
    "connect_seconds": 10,
    "query_seconds": 30
  }
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr != nil {
		t.Fatalf("doctorConnection returned error: %v", doctorErr)
	}
	if !result.OK {
		t.Fatalf("expected warnings only: %+v", result)
	}
	if !hasDoctorCheck(result, "database password env MISSING_DB_PASSWORD is set", "warn") {
		t.Fatalf("expected password env warning: %+v", result.Checks)
	}
}

func TestDoctorConnectionInlinePasswordWarns(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "direct",
  "host": "127.0.0.1",
  "port": 3306,
  "user": "root",
  "password": "super-secret"
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr != nil {
		t.Fatalf("doctorConnection returned error: %v", doctorErr)
	}
	if !hasDoctorCheck(result, "database password is inline in config", "warn") {
		t.Fatalf("expected inline password warning: %+v", result.Checks)
	}
	if containsDoctorSecret(result, "super-secret") {
		t.Fatalf("doctor result leaked secret: %+v", result)
	}
}

func TestDoctorConnectionInvalidPortFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "direct",
  "host": "127.0.0.1",
  "port": 70000,
  "user": "root"
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr == nil {
		t.Fatalf("expected failure")
	}
	if result.OK {
		t.Fatalf("expected failed doctor result")
	}
	if !hasDoctorCheck(result, "database port 70000", "fail") {
		t.Fatalf("expected invalid port failure: %+v", result.Checks)
	}
}

func TestDoctorConnectionDirectWithProxyFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "direct",
  "host": "127.0.0.1",
  "port": 3306,
  "user": "root",
  "proxy": {"url": "socks5://proxy_user:proxy_password@127.0.0.1:1080"}
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr == nil {
		t.Fatalf("expected failure")
	}
	if !hasDoctorCheck(result, "proxy config must be empty for direct mode", "fail") {
		t.Fatalf("expected direct proxy failure: %+v", result.Checks)
	}
}

func TestDoctorConnectionProxySkipsSSHChecks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "proxy",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "proxy": {"url": "socks5://proxy_user:proxy_password@127.0.0.1:1080"}
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr != nil {
		t.Fatalf("doctorConnection returned error: %v", doctorErr)
	}
	if !hasDoctorCheck(result, "proxy scheme socks5", "ok") {
		t.Fatalf("expected proxy checks: %+v", result.Checks)
	}
	for _, check := range result.Checks {
		if strings.HasPrefix(check.Name, "ssh ") {
			t.Fatalf("proxy mode should skip ssh checks: %+v", result.Checks)
		}
	}
}

func TestDoctorConnectionProxyRejectsSSHConfig(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "proxy",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "proxy": {"url": "socks5://127.0.0.1:1080"},
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  }
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr == nil {
		t.Fatalf("expected failure")
	}
	if !hasDoctorCheck(result, "ssh config must be empty for proxy mode", "fail") {
		t.Fatalf("expected proxy ssh config failure: %+v", result.Checks)
	}
}

func TestDoctorConnectionSSHMissingAuthFails(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu"
  }
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr == nil {
		t.Fatalf("expected failure")
	}
	if !hasDoctorCheck(result, "ssh auth method is configured", "fail") {
		t.Fatalf("expected ssh auth failure: %+v", result.Checks)
	}
}

func TestDoctorConnectionSSHPrivateKeyMissingFails(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  }
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr == nil {
		t.Fatalf("expected failure")
	}
	if !hasDoctorCheckPrefix(result, "ssh private key exists ", "fail") {
		t.Fatalf("expected missing private key failure: %+v", result.Checks)
	}
}

func TestDoctorConnectionSSHPrivateKeyPermissionWarns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission warning is unix-specific")
	}

	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(keyDir, "id_rsa")
	if err := os.WriteFile(keyPath, []byte("key"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  }
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr != nil {
		t.Fatalf("doctorConnection returned error: %v", doctorErr)
	}
	if !hasDoctorCheck(result, "ssh private key permissions are strict", "warn") {
		t.Fatalf("expected private key permission warning: %+v", result.Checks)
	}
}

func TestDoctorConnectionProxyUnsupportedSchemeFails(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "proxy-ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "proxy": {"url": "http://proxy_user:proxy_password@127.0.0.1:1080"},
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  }
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr == nil {
		t.Fatalf("expected failure")
	}
	if !hasDoctorCheck(result, "proxy scheme is supported", "fail") {
		t.Fatalf("expected unsupported proxy scheme failure: %+v", result.Checks)
	}
	if containsDoctorSecret(result, "proxy_password") {
		t.Fatalf("doctor result leaked proxy password: %+v", result)
	}
}

func TestDoctorConnectionKnownHostsWarnings(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  }
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keyDir, "id_rsa"), []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, doctorErr := app.doctorConnection("prod")
	if doctorErr != nil {
		t.Fatalf("doctorConnection returned error: %v", doctorErr)
	}
	if !hasDoctorCheck(result, "known_hosts file not found", "warn") {
		t.Fatalf("expected known_hosts missing warning: %+v", result.Checks)
	}

	if err := os.WriteFile(filepath.Join(keyDir, "known_hosts"), []byte("other.example.com ssh-ed25519 AAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, doctorErr = app.doctorConnection("prod")
	if doctorErr != nil {
		t.Fatalf("doctorConnection returned error: %v", doctorErr)
	}
	if !hasDoctorCheck(result, "known_hosts entry missing for bastion.example.com", "warn") {
		t.Fatalf("expected known_hosts host warning: %+v", result.Checks)
	}
}

func TestCLIConnectionDoctorParsesName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	connector := &diagnosticConnector{}
	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: connector,
	})
	err := app.Run(context.Background(), []string{"doctor", "connection", "prod", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if connector.openCalls != 0 {
		t.Fatalf("doctor should not open network connections")
	}
	if !strings.Contains(stdout.String(), "Connection doctor: prod") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestCLIConnectionDoctorJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "direct",
  "host": "127.0.0.1",
  "port": 3306,
  "user": "root",
  "password": "super-secret"
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"doctor", "connection", "prod", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}

	var result DoctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("warnings should not set ok=false: %+v", result)
	}
	if !hasDoctorCheck(&result, "database password is inline in config", "warn") {
		t.Fatalf("expected inline password warning: %+v", result.Checks)
	}
	if containsDoctorSecret(&result, "super-secret") {
		t.Fatalf("doctor json leaked secret: %s", stdout.String())
	}
}

func TestCLIConnectionDoctorJSONFailNonZero(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	raw := `{
  "name": "prod",
  "driver": "mysql",
  "mode": "direct",
  "host": "127.0.0.1",
  "port": 70000,
  "user": "root"
}`
	writeRawConnectionConfig(t, store, "prod", raw)

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"doctor", "connection", "prod", "--format", "json", "--config-dir", root})
	if err == nil {
		t.Fatalf("expected non-zero failure")
	}
	if app.ExitStatus() == 0 {
		t.Fatalf("expected non-zero exit status")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var result DoctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if result.OK {
		t.Fatalf("expected ok=false: %+v", result)
	}
	if !hasDoctorCheck(&result, "database port 70000", "fail") {
		t.Fatalf("expected failing check: %+v", result.Checks)
	}
}

func hasDoctorCheck(result *DoctorResult, name string, status string) bool {
	for _, check := range result.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func hasDoctorCheckPrefix(result *DoctorResult, prefix string, status string) bool {
	for _, check := range result.Checks {
		if strings.HasPrefix(check.Name, prefix) && check.Status == status {
			return true
		}
	}
	return false
}

func containsDoctorSecret(result *DoctorResult, secret string) bool {
	if secret == "" {
		return false
	}
	data, err := json.Marshal(result)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), secret)
}

func writeRawConnectionConfig(t *testing.T, store *config.Store, name string, content string) {
	t.Helper()
	path := store.ConnectionConfigPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
