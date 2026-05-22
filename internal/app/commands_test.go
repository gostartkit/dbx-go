package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
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

func TestHandleConnectionsIncludesInvalidConfigurations(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	invalidPath := store.ConnectionConfigPath("broken")
	if err := os.MkdirAll(filepath.Dir(invalidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidPath, []byte(`{"name":"broken","driver":"mysql","mode":"direct","host":"127.0.0.1","port":70000,"user":"root"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{})
	if err := app.handleConnections(context.Background()); err != nil {
		t.Fatalf("handleConnections returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Configured connections:") {
		t.Fatalf("output missing header: %q", output)
	}
	if !strings.Contains(output, "broken [invalid]") {
		t.Fatalf("output missing invalid connection: %q", output)
	}
	if !strings.Contains(output, "port must be greater than zero") {
		t.Fatalf("output missing invalid reason: %q", output)
	}
	if !strings.Contains(output, "prod (mysql direct 127.0.0.1:3306)") {
		t.Fatalf("output missing valid connection: %q", output)
	}
}

func TestHandleConnectionShowIncludesInvalidConfigurationStatus(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	invalidPath := store.ConnectionConfigPath("broken")
	if err := os.MkdirAll(filepath.Dir(invalidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidPath, []byte(`{"name":"broken","driver":"mysql","mode":"direct","host":"127.0.0.1","port":70000,"user":"root"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{})
	if err := app.handleConnectionShow(context.Background(), "broken"); err != nil {
		t.Fatalf("handleConnectionShow returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Name: broken") {
		t.Fatalf("output missing name: %q", output)
	}
	if !strings.Contains(output, "Status: invalid") {
		t.Fatalf("output missing invalid status: %q", output)
	}
	if !strings.Contains(output, "Issue: port must be greater than zero") {
		t.Fatalf("output missing issue: %q", output)
	}
}

func TestHandleConnectionShowIncludesParseErrors(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	invalidPath := store.ConnectionConfigPath("broken")
	if err := os.MkdirAll(filepath.Dir(invalidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidPath, []byte(`{"name":"broken",`), 0o644); err != nil {
		t.Fatal(err)
	}

	app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{})
	if err := app.handleConnectionShow(context.Background(), "broken"); err != nil {
		t.Fatalf("handleConnectionShow returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Name: broken") {
		t.Fatalf("output missing name: %q", output)
	}
	if !strings.Contains(output, "Status: invalid") {
		t.Fatalf("output missing invalid status: %q", output)
	}
	if !strings.Contains(output, "Issue: unexpected end of JSON input") {
		t.Fatalf("output missing parse issue: %q", output)
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
	if err := printHelpTopic(&prompt, "exec"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	joined = strings.Join(prompt.lines, "\n")
	if !strings.Contains(joined, "Execute a named operation") {
		t.Fatalf("help output missing exec command text: %q", joined)
	}
	if !strings.Contains(joined, "--validate") {
		t.Fatalf("help output missing validate flag text: %q", joined)
	}
	if strings.Contains(joined, "template <name>") {
		t.Fatalf("help output still contains removed template subcommand: %q", joined)
	}

	prompt.lines = nil
	if err := printHelpTopic(&prompt, "show"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	joined = strings.Join(prompt.lines, "\n")
	if !strings.Contains(joined, "users") {
		t.Fatalf("help output missing users subcommand: %q", joined)
	}
	if strings.Contains(joined, "user <name>") {
		t.Fatalf("help output still contains removed user subcommand: %q", joined)
	}

	prompt.lines = nil
	if err := printHelpTopic(&prompt, "template"); err != nil {
		t.Fatalf("printHelpTopic returned error: %v", err)
	}
	joined = strings.Join(prompt.lines, "\n")
	if !strings.Contains(joined, "Subcommands:") || !strings.Contains(joined, "render") {
		t.Fatalf("schema help output missing template subcommands: %q", joined)
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
	if !strings.Contains(out.String(), "connect\n  connect prod") || !strings.Contains(out.String(), "show tables\n  show table users") || !strings.Contains(out.String(), "exec\n  exec create_database_with_user --validate") {
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
		{line: "help exec", want: "Usage: dbx exec <operation> [flags]"},
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

func TestREPLUnknownShowTargetUsesShowContext(t *testing.T) {
	t.Parallel()

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	_, runErr := app.handleLine(context.Background(), "show indexes")
	if runErr == nil {
		t.Fatalf("expected error")
	}
	message := runErr.Error()
	if !strings.Contains(message, `unknown show target "indexes"`) {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if strings.Contains(message, `Did you mean show?`) {
		t.Fatalf("unexpected root-level suggestion: %v", runErr)
	}
	if !strings.Contains(message, "available show targets:") {
		t.Fatalf("missing show targets list: %v", runErr)
	}
	for _, name := range []string{"databases", "tables", "table", "columns", "rows", "connections", "connection", "users", "templates", "context"} {
		if !strings.Contains(message, "\n  "+name) {
			t.Fatalf("missing show target %q in error: %v", name, runErr)
		}
	}
	if strings.Contains(message, "\n  user\n") {
		t.Fatalf("unexpected removed show target in error: %v", runErr)
	}
	if strings.Contains(message, "\n  connect\n") || strings.Contains(message, "\n  create\n") {
		t.Fatalf("unexpected root command in error: %v", runErr)
	}
}

func TestREPLUnknownShowTargetSuggestsClosestMatch(t *testing.T) {
	t.Parallel()

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	_, runErr := app.handleLine(context.Background(), "show tabl")
	if runErr == nil {
		t.Fatalf("expected error")
	}
	message := runErr.Error()
	if !strings.Contains(message, `unknown show target "tabl"`) {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(message, `did you mean "table"?`) {
		t.Fatalf("missing contextual suggestion: %v", runErr)
	}
}

func TestREPLOldRunCommandIsRemoved(t *testing.T) {
	t.Parallel()

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	_, runErr := app.handleLine(context.Background(), "run deploy")
	if runErr == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(runErr.Error(), `unknown command "run"`) {
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
