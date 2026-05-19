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
