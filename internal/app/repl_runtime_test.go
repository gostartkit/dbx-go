package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestApplicationRunUsesCmdPromptAndHistoryHooks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	application, err := NewWithOptions(strings.NewReader("quit\n"), &stdout, &stderr, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	defer application.Close()

	application.session.Connection = sampleConnection("prod")
	application.dryRun = true

	if err := application.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dbx[prod][disconnected][dry-run]> ") {
		t.Fatalf("expected dynamic prompt in stdout, got %q", stdout.String())
	}

	store := config.NewStore(root)
	history, err := store.LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory returned error: %v", err)
	}
	if len(history) != 1 || history[0] != "quit" {
		t.Fatalf("history = %#v, want [quit]", history)
	}
}
