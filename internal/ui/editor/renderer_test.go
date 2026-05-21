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

func TestBuildRenderFrameSupportsContinuationPrompt(t *testing.T) {
	t.Parallel()

	buffer := NewBufferFromString("exec grant-readonly\n--user alice")
	frame := BuildRenderFrame("dbx> ", "... ", buffer, Position{Line: 1, Column: len([]rune("--user alice"))})
	if len(frame.Lines) != 2 {
		t.Fatalf("frame lines = %d", len(frame.Lines))
	}
	if string(cellsToRunes(frame.Lines[0].Cells)) != "dbx> exec grant-readonly" {
		t.Fatalf("line 0 = %q", string(cellsToRunes(frame.Lines[0].Cells)))
	}
	if string(cellsToRunes(frame.Lines[1].Cells)) != "... --user alice" {
		t.Fatalf("line 1 = %q", string(cellsToRunes(frame.Lines[1].Cells)))
	}
	if frame.CursorRow != 1 {
		t.Fatalf("cursor row = %d", frame.CursorRow)
	}
}

func cellsToRunes(cells []Cell) []rune {
	runes := make([]rune, 0, len(cells))
	for _, cell := range cells {
		runes = append(runes, cell.Rune)
	}
	return runes
}
