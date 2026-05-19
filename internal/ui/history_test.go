package ui

import "testing"

func TestHistoryNavigatorNavigation(t *testing.T) {
	t.Parallel()

	navigator := NewHistoryNavigator([]string{"connect", "list databases", "status"})

	if got := navigator.Up("conn"); got != "status" {
		t.Fatalf("Up() = %q, want %q", got, "status")
	}
	if got := navigator.Up("status"); got != "list databases" {
		t.Fatalf("second Up() = %q, want %q", got, "list databases")
	}
	if got := navigator.Down("list databases"); got != "status" {
		t.Fatalf("Down() = %q, want %q", got, "status")
	}
	if got := navigator.Down("status"); got != "conn" {
		t.Fatalf("Down() to draft = %q, want %q", got, "conn")
	}
}

func TestHistoryNavigatorEmptyHistoryKeepsDraft(t *testing.T) {
	t.Parallel()

	navigator := NewHistoryNavigator(nil)
	if got := navigator.Up("draft"); got != "draft" {
		t.Fatalf("Up() = %q, want %q", got, "draft")
	}
	if got := navigator.Down("draft"); got != "draft" {
		t.Fatalf("Down() = %q, want %q", got, "draft")
	}
}

func TestHistoryNavigatorAddSkipsEmptyAndDuplicate(t *testing.T) {
	t.Parallel()

	navigator := NewHistoryNavigator([]string{"connect"})
	if added := navigator.Add(""); added {
		t.Fatalf("Add(empty) = true, want false")
	}
	if added := navigator.Add("connect"); added {
		t.Fatalf("Add(duplicate) = true, want false")
	}
	if added := navigator.Add("status"); !added {
		t.Fatalf("Add(status) = false, want true")
	}

	entries := navigator.Entries()
	if len(entries) != 2 || entries[0] != "connect" || entries[1] != "status" {
		t.Fatalf("Entries() = %#v", entries)
	}
}
