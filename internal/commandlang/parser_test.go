package commandlang

import "testing"

func TestParseProgramCommandsAndFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantName   string
		wantArgs   []string
		wantFlags  map[string]string
		wantQuoted []string
		wantPipes  int
		wantErrs   int
	}{
		{
			name:      "exec flag value",
			input:     `exec create-user --role readonly`,
			wantName:  "exec",
			wantArgs:  []string{"create-user"},
			wantFlags: map[string]string{"--role": "readonly"},
		},
		{
			name:      "exec flag equals",
			input:     `exec create-user --role=readonly`,
			wantName:  "exec",
			wantArgs:  []string{"create-user"},
			wantFlags: map[string]string{"--role": "readonly"},
		},
		{
			name:      "connection create",
			input:     `connection create dev --driver mysql`,
			wantName:  "connection",
			wantArgs:  []string{"create", "dev"},
			wantFlags: map[string]string{"--driver": "mysql"},
		},
		{
			name:      "template render",
			input:     `template render create-user --var name=alice`,
			wantName:  "template",
			wantArgs:  []string{"render", "create-user"},
			wantFlags: map[string]string{"--var": "name=alice"},
		},
		{
			name:      "help",
			input:     `help connection`,
			wantName:  "help",
			wantArgs:  []string{"connection"},
			wantFlags: map[string]string{},
		},
		{
			name:      "show columns",
			input:     `show columns`,
			wantName:  "show",
			wantArgs:  []string{"columns"},
			wantFlags: map[string]string{},
		},
		{
			name:      "pipeline",
			input:     `exec grant-readonly | exec audit-log`,
			wantName:  "exec",
			wantArgs:  []string{"grant-readonly"},
			wantFlags: map[string]string{},
			wantPipes: 1,
		},
		{
			name:       "quoted values",
			input:      `exec "create user" --role "read only"`,
			wantName:   "exec",
			wantArgs:   []string{"create user"},
			wantFlags:  map[string]string{"--role": "read only"},
			wantQuoted: []string{"create user", "read only"},
		},
		{
			name:      "escaped spaces",
			input:     `exec create\ user --role read\ only`,
			wantName:  "exec",
			wantArgs:  []string{"create user"},
			wantFlags: map[string]string{"--role": "read only"},
		},
		{
			name:      "unterminated quote",
			input:     `exec "create user`,
			wantName:  "exec",
			wantArgs:  []string{"create user"},
			wantFlags: map[string]string{},
			wantErrs:  1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			program := ParseProgram(test.input)
			if len(program.Commands) == 0 {
				t.Fatalf("expected at least one command")
			}
			command := program.Commands[0]
			if command.Name != test.wantName {
				t.Fatalf("command name = %q, want %q", command.Name, test.wantName)
			}
			if got := argValues(command.Args); !equalStrings(got, test.wantArgs) {
				t.Fatalf("args = %#v, want %#v", got, test.wantArgs)
			}
			if got := flagValues(command.Flags); !equalStringMap(got, test.wantFlags) {
				t.Fatalf("flags = %#v, want %#v", got, test.wantFlags)
			}
			if len(program.Pipelines) != test.wantPipes {
				t.Fatalf("pipelines = %d, want %d", len(program.Pipelines), test.wantPipes)
			}
			if len(program.Errors) != test.wantErrs {
				t.Fatalf("errors = %d, want %d", len(program.Errors), test.wantErrs)
			}
			if len(test.wantQuoted) > 0 {
				quoted := quotedValues(command)
				if !equalStrings(quoted, test.wantQuoted) {
					t.Fatalf("quoted values = %#v, want %#v", quoted, test.wantQuoted)
				}
			}
		})
	}
}

func TestBuildSyntaxContext(t *testing.T) {
	t.Parallel()

	knownPaths := [][]string{
		{"exec"},
		{"connection", "create"},
		{"template", "render"},
		{"help"},
		{"show", "columns"},
	}

	tests := []struct {
		name            string
		input           string
		cursor          int
		wantPath        []string
		wantParent      []string
		wantCommandName bool
		wantSubcommand  bool
		wantArg         bool
		wantFlagName    bool
		wantFlagValue   bool
		wantFlag        string
		wantArgIndex    int
	}{
		{
			name:         "exec arg position",
			input:        "exec ",
			cursor:       len([]rune("exec ")),
			wantPath:     []string{"exec"},
			wantParent:   []string{"exec"},
			wantArg:      true,
			wantArgIndex: 0,
		},
		{
			name:          "flag value spaced",
			input:         "exec create-user --role ",
			cursor:        len([]rune("exec create-user --role ")),
			wantPath:      []string{"exec"},
			wantParent:    []string{"exec"},
			wantFlagValue: true,
			wantFlag:      "--role",
		},
		{
			name:          "flag value equals",
			input:         "exec create-user --role=",
			cursor:        len([]rune("exec create-user --role=")),
			wantPath:      []string{"exec"},
			wantParent:    []string{"exec"},
			wantFlagValue: true,
			wantFlag:      "--role",
		},
		{
			name:         "connection create arg",
			input:        "connection create ",
			cursor:       len([]rune("connection create ")),
			wantPath:     []string{"connection", "create"},
			wantParent:   []string{"connection", "create"},
			wantArg:      true,
			wantArgIndex: 0,
		},
		{
			name:         "flag name partial",
			input:        "connection create dev --dr",
			cursor:       len([]rune("connection create dev --dr")),
			wantPath:     []string{"connection", "create"},
			wantFlagName: true,
			wantFlag:     "--dr",
			wantParent:   []string{"connection", "create"},
			wantArgIndex: 0,
		},
		{
			name:         "template render arg",
			input:        "template render ",
			cursor:       len([]rune("template render ")),
			wantPath:     []string{"template", "render"},
			wantParent:   []string{"template", "render"},
			wantArg:      true,
			wantArgIndex: 0,
		},
		{
			name:         "help topic arg",
			input:        "help ",
			cursor:       len([]rune("help ")),
			wantPath:     []string{"help"},
			wantParent:   []string{"help"},
			wantArg:      true,
			wantArgIndex: 0,
		},
		{
			name:           "show subcommand",
			input:          "show ",
			cursor:         len([]rune("show ")),
			wantPath:       []string{"show"},
			wantParent:     []string{"show"},
			wantSubcommand: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			program := ParseProgram(test.input)
			ctx := BuildSyntaxContext(program, test.cursor, knownPaths)
			if !equalStrings(ctx.CommandPath, test.wantPath) {
				t.Fatalf("command path = %#v, want %#v", ctx.CommandPath, test.wantPath)
			}
			if !equalStrings(ctx.ParentPath, test.wantParent) {
				t.Fatalf("parent path = %#v, want %#v", ctx.ParentPath, test.wantParent)
			}
			if ctx.InCommandName != test.wantCommandName || ctx.InSubcommand != test.wantSubcommand || ctx.InArg != test.wantArg || ctx.InFlagName != test.wantFlagName || ctx.InFlagValue != test.wantFlagValue {
				t.Fatalf("syntax flags = %+v", ctx)
			}
			if ctx.CurrentFlag != test.wantFlag {
				t.Fatalf("current flag = %q, want %q", ctx.CurrentFlag, test.wantFlag)
			}
			if ctx.ArgIndex != test.wantArgIndex {
				t.Fatalf("arg index = %d, want %d", ctx.ArgIndex, test.wantArgIndex)
			}
		})
	}
}

func argValues(args []*ArgNode) []string {
	values := make([]string, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	return values
}

func flagValues(flags []*FlagNode) map[string]string {
	values := make(map[string]string, len(flags))
	for _, flag := range flags {
		if flag.Value != nil {
			values[flag.Name] = flag.Value.Value
			continue
		}
		values[flag.Name] = ""
	}
	return values
}

func quotedValues(command *CommandNode) []string {
	values := make([]string, 0)
	for _, arg := range command.Positionals {
		if arg.Quoted {
			values = append(values, arg.Value)
		}
	}
	for _, flag := range command.Flags {
		if flag.Value != nil && flag.Value.Quoted {
			values = append(values, flag.Value.Value)
		}
	}
	return values
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func equalStringMap(left map[string]string, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
