package app

import (
	"testing"

	"pkg.gostartkit.com/cmd"
)

func TestREPLCommandSpecsIncludeSharedAndOverlayCommands(t *testing.T) {
	t.Parallel()

	have := map[string]struct{}{}
	for _, spec := range replCommandSpecs() {
		have[normalizeHelpTopic(spec.Path)] = struct{}{}
	}

	for _, want := range []string{
		"connect",
		"create connection",
		"edit connection",
		"drop connection",
		"show connection",
		"show connections",
		"show databases",
		"show template",
		"run template",
		"use database",
		"test connection",
		"doctor connection",
		"clear",
		"exit",
	} {
		if _, ok := have[normalizeHelpTopic(want)]; !ok {
			t.Fatalf("missing command spec %q", want)
		}
	}
}

func TestCLIAndREPLShareBusinessCommandTree(t *testing.T) {
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

	for path := range cliCommands {
		if _, ok := replCommands[path]; !ok {
			t.Fatalf("repl tree missing cli command path %q", path)
		}
	}
}

func TestHelpCompletionContainsHelpTopics(t *testing.T) {
	t.Parallel()

	have := map[string]struct{}{}
	for _, suggestion := range helpCompletionTopics() {
		have[suggestion.Value] = struct{}{}
	}

	for _, want := range []string{"test connection", "show templates", "run template", "show connection"} {
		if _, ok := have[want]; !ok {
			t.Fatalf("missing help topic %q", want)
		}
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
