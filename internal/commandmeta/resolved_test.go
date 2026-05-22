package commandmeta

import "testing"

func TestLookupCommandPathMatchesVisiblePathAndAlias(t *testing.T) {
	t.Parallel()

	command, ok := LookupCommandPath(DefaultManifest(), "show templates")
	if !ok {
		t.Fatalf("expected visible command lookup to succeed")
	}
	if command.Command == nil || command.Command.Name != "templates" {
		t.Fatalf("unexpected visible command: %#v", command.Command)
	}
	if command.Alias {
		t.Fatalf("expected visible path lookup to be canonical")
	}

	alias, ok := LookupCommandPath(DefaultManifest(), "q")
	if !ok {
		t.Fatalf("expected alias lookup to succeed")
	}
	if alias.Command == nil || alias.Command.Name != "exit" {
		t.Fatalf("unexpected alias command: %#v", alias.Command)
	}
	if !alias.Alias {
		t.Fatalf("expected alias lookup to be marked as alias")
	}
}

func TestFlattenCommandsPropagatesHiddenToSubcommands(t *testing.T) {
	t.Parallel()

	command, ok := LookupCommandPath(DefaultManifest(), "connection create")
	if !ok {
		t.Fatalf("expected hidden command lookup to succeed")
	}
	if !command.Hidden {
		t.Fatalf("expected hidden parent visibility to propagate to subcommand")
	}
}
