package editor

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/commandlang"
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

func TestTerminalMultilineContinuationBuildsLogicalCommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &out, nil)
	terminal.rawActive = true
	terminal.prompt = "dbx> "
	terminal.editor.SetText("exec grant-readonly \\")

	terminal.editor.setCurrentLine(trimContinuationMarker([]rune(terminal.editor.CurrentLine())))
	terminal.editor.AppendLine()
	terminal.editor.InsertRune('-')
	terminal.editor.InsertRune('-')
	terminal.editor.InsertRune('u')
	terminal.editor.InsertRune('s')
	terminal.editor.InsertRune('e')
	terminal.editor.InsertRune('r')
	terminal.editor.InsertRune(' ')
	terminal.editor.InsertRune('a')
	terminal.editor.InsertRune('l')
	terminal.editor.InsertRune('i')
	terminal.editor.InsertRune('c')
	terminal.editor.InsertRune('e')

	got := commandlang.JoinLogicalLines(terminal.editor.Text())
	if got != "exec grant-readonly --user alice" {
		t.Fatalf("logical command = %q", got)
	}
}

func TestTerminalCompletionCycleReplacesOriginalToken(t *testing.T) {
	t.Parallel()

	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, nil)
	terminal.completer = func(request CompletionRequest) Completion {
		return Completion{
			Suggestions: []Suggestion{
				{
					Value: "columns",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "columns"}},
						Cursor: len([]rune("show columns")),
					},
				},
				{
					Value: "connections",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "connections"}},
						Cursor: len([]rune("show connections")),
					},
				},
				{
					Value: "config",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "config"}},
						Cursor: len([]rune("show config")),
					},
				},
			},
		}
	}
	terminal.editor.SetText("show col")

	if !terminal.applyCompletion() {
		t.Fatalf("expected first completion")
	}
	if got := terminal.editor.CurrentLine(); got != "show co" {
		t.Fatalf("first line = %q", got)
	}
	if terminal.applyCompletion() {
		t.Fatalf("second tab should only print suggestions")
	}
	if !terminal.applyCompletion() {
		t.Fatalf("third tab should apply first suggestion")
	}
	if got := terminal.editor.CurrentLine(); got != "show columns" {
		t.Fatalf("third tab line = %q", got)
	}
	if !terminal.applyCompletion() {
		t.Fatalf("fourth tab should cycle to next suggestion")
	}
	if got := terminal.editor.CurrentLine(); got != "show connections" {
		t.Fatalf("fourth tab line = %q", got)
	}
	if strings.Contains(terminal.editor.CurrentLine(), "connectioncolumns") {
		t.Fatalf("unexpected concatenated completion: %q", terminal.editor.CurrentLine())
	}
	if !terminal.applyCompletion() {
		t.Fatalf("fifth tab should cycle to next suggestion")
	}
	if got := terminal.editor.CurrentLine(); got != "show config" {
		t.Fatalf("fifth tab line = %q", got)
	}
}

func TestTerminalCompletionCyclePreservesSuffix(t *testing.T) {
	t.Parallel()

	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, nil)
	terminal.completer = func(request CompletionRequest) Completion {
		return Completion{
			Suggestions: []Suggestion{
				{
					Value: "columns",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "columns"}},
						Cursor: len([]rune("show columns")),
					},
				},
				{
					Value: "collectors",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "collectors"}},
						Cursor: len([]rune("show collectors")),
					},
				},
			},
		}
	}
	terminal.editor.SetText("show col --format json")
	terminal.editor.MoveHome()
	for i := 0; i < len([]rune("show col")); i++ {
		terminal.editor.MoveRight()
	}

	if !terminal.applyCompletion() {
		t.Fatalf("expected first completion")
	}
	if got := terminal.editor.CurrentLine(); got != "show columns --format json" {
		t.Fatalf("line = %q", got)
	}
	if terminal.applyCompletion() {
		t.Fatalf("second tab should only print suggestions")
	}
	if !terminal.applyCompletion() {
		t.Fatalf("third tab should apply first suggestion")
	}
	if got := terminal.editor.CurrentLine(); got != "show collectors --format json" {
		t.Fatalf("cycled line = %q", got)
	}
	if !terminal.applyCompletion() {
		t.Fatalf("fourth tab should cycle to next suggestion")
	}
	if got := terminal.editor.CurrentLine(); got != "show columns --format json" {
		t.Fatalf("cycled line = %q", got)
	}
}

func TestTerminalCompletionSessionResetsOnRuneInput(t *testing.T) {
	t.Parallel()

	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, nil)
	terminal.completer = func(request CompletionRequest) Completion {
		return Completion{
			Suggestions: []Suggestion{
				{
					Value: "columns",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "columns"}},
						Cursor: len([]rune("show columns")),
					},
				},
				{
					Value: "connections",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "connections"}},
						Cursor: len([]rune("show connections")),
					},
				},
			},
		}
	}
	terminal.editor.SetText("show col")
	if !terminal.applyCompletion() {
		t.Fatalf("expected completion")
	}
	if terminal.session == nil {
		t.Fatalf("expected active completion session")
	}

	done, line, err := terminal.handleEvent(KeyEvent{Type: KeyRune, Rune: 'x'})
	if done || line != "" || err != nil {
		t.Fatalf("handleEvent returned done=%v line=%q err=%v", done, line, err)
	}
	if terminal.session != nil {
		t.Fatalf("expected completion session reset after rune input")
	}
}

func TestTerminalCompletionSessionResetsOnCursorMove(t *testing.T) {
	t.Parallel()

	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, nil)
	terminal.completer = func(request CompletionRequest) Completion {
		return Completion{
			Suggestions: []Suggestion{
				{
					Value: "columns",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "columns"}},
						Cursor: len([]rune("show columns")),
					},
				},
				{
					Value: "connections",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "connections"}},
						Cursor: len([]rune("show connections")),
					},
				},
			},
		}
	}
	terminal.editor.SetText("show col")
	if !terminal.applyCompletion() {
		t.Fatalf("expected completion")
	}
	if terminal.session == nil {
		t.Fatalf("expected active completion session")
	}

	done, line, err := terminal.handleEvent(KeyEvent{Type: KeyLeft})
	if done || line != "" || err != nil {
		t.Fatalf("handleEvent returned done=%v line=%q err=%v", done, line, err)
	}
	if terminal.session != nil {
		t.Fatalf("expected completion session reset after cursor move")
	}
}

func TestTerminalCommonPrefixDoesNotAppendOnCycle(t *testing.T) {
	t.Parallel()

	terminal := NewTerminal(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, nil)
	terminal.completer = func(request CompletionRequest) Completion {
		return Completion{
			Suggestions: []Suggestion{
				{
					Value: "connection",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "connection"}},
						Cursor: len([]rune("show connection")),
					},
				},
				{
					Value: "connections",
					Result: CompletionResult{
						Edits:  []CompletionEdit{{StartRune: 5, EndRune: 8, Text: "connections"}},
						Cursor: len([]rune("show connections")),
					},
				},
			},
		}
	}
	terminal.editor.SetText("show con")

	if !terminal.applyCompletion() {
		t.Fatalf("expected common prefix completion")
	}
	if got := terminal.editor.CurrentLine(); got != "show connection" {
		t.Fatalf("common prefix line = %q", got)
	}
	if terminal.applyCompletion() {
		t.Fatalf("second tab should only print suggestions")
	}
	if !terminal.applyCompletion() {
		t.Fatalf("third tab should cycle to suggestion")
	}
	if got := terminal.editor.CurrentLine(); got != "show connections" {
		t.Fatalf("cycled line = %q", got)
	}
	if strings.Contains(terminal.editor.CurrentLine(), "connectionconnection") {
		t.Fatalf("unexpected concatenated completion: %q", terminal.editor.CurrentLine())
	}
}
