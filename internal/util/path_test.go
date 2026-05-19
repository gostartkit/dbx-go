package util

import (
	"path/filepath"
	"testing"
)

func TestExpandHomeAndConfigPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	expanded, err := ExpandHome("~/known_hosts")
	if err != nil {
		t.Fatalf("ExpandHome returned error: %v", err)
	}

	want := filepath.Join(home, "known_hosts")
	if expanded != want {
		t.Fatalf("ExpandHome = %q, want %q", expanded, want)
	}
}
