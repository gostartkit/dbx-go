package commandlang

import (
	"strings"
	"testing"
)

func TestDefaultRegistryLookup(t *testing.T) {
	t.Parallel()

	registry := testRegistry()

	spec, matched := registry.LookupCommand([]string{"exec"})
	if spec == nil || spec.Name != "exec" || matched != 1 {
		t.Fatalf("exec lookup = %#v matched=%d", spec, matched)
	}

	spec, matched = registry.LookupCommand([]string{"quit"})
	if spec == nil || spec.Name != "exit" || matched != 1 {
		t.Fatalf("alias lookup = %#v matched=%d", spec, matched)
	}

	spec, matched = registry.LookupCommand([]string{"template", "render"})
	if spec == nil || spec.Name != "render" || matched != 2 {
		t.Fatalf("nested lookup = %#v matched=%d", spec, matched)
	}

	if flag := spec.FindFlag("--var"); flag == nil || flag.Name != "--var" {
		t.Fatalf("flag lookup = %#v", flag)
	}

	execSpec, _ := registry.LookupCommand([]string{"exec"})
	if arg := execSpec.ArgSpec(0); arg == nil || arg.ValueType != ValueOperation {
		t.Fatalf("arg spec = %#v", arg)
	}
}

func TestValidateProgramWithSchema(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		Commands: []*CommandSpec{
			{
				Name: "exec",
				Args: []*ArgSpec{{Name: "operation", Required: true, ValueType: ValueOperation}},
				Flags: []*FlagSpec{
					{Name: "--mode", ValueType: ValueEnum, EnumValues: []string{"readonly", "admin"}},
					{Name: "--target", ValueType: ValueString, Required: true},
				},
			},
			{
				Name: "template",
				Subcommands: []*CommandSpec{
					{Name: "render", Args: []*ArgSpec{{Name: "template", Required: true, ValueType: ValueTemplate}}},
				},
			},
		},
	}

	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "unknown command", line: "unknown foo", want: "unknown command"},
		{name: "unknown subcommand", line: "template missing", want: "unknown subcommand"},
		{name: "unknown flag", line: "exec op --bogus", want: "unknown flag"},
		{name: "missing required arg", line: "exec", want: "missing required arg"},
		{name: "missing flag value", line: "exec op --mode", want: "missing flag value"},
		{name: "enum invalid", line: "exec op --mode owner", want: "invalid value"},
		{name: "too many args", line: "exec op extra", want: "too many positional args"},
		{name: "missing required flag", line: "exec op --mode readonly", want: "missing required flag"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			errors := registry.ValidateProgram(ParseProgram(test.line))
			if len(errors) == 0 {
				t.Fatalf("expected validation error")
			}
			if !strings.Contains(errors[0].Message, test.want) {
				t.Fatalf("message = %q, want substring %q", errors[0].Message, test.want)
			}
		})
	}
}

func TestRegistryHelp(t *testing.T) {
	t.Parallel()

	registry := testRegistry()
	for _, topic := range []string{"exec", "template", "connection"} {
		doc, ok := registry.Help(topic)
		if !ok {
			t.Fatalf("expected help for %q", topic)
		}
		body := doc.Title + "\n" + doc.Body
		if !strings.Contains(body, "Usage:") {
			t.Fatalf("help body missing usage for %q: %q", topic, body)
		}
		if topic == "exec" && (!strings.Contains(body, "<operation>") || !strings.Contains(body, "--dry-run")) {
			t.Fatalf("exec help missing args/flags: %q", body)
		}
		if topic == "template" && !strings.Contains(body, "Subcommands:") {
			t.Fatalf("template help missing subcommands: %q", body)
		}
		if topic == "connection" && !strings.Contains(body, "use") {
			t.Fatalf("connection help missing subcommands: %q", body)
		}
	}
}
