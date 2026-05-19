package app

import (
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

	got := normalizeHelpTopic("  connection   create  ")
	if got != "connection create" {
		t.Fatalf("normalizeHelpTopic = %q", got)
	}
}

func TestPrintHelpTopic(t *testing.T) {
	t.Parallel()

	var prompt testPrinter
	if err := printHelpTopic(&prompt, "connection create"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}

	joined := strings.Join(prompt.lines, "\n")
	if !strings.Contains(joined, "Create a new saved connection.") {
		t.Fatalf("help output missing expected text: %q", joined)
	}
	if !strings.Contains(joined, "~/.config/dbx/{name}/config.json") {
		t.Fatalf("help output missing config path: %q", joined)
	}
}
