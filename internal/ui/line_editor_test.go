package ui

import "testing"

func TestLineEditorInsertInMiddle(t *testing.T) {
	t.Parallel()

	editor := newLineEditor(nil)
	editor.SetLine("abcd")
	editor.HandleKey(keyEvent{kind: keyLeft})
	editor.HandleKey(keyEvent{kind: keyLeft})

	result := editor.HandleKey(keyEvent{kind: keyRune, r: 'X'})
	if !result.changed {
		t.Fatalf("expected insert to change buffer")
	}
	if got := editor.String(); got != "abXcd" {
		t.Fatalf("buffer = %q", got)
	}
	if got := editor.Cursor(); got != 3 {
		t.Fatalf("cursor = %d, want 3", got)
	}
}

func TestLineEditorCursorMovementAndHomeEnd(t *testing.T) {
	t.Parallel()

	editor := newLineEditor(nil)
	editor.SetLine("select")

	editor.HandleKey(keyEvent{kind: keyLeft})
	editor.HandleKey(keyEvent{kind: keyLeft})
	if got := editor.Cursor(); got != 4 {
		t.Fatalf("cursor after left = %d, want 4", got)
	}

	editor.HandleKey(keyEvent{kind: keyHome})
	if got := editor.Cursor(); got != 0 {
		t.Fatalf("cursor after home = %d, want 0", got)
	}

	editor.HandleKey(keyEvent{kind: keyEnd})
	if got := editor.Cursor(); got != len([]rune("select")) {
		t.Fatalf("cursor after end = %d", got)
	}

	editor.HandleKey(keyEvent{kind: keyCtrlA})
	if got := editor.Cursor(); got != 0 {
		t.Fatalf("cursor after Ctrl+A = %d, want 0", got)
	}

	editor.HandleKey(keyEvent{kind: keyCtrlE})
	if got := editor.Cursor(); got != len([]rune("select")) {
		t.Fatalf("cursor after Ctrl+E = %d", got)
	}
}

func TestLineEditorBackspaceAndDelete(t *testing.T) {
	t.Parallel()

	editor := newLineEditor(nil)
	editor.SetLine("abcde")
	editor.HandleKey(keyEvent{kind: keyLeft})
	editor.HandleKey(keyEvent{kind: keyLeft})

	editor.HandleKey(keyEvent{kind: keyBackspace})
	if got := editor.String(); got != "abde" {
		t.Fatalf("buffer after backspace = %q, want %q", got, "abde")
	}
	if got := editor.Cursor(); got != 2 {
		t.Fatalf("cursor after backspace = %d, want 2", got)
	}

	editor.HandleKey(keyEvent{kind: keyDelete})
	if got := editor.String(); got != "abe" {
		t.Fatalf("buffer after delete = %q, want %q", got, "abe")
	}
	if got := editor.Cursor(); got != 2 {
		t.Fatalf("cursor after delete = %d, want 2", got)
	}
}

func TestLineEditorUTF8InsertMoveAndDelete(t *testing.T) {
	t.Parallel()

	editor := newLineEditor(nil)
	editor.SetLine("你好界")
	editor.HandleKey(keyEvent{kind: keyLeft})
	editor.HandleKey(keyEvent{kind: keyRune, r: '世'})
	if got := editor.String(); got != "你好世界" {
		t.Fatalf("buffer = %q, want %q", got, "你好世界")
	}

	editor.HandleKey(keyEvent{kind: keyBackspace})
	if got := editor.String(); got != "你好界" {
		t.Fatalf("buffer after UTF-8 backspace = %q, want %q", got, "你好界")
	}

	editor.HandleKey(keyEvent{kind: keyHome})
	editor.HandleKey(keyEvent{kind: keyDelete})
	if got := editor.String(); got != "好界" {
		t.Fatalf("buffer after UTF-8 delete = %q, want %q", got, "好界")
	}
}

func TestLineEditorHistoryNavigation(t *testing.T) {
	t.Parallel()

	editor := newLineEditor(NewHistoryNavigator([]string{"show databases", "show tables"}))
	editor.SetLine("show ")

	editor.HandleKey(keyEvent{kind: keyUp})
	if got := editor.String(); got != "show tables" {
		t.Fatalf("history up 1 = %q", got)
	}

	editor.HandleKey(keyEvent{kind: keyUp})
	if got := editor.String(); got != "show databases" {
		t.Fatalf("history up 2 = %q", got)
	}

	editor.HandleKey(keyEvent{kind: keyDown})
	if got := editor.String(); got != "show tables" {
		t.Fatalf("history down 1 = %q", got)
	}

	editor.HandleKey(keyEvent{kind: keyDown})
	if got := editor.String(); got != "show " {
		t.Fatalf("history down 2 = %q", got)
	}
}

func TestLineEditorCtrlCAndCtrlD(t *testing.T) {
	t.Parallel()

	editor := newLineEditor(nil)
	cancel := editor.HandleKey(keyEvent{kind: keyCtrlC})
	if !cancel.cancel {
		t.Fatalf("expected Ctrl+C to cancel")
	}

	eof := editor.HandleKey(keyEvent{kind: keyCtrlD})
	if !eof.eof {
		t.Fatalf("expected Ctrl+D on empty buffer to signal EOF")
	}

	editor.SetLine("abc")
	editor.cursor = 1
	changed := editor.HandleKey(keyEvent{kind: keyCtrlD})
	if !changed.changed {
		t.Fatalf("expected Ctrl+D with content to delete at cursor")
	}
	if got := editor.String(); got != "ac" {
		t.Fatalf("buffer after Ctrl+D delete = %q, want %q", got, "ac")
	}
}
