package editor

import "testing"

func TestHistoryNavigatorNavigation(t *testing.T) {
	t.Parallel()

	navigator := NewHistoryNavigator([]string{"connect", "list databases", "status"})
	if got := navigator.Up("conn"); got != "status" {
		t.Fatalf("Up() = %q", got)
	}
	if got := navigator.Up("status"); got != "list databases" {
		t.Fatalf("second Up() = %q", got)
	}
	if got := navigator.Down("list databases"); got != "status" {
		t.Fatalf("Down() = %q", got)
	}
	if got := navigator.Down("status"); got != "conn" {
		t.Fatalf("Down() to draft = %q", got)
	}
}

func TestHistoryNavigatorAddSkipsEmptyAndDuplicate(t *testing.T) {
	t.Parallel()

	navigator := NewHistoryNavigator([]string{"connect"})
	if added := navigator.Add(""); added {
		t.Fatalf("Add(empty) = true")
	}
	if added := navigator.Add("connect"); added {
		t.Fatalf("Add(duplicate) = true")
	}
	if added := navigator.Add("status"); !added {
		t.Fatalf("Add(status) = false")
	}
}
