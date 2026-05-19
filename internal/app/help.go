package app

import (
	"fmt"
	"strings"
)

type helpEntry struct {
	title string
	body  string
}

var helpEntries = map[string]helpEntry{
	"": {
		title: "Available commands",
		body: strings.TrimSpace(`
/                     Show all commands
help                  Show command help
help <command>        Show help for a command
help aliases          Show supported aliases

connect               Connect to a saved connection
connect <name>        Connect to a specific saved connection
connections           List saved connections

connection create     Create a new saved connection
connection edit <name> Edit a saved connection
connection delete <name> Delete a saved connection
connection show <name> Show a saved connection

create database       Create a database from a template
list databases        List databases on the active connection
drop database         Drop a database from a template

status                Show the current session status
dry-run on            Preview SQL without executing it
dry-run off           Disable dry-run mode
exit                  Exit dbx`),
	},
	"connection": {
		title: "connection",
		body: strings.TrimSpace(`
Manage saved connections.

Subcommands:
  connection create
  connection edit <name>
  connection delete <name>
  connection show <name>

Examples:
  connection create
  connection edit prod
  connection show dev`),
	},
	"connection create": {
		title: "connection create",
		body: strings.TrimSpace(`
Create a new saved connection.

This command:
  - creates ~/.config/dbx/{name}/config.json
  - optionally tests the connection
  - optionally connects immediately

Examples:
  connection create`),
	},
	"connection edit": {
		title: "connection edit",
		body: strings.TrimSpace(`
Edit an existing saved connection.

This command:
  - loads the current config
  - prompts field-by-field with defaults
  - preserves unspecified values
  - rewrites config.json safely

Example:
  connection edit prod`),
	},
	"connection delete": {
		title: "connection delete",
		body: strings.TrimSpace(`
Delete a saved connection.

This command:
  - asks for confirmation
  - removes ~/.config/dbx/{name}/
  - clears the session if that connection is active

Example:
  connection delete prod`),
	},
	"connection show": {
		title: "connection show",
		body: strings.TrimSpace(`
Show a saved connection summary.

Secrets are redacted in the output.

Example:
  connection show prod`),
	},
	"connect": {
		title: "connect",
		body: strings.TrimSpace(`
Connect to a saved connection.

Usage:
  connect
  connect <name>

When no name is provided, dbx prompts for a selection.`),
	},
	"connections": {
		title: "connections",
		body: strings.TrimSpace(`
List all saved connections.

Example:
  connections`),
	},
	"create database": {
		title: "create database",
		body: strings.TrimSpace(`
Create a database from the resolved template.

This command:
  - collects identifier and template inputs
  - previews the execution plan
  - requires confirmation before execution

Example:
  create database`),
	},
	"list databases": {
		title: "list databases",
		body: strings.TrimSpace(`
List databases on the active connection.

Example:
  list databases`),
	},
	"drop database": {
		title: "drop database",
		body: strings.TrimSpace(`
Drop a database from the resolved template.

This command:
  - prompts for a database choice
  - previews the execution plan
  - stops on first execution failure

Example:
  drop database`),
	},
	"status": {
		title: "status",
		body: strings.TrimSpace(`
Show the current session status.

This includes:
  - active connection name
  - connection mode and address
  - timeout settings
  - dry-run state`),
	},
	"dry-run": {
		title: "dry-run",
		body: strings.TrimSpace(`
Control session-scoped dry-run mode.

Usage:
  dry-run on
  dry-run off`),
	},
	"aliases": {
		title: "aliases",
		body: strings.TrimSpace(`
Supported aliases:
  q        -> exit
  quit     -> exit
  conn     -> connect
  cx       -> connect
  conns    -> connections
  ls db    -> list databases
  show dbs -> list databases
  create db -> create database
  drop db   -> drop database
  dry on    -> dry-run on
  dry off   -> dry-run off`),
	},
	"exit": {
		title: "exit",
		body: strings.TrimSpace(`
Exit the REPL gracefully.`),
	},
}

func printHelpTopic(prompt printer, topic string) error {
	topic = normalizeHelpTopic(topic)
	entry, ok := helpEntries[topic]
	if !ok {
		return fmt.Errorf("unknown help topic %q; use / or help", topic)
	}

	if entry.title != "" {
		prompt.Println(entry.title)
	}
	if entry.body != "" {
		prompt.Println(entry.body)
	}
	return nil
}

func normalizeHelpTopic(topic string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(topic)), " ")
}

type printer interface {
	Println(args ...any)
}
