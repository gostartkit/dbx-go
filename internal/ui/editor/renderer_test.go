package editor

import "testing"

func TestRenderSingleLineRestoresCursorByVisualWidth(t *testing.T) {
	t.Parallel()

	buffer := NewBufferFromString("你a好")
	got := RenderSingleLine("dbx> ", buffer, Position{Line: 0, Column: 1})
	want := "\r\033[2Kdbx> 你a好\033[3D"
	if got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

func TestRenderSingleLineHandlesCombiningMarks(t *testing.T) {
	t.Parallel()

	buffer := NewBufferFromString("e\u0301x")
	got := RenderSingleLine("dbx> ", buffer, Position{Line: 0, Column: 2})
	want := "\r\033[2Kdbx> e\u0301x\033[1D"
	if got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}
