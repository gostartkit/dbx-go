package app

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	tpl "pkg.gostartkit.com/dbx/internal/template"
)

func TestRedactTemplateValues(t *testing.T) {
	t.Parallel()

	template := &tpl.Template{
		Inputs: []tpl.Input{
			{Name: "password", Secret: true},
			{Name: "api_key", Type: "secret"},
			{Name: "database", Secret: false},
		},
	}

	values := map[string]string{
		"database": "appdb",
		"password": "super-secret",
		"api_key":  "typed-secret",
	}

	redacted := redactTemplateValues(template, values)

	if redacted["password"] != "***" {
		t.Fatalf("password redaction = %q, want ***", redacted["password"])
	}
	if redacted["api_key"] != "***" {
		t.Fatalf("api_key redaction = %q, want ***", redacted["api_key"])
	}
	if redacted["database"] != "appdb" {
		t.Fatalf("database value = %q, want appdb", redacted["database"])
	}
	if values["password"] != "super-secret" {
		t.Fatalf("original values were mutated")
	}
	if values["api_key"] != "typed-secret" {
		t.Fatalf("original typed secret was mutated")
	}
}

type testPrinter struct {
	lines []string
}

func (p *testPrinter) Println(args ...any) {
	if len(args) == 0 {
		p.lines = append(p.lines, "")
		return
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, fmt.Sprint(arg))
	}
	p.lines = append(p.lines, strings.TrimSpace(strings.Join(parts, " ")))
}

func TestNormalizeHelpTopic(t *testing.T) {
	t.Parallel()

	got := normalizeHelpTopic("  create   connection  ")
	if got != "create connection" {
		t.Fatalf("normalizeHelpTopic = %q", got)
	}
}

func TestPrintHelpTopic(t *testing.T) {
	t.Parallel()

	var prompt testPrinter
	if err := printHelpTopic(&prompt, "create connection"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}

	joined := strings.Join(prompt.lines, "\n")
	if !strings.Contains(joined, "Create a saved connection.") {
		t.Fatalf("help output missing expected text: %q", joined)
	}
	if !strings.Contains(joined, "~/.config/dbx/{name}/config.json") {
		t.Fatalf("help output missing config path: %q", joined)
	}
}

func TestPrintHelpTemplateTopics(t *testing.T) {
	t.Parallel()

	var prompt testPrinter
	if err := printHelpTopic(&prompt, "show templates"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	joined := strings.Join(prompt.lines, "\n")
	if !strings.Contains(joined, "Show resolved workflow templates") {
		t.Fatalf("help output missing show templates text: %q", joined)
	}

	prompt.lines = nil
	if err := printHelpTopic(&prompt, "show template"); err == nil {
		t.Fatalf("expected removed help topic to fail")
	}

	prompt.lines = nil
	if err := printHelpTopic(&prompt, "run template"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	joined = strings.Join(prompt.lines, "\n")
	if !strings.Contains(joined, "Run or validate a workflow template") {
		t.Fatalf("help output missing template run text: %q", joined)
	}
	if !strings.Contains(joined, "--validate") {
		t.Fatalf("help output missing validate flag text: %q", joined)
	}

	prompt.lines = nil
	if err := printHelpTopic(&prompt, "show"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	joined = strings.Join(prompt.lines, "\n")
	if !strings.Contains(joined, "users") || !strings.Contains(joined, "user <name>") {
		t.Fatalf("help output missing user subcommands: %q", joined)
	}
}

func TestREPLHelpCommandHasOutput(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	if err := app.replCommandApp().RunLine(context.Background(), "help"); err != nil {
		t.Fatalf("RunLine returned error: %v", err)
	}
	if !strings.Contains(out.String(), "Usage:") || !strings.Contains(out.String(), "Available Commands:") {
		t.Fatalf("help output missing expected sections: %q", out.String())
	}
	if !strings.Contains(out.String(), "connect\n  connect prod") || !strings.Contains(out.String(), "run\n  run template seed-users --validate") {
		t.Fatalf("help output missing grouped examples: %q", out.String())
	}
}

func TestREPLHelpTopicsHaveOutput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		line string
		want string
	}{
		{line: "help show", want: "Usage: dbx show <subcommand>"},
		{line: "help connect", want: "Usage: dbx connect <name>"},
		{line: "help run", want: "Usage: dbx run <subcommand>"},
	}

	for _, tc := range cases {
		var out bytes.Buffer
		app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: t.TempDir()})
		if err != nil {
			t.Fatalf("NewWithOptions returned error: %v", err)
		}
		if err := app.replCommandApp().RunLine(context.Background(), tc.line); err != nil {
			t.Fatalf("RunLine(%q) returned error: %v", tc.line, err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("help output for %q missing %q: %q", tc.line, tc.want, out.String())
		}
	}
}

func TestREPLUnknownCommandSuggestsClosestMatch(t *testing.T) {
	t.Parallel()

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	_, runErr := app.handleLine(context.Background(), "cnnect prod")
	if runErr == nil || !strings.Contains(runErr.Error(), `Did you mean connect?`) {
		t.Fatalf("unexpected error: %v", runErr)
	}
}

func TestREPLMissingArgumentShowsUsage(t *testing.T) {
	t.Parallel()

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	_, runErr := app.handleLine(context.Background(), "show rows")
	if runErr == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(runErr.Error(), "missing required argument: table") || !strings.Contains(runErr.Error(), "Usage: show rows <table> [--limit n]") {
		t.Fatalf("unexpected error: %v", runErr)
	}
}

func TestREPLMissingConnectionSuggestsNextStep(t *testing.T) {
	t.Parallel()

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	_, runErr := app.handleLine(context.Background(), "show connection prod")
	if runErr == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(runErr.Error(), `Connection "prod" not found.`) || !strings.Contains(runErr.Error(), "Next: show connections") {
		t.Fatalf("unexpected error: %v", runErr)
	}
}
