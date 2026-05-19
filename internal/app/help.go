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
audit log             Show recent audit entries

connection create     Create a new saved connection
connection edit <name> Edit a saved connection
connection delete <name> Delete a saved connection
connection show <name> Show a saved connection
connection test [name] [verbose] Test a saved connection
connection doctor [name] Inspect a saved connection statically

create database       Create a database from a template
list databases        List databases on the active connection
drop database         Drop a database from a template
create user           Create a MySQL user
show users            List MySQL users
drop user             Drop a MySQL user
show tables           List tables in the current database
describe [table]      Describe a table in the current database
show grants <user>    Show MySQL grants for a user
use <database>        Select a database for the active REPL session

status                Show the current session status
context               Show the active REPL context
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
  connection test [name] [verbose]
  connection doctor [name]

Examples:
  connection create
  connection edit prod
  connection show dev
  connection test prod
  connection test prod verbose
  connection doctor prod`),
	},
	"connection create": {
		title: "connection create",
		body: strings.TrimSpace(`
Create a new saved connection.

This command:
  - creates ~/.config/dbx/{name}/config.json
  - supports direct, ssh, proxy, and proxy-ssh modes
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
	"connection test": {
		title: "connection test",
		body: strings.TrimSpace(`
Test a saved connection and report which layer succeeds or fails.

This command checks:
  - config
  - proxy when mode is proxy or proxy-ssh
  - ssh when mode is ssh or proxy-ssh
  - mysql

Examples:
  connection test
  connection test prod
  connection test prod verbose
  dbx connection test prod --verbose`),
	},
	"connection doctor": {
		title: "connection doctor",
		body: strings.TrimSpace(`
Inspect a saved connection statically without opening the network path.

This command checks:
  - config structure
  - password source setup
  - proxy URL shape for proxy and proxy-ssh
  - SSH auth and private key files
  - known_hosts file presence and plain host matches

Examples:
  connection doctor
  connection doctor prod`),
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
	"audit log": {
		title: "audit log",
		body: strings.TrimSpace(`
Show recent local audit log entries.

Audit entries are stored at:
  ~/.config/dbx/logs/audit.jsonl

Examples:
  audit log
  dbx audit log --format json`),
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
	"create user": {
		title: "create user",
		body: strings.TrimSpace(`
Create a MySQL user from the resolved template.

This command:
  - collects username, host, and password details
  - can grant access to the current REPL database
  - previews the execution plan
  - requires confirmation before execution

Examples:
  create user
  create user analytics-ro`),
	},
	"list databases": {
		title: "list databases",
		body: strings.TrimSpace(`
List databases on the active connection.

Example:
  list databases`),
	},
	"show users": {
		title: "show users",
		body: strings.TrimSpace(`
List MySQL user accounts on the active connection.

Aliases:
  list users
  show user accounts

Example:
  show users`),
	},
	"show tables": {
		title: "show tables",
		body: strings.TrimSpace(`
List tables in the current database context.

This command:
  - requires an active connection
  - requires a selected database
  - does not require confirmation

Example:
  show tables`),
	},
	"describe": {
		title: "describe",
		body: strings.TrimSpace(`
Describe a table in the current database context.

Usage:
  describe users
  describe table users

This command:
  - requires an active connection
  - requires a selected database
  - does not require confirmation`),
	},
	"show grants": {
		title: "show grants",
		body: strings.TrimSpace(`
Show MySQL grants for a user.

Usage:
  show grants analytics-ro
  show grants analytics-ro localhost

This command:
  - defaults the host to %
  - does not require confirmation`),
	},
	"use": {
		title: "use",
		body: strings.TrimSpace(`
Select a database for the active REPL session.

This command:
  - validates the database name
  - verifies that the database exists
  - updates the session prompt and saved session state

Example:
  use app_prod`),
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
	"drop user": {
		title: "drop user",
		body: strings.TrimSpace(`
Drop a MySQL user from the resolved template.

This command:
  - prompts for the username and host when needed
  - previews the execution plan
  - requires confirmation before execution

Example:
  drop user analytics-ro`),
	},
	"status": {
		title: "status",
		body: strings.TrimSpace(`
Show the current session status.

This includes:
  - active connection name
  - selected database when set
  - connection mode and address
  - timeout settings
  - dry-run state`),
	},
	"context": {
		title: "context",
		body: strings.TrimSpace(`
Show the active REPL context in a compact form.

This includes:
  - selected connection
  - selected database
  - connection mode
  - dry-run state

Aliases:
  ctx`),
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
  ctx      -> context
  ls db    -> list databases
  show databases -> list databases
  show dbs -> list databases
  list users -> show users
  show user accounts -> show users
  desc table -> describe table
  create db -> create database
  drop db   -> drop database
  test conn -> connection test
  doctor conn -> connection doctor
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
