package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultRootDirAndStorePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root, err := DefaultRootDir()
	if err != nil {
		t.Fatalf("DefaultRootDir returned error: %v", err)
	}

	wantRoot := filepath.Join(home, ".config", "dbx")
	if root != wantRoot {
		t.Fatalf("DefaultRootDir = %q, want %q", root, wantRoot)
	}

	store := NewStore(root)
	if got := store.ConnectionConfigPath("prod"); got != filepath.Join(wantRoot, "prod", "config.json") {
		t.Fatalf("ConnectionConfigPath = %q", got)
	}
	if got := store.ConnectionTemplatesDir("prod"); got != filepath.Join(wantRoot, "prod", "templates") {
		t.Fatalf("ConnectionTemplatesDir = %q", got)
	}
	if got := store.GlobalTemplatesDir(); got != filepath.Join(wantRoot, "templates") {
		t.Fatalf("GlobalTemplatesDir = %q", got)
	}
	if got := store.AuditLogPath(); got != filepath.Join(wantRoot, "logs", "audit.jsonl") {
		t.Fatalf("AuditLogPath = %q", got)
	}
}

func TestSaveLoadAndDeleteConnection(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	cfg := &ConnectionConfig{
		Name:           "prod",
		Driver:         "mysql",
		Mode:           "proxy-ssh",
		Host:           "10.0.1.20",
		Port:           3306,
		User:           "root",
		PasswordEnv:    "MYSQL_PROD_PASSWORD",
		PasswordPrompt: false,
		Proxy: &ProxyConfig{
			URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080",
		},
		SSH: &SSHConfig{
			Host:       "bastion.example.com",
			Port:       22,
			User:       "ubuntu",
			PrivateKey: "~/.ssh/id_rsa",
		},
	}

	if err := store.SaveConnection(cfg); err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	loaded, err := store.LoadConnection("prod")
	if err != nil {
		t.Fatalf("LoadConnection returned error: %v", err)
	}

	if loaded.Name != cfg.Name || loaded.PasswordEnv != cfg.PasswordEnv {
		t.Fatalf("loaded config = %#v", loaded)
	}
	if loaded.Version != CurrentConnectionSchemaVersion {
		t.Fatalf("loaded version = %d", loaded.Version)
	}
	if loaded.Proxy == nil || loaded.Proxy.URL != cfg.Proxy.URL {
		t.Fatalf("loaded proxy config = %#v", loaded.Proxy)
	}
	if loaded.SSH == nil || loaded.SSH.Host != cfg.SSH.Host {
		t.Fatalf("loaded SSH config = %#v", loaded.SSH)
	}

	if err := store.DeleteConnection("prod"); err != nil {
		t.Fatalf("DeleteConnection returned error: %v", err)
	}

	_, err = store.LoadConnection("prod")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LoadConnection after delete error = %v, want os.ErrNotExist", err)
	}
}

func TestListConnectionRecordsIncludesInvalidConnections(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	valid := &ConnectionConfig{
		Name:        "prod",
		Driver:      "mysql",
		Mode:        "direct",
		Host:        "127.0.0.1",
		Port:        3306,
		User:        "root",
		PasswordEnv: "MYSQL_PROD_PASSWORD",
	}
	if err := store.SaveConnection(valid); err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	invalidPath := store.ConnectionConfigPath("broken")
	if err := os.MkdirAll(filepath.Dir(invalidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidPath, []byte(`{"name":"broken","driver":"mysql","mode":"direct","host":"127.0.0.1","port":70000,"user":"root"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := store.ListConnectionRecords()
	if err != nil {
		t.Fatalf("ListConnectionRecords returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("record count = %d, want 2", len(records))
	}

	if records[0].Name != "broken" || records[0].Error == nil {
		t.Fatalf("unexpected invalid record: %+v", records[0])
	}
	if !strings.Contains(records[0].Error.Error(), "port must be greater than zero") {
		t.Fatalf("unexpected invalid record error: %v", records[0].Error)
	}
	if records[1].Name != "prod" || records[1].Error != nil || records[1].Config == nil {
		t.Fatalf("unexpected valid record: %+v", records[1])
	}

	connections, err := store.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections returned error: %v", err)
	}
	if len(connections) != 1 || connections[0].Name != "prod" {
		t.Fatalf("unexpected valid connections: %+v", connections)
	}
}

func TestLoadConnectionRecordPreservesParseErrors(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	path := store.ConnectionConfigPath("broken")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"name":"broken",`), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := store.LoadConnectionRecord("broken")
	if err != nil {
		t.Fatalf("LoadConnectionRecord returned error: %v", err)
	}
	if record.Name != "broken" || record.Config != nil || record.Error == nil {
		t.Fatalf("unexpected record: %+v", record)
	}
	if !strings.Contains(record.Error.Error(), "unexpected end of JSON input") {
		t.Fatalf("unexpected parse error: %v", record.Error)
	}
}

func TestAppendAndLoadAudit(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	record := &AuditRecord{
		Timestamp:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		Command:    "create database",
		Connection: "prod",
		Mode:       "proxy-ssh",
		DryRun:     false,
		Success:    true,
		DurationMS: 123,
	}
	if err := store.AppendAudit(record); err != nil {
		t.Fatalf("AppendAudit returned error: %v", err)
	}

	loaded, err := store.LoadAudit(10)
	if err != nil {
		t.Fatalf("LoadAudit returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("audit entries = %d", len(loaded))
	}
	if loaded[0].Command != "create database" || loaded[0].Connection != "prod" {
		t.Fatalf("unexpected audit entry: %+v", loaded[0])
	}

	data, err := os.ReadFile(store.AuditLogPath())
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	var parsed AuditRecord
	if err := json.Unmarshal(data[:len(data)-1], &parsed); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if parsed.Mode != "proxy-ssh" || parsed.DurationMS != 123 {
		t.Fatalf("unexpected parsed audit record: %+v", parsed)
	}
}

func TestSaveAndLoadSessionWithDatabase(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	session := &SessionFile{
		CurrentConnection: "prod",
		CurrentDatabase:   "app_prod",
	}
	if err := store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	loaded, err := store.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}
	if loaded.CurrentConnection != "prod" || loaded.CurrentDatabase != "app_prod" {
		t.Fatalf("unexpected session: %+v", loaded)
	}

	data, err := os.ReadFile(store.SessionPath())
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "{\n  \"connection\": \"prod\",\n  \"database\": \"app_prod\"\n}\n" {
		t.Fatalf("unexpected session file: %q", string(data))
	}
}
