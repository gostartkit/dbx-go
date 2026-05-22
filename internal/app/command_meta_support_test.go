package app

import (
	"testing"

	"pkg.gostartkit.com/cmd"
)

func TestManifestPositionalsBuildsKnownArguments(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	positionals := builder.manifestPositionals("show rows", nil)
	if len(positionals) != 1 {
		t.Fatalf("positionals count = %d, want 1", len(positionals))
	}
	if positionals[0].Name != "table" {
		t.Fatalf("positional name = %q, want %q", positionals[0].Name, "table")
	}
	if !positionals[0].Required {
		t.Fatalf("expected positional to be required")
	}
	if positionals[0].Completion == nil {
		t.Fatalf("expected positional completion")
	}
}

func TestManifestPositionalsFallsBackForUnknownPath(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	fallback := []cmd.PositionalArg{{Name: "name", Usage: "fallback"}}
	positionals := builder.manifestPositionals("missing path", fallback)
	if len(positionals) != 1 {
		t.Fatalf("positionals count = %d, want 1", len(positionals))
	}
	if positionals[0].Name != "name" {
		t.Fatalf("fallback positional name = %q, want %q", positionals[0].Name, "name")
	}
	if positionals[0].Usage != "fallback" {
		t.Fatalf("fallback positional usage = %q, want %q", positionals[0].Usage, "fallback")
	}
}

func TestNewManifestCommandUsesManifestDefaults(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	command := builder.newManifestCommand(manifestCommandOptions{
		Path:          "exit",
		UsageFallback: "dbx exit",
		ShortFallback: "fallback short",
	})
	if command.Name != "exit" {
		t.Fatalf("command name = %q, want %q", command.Name, "exit")
	}
	if command.UsageLine != "dbx exit" {
		t.Fatalf("command usage = %q, want %q", command.UsageLine, "dbx exit")
	}
	if command.Short != "Exit the REPL." {
		t.Fatalf("command short = %q, want %q", command.Short, "Exit the REPL.")
	}
	if len(command.Aliases) != 1 || command.Aliases[0] != "quit" {
		t.Fatalf("command aliases = %#v", command.Aliases)
	}
}

func TestNewManifestCommandSupportsNameOverrideAndFallbackUsage(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	command := builder.newManifestCommand(manifestCommandOptions{
		Path:                "show connections",
		Name:                "connections",
		UsageFallback:       "dbx connections",
		PreferFallbackUsage: true,
		ShortFallback:       "fallback short",
	})
	if command.Name != "connections" {
		t.Fatalf("command name = %q, want %q", command.Name, "connections")
	}
	if command.UsageLine != "dbx connections" {
		t.Fatalf("command usage = %q, want %q", command.UsageLine, "dbx connections")
	}
	if command.Short != "Show saved connections." {
		t.Fatalf("command short = %q, want %q", command.Short, "Show saved connections.")
	}
}

func TestManifestFlagDefaultIntReadsManifestValue(t *testing.T) {
	t.Parallel()

	got := manifestFlagDefaultInt("show rows", "limit", 99)
	if got != 10 {
		t.Fatalf("manifest flag default = %d, want %d", got, 10)
	}
}

func TestManifestFlagDefaultStringReadsManifestValue(t *testing.T) {
	t.Parallel()

	got := manifestFlagDefaultString("create database", "charset", "latin1")
	if got != "utf8mb4" {
		t.Fatalf("manifest flag default = %q, want %q", got, "utf8mb4")
	}
}

func TestBindManifestStringFlagUsesManifestMetadata(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	flags := cmd.NewFlagSet("test", cmd.ContinueOnError)
	var value string
	builder.bindManifestStringFlag(flags, "show templates", "tag", &value, "", "fallback usage")

	flag, ok := flags.Lookup("tag")
	if !ok {
		t.Fatalf("expected tag flag to exist")
	}
	if flag.Usage != "Filter templates by tag." {
		t.Fatalf("flag usage = %q, want %q", flag.Usage, "Filter templates by tag.")
	}
	if flag.Completion == nil {
		t.Fatalf("expected manifest completion to be attached")
	}
}

func TestBindManifestIntFlagUsesManifestMetadata(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	flags := cmd.NewFlagSet("test", cmd.ContinueOnError)
	value := 0
	builder.bindManifestIntFlag(flags, "show rows", "limit", &value, 99, "fallback usage")

	flag, ok := flags.Lookup("limit")
	if !ok {
		t.Fatalf("expected limit flag to exist")
	}
	if flag.Usage != "Limit the number of rows returned." {
		t.Fatalf("flag usage = %q, want %q", flag.Usage, "Limit the number of rows returned.")
	}
	if flag.DefValue != "99" {
		t.Fatalf("flag default = %q, want %q", flag.DefValue, "99")
	}
}

func TestBindManifestBoolFlagUsesManifestMetadata(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	flags := cmd.NewFlagSet("test", cmd.ContinueOnError)
	value := false
	builder.bindManifestBoolFlag(flags, "exec", "preview", &value, false, "fallback usage")

	flag, ok := flags.Lookup("preview")
	if !ok {
		t.Fatalf("expected preview flag to exist")
	}
	if flag.Usage != "Show the execution preview before running." {
		t.Fatalf("flag usage = %q, want %q", flag.Usage, "Show the execution preview before running.")
	}
	if flag.DefValue != "false" {
		t.Fatalf("flag default = %q, want %q", flag.DefValue, "false")
	}
}

func TestBindManifestStringFlagUsesTemplateCompletionWhenDeclared(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	flags := cmd.NewFlagSet("test", cmd.ContinueOnError)
	var value string
	builder.bindManifestStringFlag(flags, "create database", "template", &value, "", "fallback usage")

	flag, ok := flags.Lookup("template")
	if !ok {
		t.Fatalf("expected template flag to exist")
	}
	if flag.Usage != "Template name." {
		t.Fatalf("flag usage = %q, want %q", flag.Usage, "Template name.")
	}
	if flag.Completion == nil {
		t.Fatalf("expected template completion to be attached")
	}
}

func TestManifestFlagDefaultStringReadsUserHostDefault(t *testing.T) {
	t.Parallel()

	got := manifestFlagDefaultString("create user", "host", "localhost")
	if got != "%" {
		t.Fatalf("manifest flag default = %q, want %q", got, "%")
	}
}

func TestBindManifestStringFlagUsesEnumMetadataWhenDeclared(t *testing.T) {
	t.Parallel()

	builder := &cliBuilder{}
	flags := cmd.NewFlagSet("test", cmd.ContinueOnError)
	var value string
	builder.bindManifestStringFlag(flags, "create user", "grant", &value, "", "fallback usage")

	flag, ok := flags.Lookup("grant")
	if !ok {
		t.Fatalf("expected grant flag to exist")
	}
	if len(flag.Enum) != 2 || flag.Enum[0] != "all" || flag.Enum[1] != "readonly" {
		t.Fatalf("flag enum = %#v", flag.Enum)
	}
	if flag.Usage != "Database grant mode." {
		t.Fatalf("flag usage = %q, want %q", flag.Usage, "Database grant mode.")
	}
}
