package ui

import (
	"bytes"
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

func TestPromptRedrawLineWithHintRestoresCursor(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	prompt := NewPrompt(strings.NewReader(""), &out)

	prompt.redrawLine("dbx> ", "crea", "te database")

	want := "\r\033[2Kdbx> crea\033[90mte database\033[0m\033[11D"
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
