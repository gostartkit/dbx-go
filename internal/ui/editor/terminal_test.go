package editor

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestTerminalPrintlnAndPrintfClearLineOnTerminal(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &out, nil)
	terminal.isTerm = func() bool { return true }

	terminal.Println("validation error")
	terminal.Printf("Error: %s\n", "broken")

	want := "\r\033[2Kvalidation error\n\r\033[2KError: broken\n"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestTerminalPrintSystemOutputRedrawsCurrentBuffer(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &out, nil)
	terminal.isTerm = func() bool { return true }
	terminal.rawActive = true
	terminal.prompt = "dbx(prod)> "
	terminal.editor.SetText("use db1")

	terminal.PrintSystemOutput(func(w io.Writer) {
		w.Write([]byte("db1\n"))
		w.Write([]byte("db2\n\n"))
	})

	want := "\r\033[2K\r\ndb1\r\ndb2\r\n\r\n\r\033[2Kdbx(prod)> use db1"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestTerminalApplyCompletionKeepsSuffixAndCursor(t *testing.T) {
	t.Parallel()

	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, nil)
	terminal.completer = func(request CompletionRequest) Completion {
		if request.CurrentLinePrefix() != "show us" {
			t.Fatalf("completion prefix = %q", request.CurrentLinePrefix())
		}
		return Completion{
			Suggestions: []Suggestion{
				{
					Value: "users",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 7, Text: "users"}},
						Cursor: len([]rune("show users")),
					},
				},
			},
		}
	}
	terminal.editor.SetText("show us where id = 1")
	terminal.editor.MoveHome()
	for i := 0; i < len([]rune("show us")); i++ {
		terminal.editor.MoveRight()
	}

	if !terminal.applyCompletion() {
		t.Fatalf("expected completion to apply")
	}
	if got := terminal.editor.CurrentLine(); got != "show users where id = 1" {
		t.Fatalf("line = %q", got)
	}
	if got := terminal.editor.Cursor(); got != len([]rune("show users")) {
		t.Fatalf("cursor = %d", got)
	}
}
