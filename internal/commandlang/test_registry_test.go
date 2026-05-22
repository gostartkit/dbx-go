package commandlang

import "pkg.gostartkit.com/cmd"

func testRegistry() *Registry {
	spec := cmd.AppSpec{
		Commands: []cmd.CommandSpec{
			{
				Name:      "exec",
				UsageLine: "dbx exec <operation> [flags]",
				Short:     "Execute a named operation.",
				Positionals: []cmd.PositionalSpec{{
					Name:          "operation",
					Usage:         "Operation name to execute.",
					Required:      true,
					Kind:          "operation",
					CompletionKey: "operation",
				}},
				Flags: []cmd.FlagSpec{
					{Name: "dry-run", Type: "bool", Usage: "Render and validate without applying changes."},
					{Name: "role", Usage: "Role to grant."},
				},
			},
			{
				Name:      "help",
				UsageLine: "dbx help [topic]",
				Short:     "Show help for a command or topic.",
				Positionals: []cmd.PositionalSpec{{
					Name:          "topic",
					Usage:         "Command or topic to describe.",
					CompletionKey: "topic",
				}},
			},
			{
				Name:      "template",
				UsageLine: "dbx template <subcommand>",
				Short:     "Template maintenance commands.",
				SubCommands: []cmd.CommandSpec{
					{
						Name:      "render",
						UsageLine: "dbx template render <template>",
						Short:     "Render a template preview.",
						Positionals: []cmd.PositionalSpec{{
							Name:          "template",
							Usage:         "Template name.",
							Required:      true,
							Kind:          "template",
							CompletionKey: "template",
						}},
						Flags: []cmd.FlagSpec{
							{Name: "var", Usage: "Template input override in key=value form."},
						},
					},
				},
			},
			{
				Name:      "connection",
				UsageLine: "dbx connection <subcommand>",
				Short:     "Connection management commands.",
				SubCommands: []cmd.CommandSpec{
					{
						Name:      "use",
						UsageLine: "dbx connection use <connection>",
						Short:     "Select a saved connection.",
						Positionals: []cmd.PositionalSpec{{
							Name:          "name",
							Usage:         "Saved connection name.",
							Required:      true,
							Kind:          "connection",
							CompletionKey: "connection",
						}},
					},
				},
			},
			{
				Name:      "database",
				UsageLine: "dbx database <subcommand>",
				Short:     "Database selection commands.",
				SubCommands: []cmd.CommandSpec{
					{
						Name:      "use",
						UsageLine: "dbx database use <database>",
						Short:     "Select the current database.",
						Positionals: []cmd.PositionalSpec{{
							Name:          "database",
							Usage:         "Database name.",
							Required:      true,
							Kind:          "database",
							CompletionKey: "database",
						}},
					},
				},
			},
			{
				Name:      "show",
				UsageLine: "dbx show <subcommand>",
				Short:     "Inspect configuration and database state.",
				SubCommands: []cmd.CommandSpec{
					{
						Name:      "templates",
						UsageLine: "dbx show templates [query] [--tag value]",
						Short:     "Show resolved workflow templates.",
						Positionals: []cmd.PositionalSpec{{
							Name:          "query",
							Usage:         "Optional template search query.",
							CompletionKey: "template",
						}},
						Flags: []cmd.FlagSpec{
							{Name: "tag", Usage: "Filter templates by tag.", CompletionKey: "template-tag"},
						},
					},
				},
			},
			{
				Name:      "exit",
				UsageLine: "dbx exit",
				Short:     "Exit the REPL.",
				Aliases:   []string{"quit"},
			},
		},
	}
	return RegistryFromAppSpec(spec)
}
