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

func TestPromptApplyCompletionPrintsSuggestionsOnFreshLines(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)
	prompt.isTerm = func() bool { return true }
	prompt.rawActive = true
	prompt.label = "dbx(prod)> "
	prompt.current = "connect"

	current := prompt.applyCompletion("connect", Completion{
		Prefix: "connect",
		Suggestions: []Suggestion{
			{Value: "connect", Description: "connect to a saved connection"},
			{Value: "connections", Description: "list saved connections"},
		},
	})

	if current != "connect" {
		t.Fatalf("current = %q", current)
	}
	want := "\r\033[2K\r\nconnect      connect to a saved connection\r\nconnections  list saved connections\r\n\r\n\r\033[2Kdbx(prod)> connect"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestPromptApplyCompletionSingleCandidateDoesNotPrintList(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)

	current := prompt.applyCompletion("connec", Completion{
		Prefix: "connec",
		Suggestions: []Suggestion{
			{Value: "connect", Description: "connect to a saved connection"},
		},
	})

	if current != "connect " {
		t.Fatalf("current = %q", current)
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected output: %q", out.String())
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
