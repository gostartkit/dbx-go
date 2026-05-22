package app

import (
	"context"
	"slices"
	"testing"

	"pkg.gostartkit.com/cmd"
)

func TestRootCommandSetMatchesShellSurface(t *testing.T) {
	t.Parallel()

	spec := (&cliBuilder{
		mode:    ModeREPL,
		globals: &cliGlobals{Format: "text"},
	}).buildApp().Spec()

	have := map[string]struct{}{}
	for _, command := range spec.Commands {
		have[normalizeHelpTopic(command.Name)] = struct{}{}
	}

	want := []string{"connect", "use", "show", "create", "drop", "exec", "doctor", "audit", "exit"}
	if len(have) != len(want) {
		t.Fatalf("root command count = %d, want %d (%v)", len(have), len(want), have)
	}
	for _, name := range want {
		if _, ok := have[name]; !ok {
			t.Fatalf("missing root command %q", name)
		}
	}
}

func TestRemovedRootCommandsAreAbsent(t *testing.T) {
	t.Parallel()

	spec := (&cliBuilder{
		mode:    ModeREPL,
		globals: &cliGlobals{Format: "text"},
	}).buildApp().Spec()

	have := map[string]struct{}{}
	for _, command := range spec.Commands {
		have[normalizeHelpTopic(command.Name)] = struct{}{}
	}

	for _, removed := range []string{"count", "peek", "sample", "truncate", "rename", "validate", "edit", "test", "context", "clear", "user", "users", "run"} {
		if _, ok := have[removed]; ok {
			t.Fatalf("unexpected removed root command %q", removed)
		}
	}
}

func TestSharedCommandPathsIncludeFinalCommands(t *testing.T) {
	t.Parallel()

	have := map[string]struct{}{}
	for _, spec := range replCommandSpecs() {
		have[normalizeHelpTopic(spec.Path)] = struct{}{}
	}

	for _, want := range []string{
		"connect",
		"use",
		"show databases",
		"show tables",
		"show table",
		"show columns",
		"show rows",
		"show connections",
		"show connection",
		"show users",
		"show templates",
		"show context",
		"create connection",
		"create database",
		"create user",
		"drop connection",
		"drop database",
		"drop user",
		"exec",
		"doctor",
		"audit log",
		"exit",
	} {
		if _, ok := have[want]; !ok {
			t.Fatalf("missing command path %q", want)
		}
	}
}

func TestRemovedCommandPathsAreAbsent(t *testing.T) {
	t.Parallel()

	have := map[string]struct{}{}
	for _, spec := range replCommandSpecs() {
		have[normalizeHelpTopic(spec.Path)] = struct{}{}
	}

	for _, removed := range []string{
		"count rows",
		"peek rows",
		"sample rows",
		"truncate table",
		"rename table",
		"validate template",
		"edit connection",
		"test connection",
		"context",
		"describe",
		"show template",
		"doctor connection",
		"show indexes",
		"show foreign keys",
		"show processlist",
		"show triggers",
		"show variables",
		"show grants",
		"show views",
		"show create table",
		"show table status",
	} {
		if _, ok := have[removed]; ok {
			t.Fatalf("unexpected removed command path %q", removed)
		}
	}
}

func TestRemovedCommandsReturnErrors(t *testing.T) {
	t.Parallel()

	app := (&cliBuilder{
		mode:    ModeREPL,
		globals: &cliGlobals{Format: "text"},
	}).buildApp()

	for _, line := range []string{
		"count rows users",
		"peek rows users",
		"sample rows users",
		"truncate table users",
		"rename table users_tmp users",
		"validate template readonly_user",
		"edit connection prod",
		"test connection prod",
		"context",
		"describe users",
		"show template readonly_user",
		"doctor connection prod",
		`run deploy`,
		`exec sql "SELECT 1"`,
		"exec template readonly_user",
	} {
		if err := app.RunLine(context.Background(), line); err == nil {
			t.Fatalf("expected removed command to fail: %q", line)
		}
	}
}

func TestCLIAndREPLShareCommandTree(t *testing.T) {
	t.Parallel()

	cli := newCLIBuilder(nil, nil, nil, Options{}).buildApp().Spec()
	repl := (&cliBuilder{
		mode:    ModeREPL,
		globals: &cliGlobals{Format: "text"},
	}).buildApp().Spec()

	cliCommands := map[string]struct{}{}
	for _, command := range cli.Commands {
		collectSpecPaths(cliCommands, "", command)
	}

	replCommands := map[string]struct{}{}
	for _, command := range repl.Commands {
		collectSpecPaths(replCommands, "", command)
	}

	if len(cliCommands) != len(replCommands) {
		t.Fatalf("cli/repl path counts differ: %d vs %d", len(cliCommands), len(replCommands))
	}
	for path := range cliCommands {
		if _, ok := replCommands[path]; !ok {
			t.Fatalf("repl tree missing cli command path %q", path)
		}
	}
}

func TestREPLCommandSpecsMatchCommandTree(t *testing.T) {
	t.Parallel()

	commandTree := (&cliBuilder{
		mode:    ModeREPL,
		globals: &cliGlobals{Format: "text"},
	}).buildApp().Spec()

	have := map[string]struct{}{}
	for _, command := range commandTree.Commands {
		collectSpecPaths(have, "", command)
	}

	want := map[string]struct{}{}
	for _, spec := range replCommandSpecs() {
		if spec.Hidden {
			continue
		}
		want[normalizeHelpTopic(spec.Path)] = struct{}{}
	}

	if len(have) != len(want) {
		t.Fatalf("visible repl/spec path counts differ: %d vs %d", len(have), len(want))
	}
	for path := range want {
		if _, ok := have[path]; !ok {
			t.Fatalf("repl tree missing command spec path %q", path)
		}
	}
}

func TestHelpCompletionContainsFinalTopics(t *testing.T) {
	t.Parallel()

	have := map[string]struct{}{}
	for _, suggestion := range helpCompletionTopics() {
		have[suggestion.Value] = struct{}{}
	}

	for _, want := range []string{"doctor", "show templates", "exec", "show rows", "show users"} {
		if _, ok := have[want]; !ok {
			t.Fatalf("missing help topic %q", want)
		}
	}
	if _, ok := have["run"]; ok {
		t.Fatalf("unexpected removed help topic %q", "run")
	}
	if _, ok := have["run template"]; ok {
		t.Fatalf("unexpected removed help topic %q", "run template")
	}
	if _, ok := have["run sql"]; ok {
		t.Fatalf("unexpected removed help topic %q", "run sql")
	}
}

func TestSharedCommandPathsAndAliasesAreUnique(t *testing.T) {
	t.Parallel()

	spec := (&cliBuilder{
		mode:    ModeREPL,
		globals: &cliGlobals{Format: "text"},
	}).buildApp().Spec()

	seen := map[string]string{}
	for _, command := range spec.Commands {
		assertUniquePaths(t, seen, "", command)
	}
}

func TestREPLSpecUsesInteractivePositionals(t *testing.T) {
	t.Parallel()

	spec := (&cliBuilder{
		mode:    ModeREPL,
		globals: &cliGlobals{Format: "text"},
	}).buildApp().SpecFor(cmd.SurfaceREPL)

	createConnection := findCommandSpec(t, spec.Commands, []string{"create", "connection"})
	if len(createConnection.Positionals) != 0 {
		t.Fatalf("expected no REPL positionals for create connection, got %+v", createConnection.Positionals)
	}

	createDatabase := findCommandSpec(t, spec.Commands, []string{"create", "database"})
	if len(createDatabase.Positionals) != 0 {
		t.Fatalf("expected no REPL positionals for create database, got %+v", createDatabase.Positionals)
	}

	createUser := findCommandSpec(t, spec.Commands, []string{"create", "user"})
	if len(createUser.Positionals) != 1 || createUser.Positionals[0].Required {
		t.Fatalf("expected optional REPL positional for create user, got %+v", createUser.Positionals)
	}
}

func TestCLISpecUsesRequiredOneShotPositionals(t *testing.T) {
	t.Parallel()

	spec := newCLIBuilder(nil, nil, nil, Options{}).buildApp().SpecFor(cmd.SurfaceCLI)

	createConnection := findCommandSpec(t, spec.Commands, []string{"create", "connection"})
	if len(createConnection.Positionals) != 1 || !createConnection.Positionals[0].Required {
		t.Fatalf("expected required CLI positional for create connection, got %+v", createConnection.Positionals)
	}

	execCommand := findCommandSpec(t, spec.Commands, []string{"exec"})
	if len(execCommand.Positionals) != 1 || !execCommand.Positionals[0].Required {
		t.Fatalf("expected required CLI positional for exec, got %+v", execCommand.Positionals)
	}
}

func collectSpecPaths(dst map[string]struct{}, prefix string, spec cmd.CommandSpec) {
	path := normalizeHelpTopic(prefix + " " + spec.Name)
	dst[path] = struct{}{}
	for _, alias := range spec.Aliases {
		dst[normalizeHelpTopic(prefix+" "+alias)] = struct{}{}
	}
	for _, subCommand := range spec.SubCommands {
		collectSpecPaths(dst, path, subCommand)
	}
}

func assertUniquePaths(t *testing.T, seen map[string]string, prefix string, spec cmd.CommandSpec) {
	t.Helper()

	path := normalizeHelpTopic(prefix + " " + spec.Name)
	if other, ok := seen[path]; ok {
		t.Fatalf("duplicate command path %q for %q and %q", path, other, spec.Name)
	}
	seen[path] = spec.Name

	for _, alias := range spec.Aliases {
		aliasPath := normalizeHelpTopic(prefix + " " + alias)
		if other, ok := seen[aliasPath]; ok {
			t.Fatalf("duplicate alias path %q for %q and %q", aliasPath, other, spec.Name)
		}
		seen[aliasPath] = spec.Name
	}

	for _, subCommand := range spec.SubCommands {
		assertUniquePaths(t, seen, path, subCommand)
	}
}

func findCommandSpec(t *testing.T, commands []cmd.CommandSpec, want []string) cmd.CommandSpec {
	t.Helper()

	for _, command := range commands {
		if slices.Equal(command.Path, want) {
			return command
		}
		if found := findCommandSpecRecursive(command.SubCommands, want); found != nil {
			return *found
		}
	}
	t.Fatalf("missing command spec for path %v", want)
	return cmd.CommandSpec{}
}

func findCommandSpecRecursive(commands []cmd.CommandSpec, want []string) *cmd.CommandSpec {
	for _, command := range commands {
		if slices.Equal(command.Path, want) {
			copyCommand := command
			return &copyCommand
		}
		if found := findCommandSpecRecursive(command.SubCommands, want); found != nil {
			return found
		}
	}
	return nil
}
