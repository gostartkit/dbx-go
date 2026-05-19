package config

import (
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
