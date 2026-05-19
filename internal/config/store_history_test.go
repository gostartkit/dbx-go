package config

import "testing"

func TestHistoryPersistence(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	if err := store.AppendHistory("connect"); err != nil {
		t.Fatalf("AppendHistory returned error: %v", err)
	}
	if err := store.AppendHistory("status"); err != nil {
		t.Fatalf("AppendHistory returned error: %v", err)
	}

	history, err := store.LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory returned error: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("history length = %d, want 2", len(history))
	}
	if history[0] != "connect" || history[1] != "status" {
		t.Fatalf("history = %#v", history)
	}
}
