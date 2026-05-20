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
		"/",
		"connect",
		"conn",
		"connection create",
		"show databases",
		"template run",
		"use",
		"dry-run on",
		"ls db",
		"test conn",
		"doctor conn",
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

	for _, want := range []string{"aliases", "connection test", "show templates", "template run"} {
		if _, ok := have[want]; !ok {
			t.Fatalf("missing help topic %q", want)
		}
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
