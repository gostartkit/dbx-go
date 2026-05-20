package ui

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestPromptPrintlnClearsCurrentLineOnTerminal(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)
	prompt.isTerm = func() bool { return true }

	prompt.Println("Error: validation error: validate database name: invalid database name \"greenhn-dev\"")

	if got := out.String(); got != "\r\033[2KError: validation error: validate database name: invalid database name \"greenhn-dev\"\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestPromptPrintfClearsCurrentLineOnTerminal(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)
	prompt.isTerm = func() bool { return true }

	prompt.Printf("Error: %s\n", "validation error")

	if got := out.String(); got != "\r\033[2KError: validation error\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestPromptRedrawLineIgnoresHintForStableRendering(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)

	prompt.redrawLine("dbx> ", "crea", "te database")

	want := "\r\033[2Kdbx> crea"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestPromptPrintSuggestionsWithDescriptions(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)

	prompt.printSuggestions([]Suggestion{
		{Value: "prod", Description: "mysql proxy-ssh"},
		{Value: "dev", Description: "mysql direct"},
	})

	if got := out.String(); !strings.Contains(got, "prod  mysql proxy-ssh") || !strings.Contains(got, "dev") {
		t.Fatalf("output = %q", got)
	}
}

func TestPromptPrintSystemOutputClearsLineAndRedrawsInRawMode(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)
	prompt.isTerm = func() bool { return true }
	prompt.rawActive = true
	prompt.label = "dbx(prod)> "
	prompt.current = "connec"

	prompt.PrintSystemOutput(func(w io.Writer) {
		w.Write([]byte("connect             connect to a saved connection\n"))
		w.Write([]byte("connections         list saved connections\n\n"))
	})

	want := "\r\033[2K\r\nconnect             connect to a saved connection\r\nconnections         list saved connections\r\n\r\n\r\033[2Kdbx(prod)> connec"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestPromptApplyCompletionCyclesSuggestions(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.completer = func(string) Completion {
		return Completion{
			Prefix: "conn",
			Suggestions: []Suggestion{
				{Value: "connect", Description: "connect to a saved connection", Replacement: "connect", ReplaceFrom: 0, ReplaceTo: 4},
				{Value: "connections", Description: "list saved connections", Replacement: "connections", ReplaceFrom: 0, ReplaceTo: 4},
				{Value: "connection create", Description: "create a saved connection", Replacement: "connection create", ReplaceFrom: 0, ReplaceTo: 4},
			},
		}
	}

	if got := prompt.applyCompletion("conn"); got != "connect" {
		t.Fatalf("first completion = %q", got)
	}
	if got := prompt.applyCompletion("connect"); got != "connect" {
		t.Fatalf("second completion should keep current input while listing candidates, got %q", got)
	}
	if got := prompt.applyCompletion("connect"); got != "connections" {
		t.Fatalf("third completion = %q", got)
	}
	if got := prompt.applyCompletion("connections"); got != "connection create" {
		t.Fatalf("fourth completion = %q", got)
	}
}

func TestPromptApplyCompletionSingleCandidateDoesNotPrintList(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)
	prompt.completer = func(string) Completion {
		return Completion{
			Prefix: "connec",
			Suggestions: []Suggestion{
				{Value: "connect", Description: "connect to a saved connection", Replacement: "connect", ReplaceFrom: 0, ReplaceTo: 6},
			},
		}
	}

	current := prompt.applyCompletion("connec")

	if current != "connect" {
		t.Fatalf("current = %q", current)
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestPromptApplyCompletionUsesOriginalBaseInputAcrossCycle(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.completer = func(input string) Completion {
		if input == "connect" {
			return Completion{
				Prefix: "connect",
				Suggestions: []Suggestion{
					{Value: "connect", Replacement: "connect", ReplaceFrom: 0, ReplaceTo: 7},
					{Value: "connections", Replacement: "connections", ReplaceFrom: 0, ReplaceTo: 7},
				},
			}
		}
		return Completion{
			Prefix: "conn",
			Suggestions: []Suggestion{
				{Value: "connect", Replacement: "connect", ReplaceFrom: 0, ReplaceTo: 4},
				{Value: "connections", Replacement: "connections", ReplaceFrom: 0, ReplaceTo: 4},
				{Value: "connection create", Replacement: "connection create", ReplaceFrom: 0, ReplaceTo: 4},
			},
		}
	}

	if got := prompt.applyCompletion("conn"); got != "connect" {
		t.Fatalf("first completion = %q", got)
	}
	if got := prompt.applyCompletion("connect"); got != "connect" {
		t.Fatalf("second tab should list candidates before cycling, got %q", got)
	}
	if got := prompt.applyCompletion("connect"); got != "connections" {
		t.Fatalf("expected cycle to continue from original base input, got %q", got)
	}
}

func TestPromptCompletionSessionReset(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.session = newCompletionSession("conn", []Suggestion{{Value: "connect", Replacement: "connect", ReplaceFrom: 0, ReplaceTo: 4}})

	prompt.resetCompletionSession()

	if prompt.session != nil {
		t.Fatalf("expected nil session after reset")
	}
}

func TestPromptApplyCompletionNoCandidatesLeavesInputUnchanged(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.completer = func(string) Completion { return Completion{} }

	if got := prompt.applyCompletion("zzz"); got != "zzz" {
		t.Fatalf("completion = %q", got)
	}
}

func TestCompletionSessionDeduplicatesSuggestionsDeterministically(t *testing.T) {
	t.Parallel()

	session := newCompletionSession("conn", []Suggestion{
		{Value: "connect"},
		{Value: "connections"},
		{Value: "connect"},
	})
	if session == nil {
		t.Fatalf("expected session")
	}
	if len(session.Suggestions) != 2 {
		t.Fatalf("suggestions = %#v", session.Suggestions)
	}
	if session.Suggestions[0].Value != "connect" || session.Suggestions[1].Value != "connections" {
		t.Fatalf("unexpected order: %#v", session.Suggestions)
	}
}

func TestPromptApplyCompletionResetsSessionWhenInputDiverges(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.completer = func(string) Completion {
		return Completion{
			Prefix: "co",
			Suggestions: []Suggestion{
				{Value: "connect", Replacement: "connect", ReplaceFrom: 0, ReplaceTo: 2},
				{Value: "connections", Replacement: "connections", ReplaceFrom: 0, ReplaceTo: 2},
			},
		}
	}
	prompt.session = newCompletionSession("conn", []Suggestion{
		{Value: "connect", Replacement: "connect", ReplaceFrom: 0, ReplaceTo: 4},
		{Value: "connections", Replacement: "connections", ReplaceFrom: 0, ReplaceTo: 4},
	})

	if got := prompt.applyCompletion("co"); got != "connect" {
		t.Fatalf("completion = %q", got)
	}
}

func TestPromptApplyCompletionPreservesCommandPrefixForArguments(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.completer = func(string) Completion {
		return Completion{
			Prefix: "gre",
			Suggestions: []Suggestion{
				{Value: "greenhn-dev", Replacement: "greenhn-dev", ReplaceFrom: 4, ReplaceTo: 7},
			},
		}
	}

	if got := prompt.applyCompletion("use gre"); got != "use greenhn-dev" {
		t.Fatalf("completion = %q", got)
	}
}

func TestPromptApplyCompletionCyclesArgumentCandidatesWithPrefixPreserved(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.completer = func(string) Completion {
		return Completion{
			Suggestions: []Suggestion{
				{Value: "db1", Replacement: "db1", ReplaceFrom: 4, ReplaceTo: 4},
				{Value: "db2", Replacement: "db2", ReplaceFrom: 4, ReplaceTo: 4},
				{Value: "db3", Replacement: "db3", ReplaceFrom: 4, ReplaceTo: 4},
			},
		}
	}

	if got := prompt.applyCompletion("use "); got != "use db" {
		t.Fatalf("first completion should expand to common prefix, got %q", got)
	}
	if got := prompt.applyCompletion("use db"); got != "use db" {
		t.Fatalf("second completion should keep current input while listing candidates, got %q", got)
	}
	if got := prompt.applyCompletion("use db"); got != "use db1" {
		t.Fatalf("third completion = %q", got)
	}
	if got := prompt.applyCompletion("use db1"); got != "use db2" {
		t.Fatalf("fourth completion = %q", got)
	}
	if got := prompt.applyCompletion("use db2"); got != "use db3" {
		t.Fatalf("fifth completion = %q", got)
	}
}

func TestPromptApplyCompletionUsesCommonPrefixForCommands(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.completer = func(string) Completion {
		return Completion{
			Prefix: "con",
			Suggestions: []Suggestion{
				{Value: "connect", Replacement: "connect", ReplaceFrom: 0, ReplaceTo: 3},
				{Value: "connections", Replacement: "connections", ReplaceFrom: 0, ReplaceTo: 3},
				{Value: "connection create", Replacement: "connection create", ReplaceFrom: 0, ReplaceTo: 3},
			},
		}
	}

	if got := prompt.applyCompletion("con"); got != "connect" {
		t.Fatalf("completion = %q", got)
	}
	if prompt.session == nil || prompt.session.CommonInput != "connect" {
		t.Fatalf("unexpected session: %#v", prompt.session)
	}
}

func TestPromptApplyCompletionUsesCommonPrefixForArguments(t *testing.T) {
	t.Parallel()

	prompt := NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	prompt.completer = func(string) Completion {
		return Completion{
			Prefix: "gr",
			Suggestions: []Suggestion{
				{Value: "greenhn-dev", Replacement: "greenhn-dev", ReplaceFrom: 4, ReplaceTo: 6},
				{Value: "greenhn-prod", Replacement: "greenhn-prod", ReplaceFrom: 4, ReplaceTo: 6},
			},
		}
	}

	if got := prompt.applyCompletion("use gr"); got != "use greenhn-" {
		t.Fatalf("completion = %q", got)
	}
}

func TestPromptApplyCompletionDoubleTabPrintsCandidateListAndRedraws(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)
	prompt.isTerm = func() bool { return true }
	prompt.rawActive = true
	prompt.label = "dbx(prod)> "
	prompt.current = "use db1"
	prompt.completer = func(string) Completion {
		return Completion{
			Suggestions: []Suggestion{
				{Value: "db1", Replacement: "db1", ReplaceFrom: 4, ReplaceTo: 7},
				{Value: "db2", Replacement: "db2", ReplaceFrom: 4, ReplaceTo: 7},
			},
		}
	}
	prompt.session = newCompletionSession("use ", []Suggestion{
		{Value: "db1", Replacement: "db1", ReplaceFrom: 4, ReplaceTo: 4},
		{Value: "db2", Replacement: "db2", ReplaceFrom: 4, ReplaceTo: 4},
	})

	if got := prompt.applyCompletion("use db1"); got != "use db1" {
		t.Fatalf("completion = %q", got)
	}

	want := "\r\033[2K\r\ndb1\r\ndb2\r\n\r\n\r\033[2Kdbx(prod)> use db1"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRawModeWriterNormalizesLineEndings(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	writer := rawModeWriter{writer: &out}

	if _, err := writer.Write([]byte("alpha\nbeta\r\ngamma")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if got, want := out.String(), "alpha\r\nbeta\r\ngamma"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
