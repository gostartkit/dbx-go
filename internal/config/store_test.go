package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
