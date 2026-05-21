package editor

import "testing"

func TestEditorInsertInMiddle(t *testing.T) {
	t.Parallel()

	editor := New()
	editor.SetText("abcd")
	editor.MoveLeft()
	editor.MoveLeft()
	editor.InsertRune('X')

	if got := editor.CurrentLine(); got != "abXcd" {
		t.Fatalf("buffer = %q", got)
	}
	if got := editor.Cursor(); got != 3 {
		t.Fatalf("cursor = %d, want 3", got)
	}
}

func TestEditorCursorMovementAndHomeEnd(t *testing.T) {
	t.Parallel()

	editor := New()
	editor.SetText("select")
	editor.MoveLeft()
	editor.MoveLeft()
	if got := editor.Cursor(); got != 4 {
		t.Fatalf("cursor after left = %d, want 4", got)
	}
	editor.MoveHome()
	if got := editor.Cursor(); got != 0 {
		t.Fatalf("cursor after home = %d, want 0", got)
	}
	editor.MoveEnd()
	if got := editor.Cursor(); got != len([]rune("select")) {
		t.Fatalf("cursor after end = %d", got)
	}
}

func TestEditorBackspaceAndDelete(t *testing.T) {
	t.Parallel()

	editor := New()
	editor.SetText("abcde")
	editor.MoveLeft()
	editor.MoveLeft()
	editor.DeleteBackward()
	if got := editor.CurrentLine(); got != "abde" {
		t.Fatalf("buffer after backspace = %q", got)
	}
	editor.DeleteForward()
	if got := editor.CurrentLine(); got != "abe" {
		t.Fatalf("buffer after delete = %q", got)
	}
}

func TestEditorUTF8AndFullWidthDelete(t *testing.T) {
	t.Parallel()

	editor := New()
	editor.SetText("你好界")
	editor.MoveLeft()
	editor.InsertRune('世')
	if got := editor.CurrentLine(); got != "你好世界" {
		t.Fatalf("buffer = %q", got)
	}
	editor.DeleteBackward()
	if got := editor.CurrentLine(); got != "你好界" {
		t.Fatalf("buffer after backspace = %q", got)
	}

	editor.SetText("A，B")
	editor.MoveLeft()
	editor.MoveLeft()
	editor.DeleteForward()
	if got := editor.CurrentLine(); got != "AB" {
		t.Fatalf("buffer after full-width delete = %q", got)
	}
	if got := editor.Cursor(); got != 1 {
		t.Fatalf("cursor after full-width delete = %d, want 1", got)
	}
}
