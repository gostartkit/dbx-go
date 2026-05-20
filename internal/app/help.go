package app

import (
	"fmt"
	"strings"
)

type helpEntry struct {
	title string
	body  string
}

var helpEntries = func() map[string]helpEntry {
	entries := map[string]helpEntry{
		"": {
			title: "dbx commands",
			body: strings.TrimSpace(`
help                  Show command help
help <command>        Show help for a command

connect <name>        Connect to a saved connection
use database <name>   Select the current database

show connections      Show saved connections
show connection <name> Show one saved connection
show context          Show the current operational context
show databases        Show databases on the active connection
show tables           Show tables in the selected database
show table <name>     Show table details
show columns <table>  Show table columns
show rows <table>     Show table rows
show templates        Show available templates
show template <name>  Show one template

describe <table>      Describe a table

create connection <name> Create or overwrite a saved connection
create database <name> Create a database
create user <name>    Create a MySQL user

drop connection <name> Drop a saved connection
drop database <name>  Drop a database
drop user <name>      Drop a MySQL user

run template <name>   Run or validate a template
run sql <sql-or-file> Run raw SQL or a SQL file

doctor                Inspect the selected connection
doctor connection <name> Inspect a saved connection statically

audit log             Show audit history
exit                  Exit the shell`),
		},
		"connect": {
			title: "connect",
			body: strings.TrimSpace(`
Connect to a saved connection.

Usage:
  connect <name>`),
		},
		"connections": {
			title: "show connections",
			body: strings.TrimSpace(`
Show all saved connections.

Usage:
  show connections`),
		},
		"connection": {
			title: "connection commands",
			body: strings.TrimSpace(`
Connection commands use a single verb-first style.

Commands:
  show connections
  show connection <name>
  create connection <name>
  drop connection <name>
  doctor connection <name>`),
		},
		"connection create": {
			title: "create connection",
			body: strings.TrimSpace(`
Create a saved connection.

This command writes:
  ~/.config/dbx/{name}/config.json

Usage:
  create connection <name> [--overwrite] [flags]`),
		},
		"connection edit": {
			title: "edit connection",
			body: strings.TrimSpace(`
Edit a saved connection in place.

Usage:
  edit connection <name> [flags]`),
		},
		"connection delete": {
			title: "drop connection",
			body: strings.TrimSpace(`
Drop a saved connection after confirmation.

Usage:
  drop connection <name> [flags]`),
		},
		"connection show": {
			title: "show connection",
			body: strings.TrimSpace(`
Show a saved connection with secrets redacted.

Usage:
  show connection <name>`),
		},
		"connection test": {
			title: "test connection",
			body: strings.TrimSpace(`
Test a saved connection and report each layer.

Checks:
  config
  proxy when required
  ssh when required
  mysql

Usage:
  test connection <name> [--verbose]`),
		},
		"connection doctor": {
			title: "doctor connection",
			body: strings.TrimSpace(`
Inspect a saved connection statically without opening the network path.

Checks:
  config structure
  password sources
  proxy URL shape
  SSH auth settings
  known_hosts presence

Usage:
  doctor connection <name>`),
		},
		"audit log": {
			title: "audit log",
			body: strings.TrimSpace(`
Show recent audit entries from:
  ~/.config/dbx/logs/audit.jsonl

Usage:
  audit log`),
		},
		"create database": {
			title: "create database",
			body: strings.TrimSpace(`
Create a database from the resolved template.

Usage:
  create database <name> [flags]`),
		},
		"drop database": {
			title: "drop database",
			body: strings.TrimSpace(`
Drop a database from the resolved template.

Usage:
  drop database <name> [flags]`),
		},
		"create user": {
			title: "create user",
			body: strings.TrimSpace(`
Create a MySQL user from the resolved template.

Usage:
  create user <name> [flags]`),
		},
		"show rows": {
			title: "show rows",
			body: strings.TrimSpace(`
Show rows from a table.

Usage:
  show rows <table> [--limit n]`),
		},
		"drop user": {
			title: "drop user",
			body: strings.TrimSpace(`
Drop a MySQL user from the resolved template.

Usage:
  drop user <name> [flags]`),
		},
		"count rows": {
			title: "count rows",
			body: strings.TrimSpace(`
Count rows in a table.

Usage:
  count rows <table>`),
		},
		"peek rows": {
			title: "peek rows",
			body: strings.TrimSpace(`
Peek a bounded number of rows from a table.

Usage:
  peek rows <table> [--limit value]`),
		},
		"sample rows": {
			title: "sample rows",
			body: strings.TrimSpace(`
Sample a bounded number of rows from a table.

Usage:
  sample rows <table> [--limit value]`),
		},
		"show databases": {
			title: "show databases",
			body: strings.TrimSpace(`
Show databases on the selected connection.

Usage:
  show databases [flags]`),
		},
		"show tables": {
			title: "show tables",
			body: strings.TrimSpace(`
Show tables in the selected database.

Usage:
  show tables`),
		},
		"show table": {
			title: "show table",
			body: strings.TrimSpace(`
Show detailed table output for one table.

Usage:
  show table <name>`),
		},
		"show columns": {
			title: "show columns",
			body: strings.TrimSpace(`
Show columns for a table in the selected database.

Usage:
  show columns <table>`),
		},
		"show foreign keys": {
			title: "show foreign keys",
			body: strings.TrimSpace(`
Show foreign keys for a table in the selected database.

Usage:
  show foreign keys <table>`),
		},
		"show indexes": {
			title: "show indexes",
			body: strings.TrimSpace(`
Show indexes for a table in the selected database.

Usage:
  show indexes <table>`),
		},
		"show create table": {
			title: "show create table",
			body: strings.TrimSpace(`
Show CREATE TABLE for a table in the selected database.

Usage:
  show create table <table>`),
		},
		"show table status": {
			title: "show table status",
			body: strings.TrimSpace(`
Show table status for one table or all tables.

Usage:
  show table status [table]`),
		},
		"show grants": {
			title: "show grants",
			body: strings.TrimSpace(`
Show grants for a MySQL user.

Usage:
  show grants <user> [host]`),
		},
		"show processlist": {
			title: "show processlist",
			body: strings.TrimSpace(`
Show the active MySQL processlist.

Usage:
  show processlist`),
		},
		"show triggers": {
			title: "show triggers",
			body: strings.TrimSpace(`
Show triggers in the selected database.

Usage:
  show triggers`),
		},
		"show variables": {
			title: "show variables",
			body: strings.TrimSpace(`
Show MySQL system variables.

Usage:
  show variables [pattern]`),
		},
		"show views": {
			title: "show views",
			body: strings.TrimSpace(`
Show views in the selected database.

Usage:
  show views`),
		},
		"show users": {
			title: "show users",
			body: strings.TrimSpace(`
Show MySQL users.

Usage:
  show users`),
		},
		"describe": {
			title: "describe",
			body: strings.TrimSpace(`
Describe a table in the selected database.

Usage:
  describe <table>`),
		},
		"show templates": {
			title: "show templates",
			body: strings.TrimSpace(`
Show resolved workflow templates.

Usage:
  show templates [query] [--tag value]`),
		},
		"describe template": {
			title: "show template",
			body: strings.TrimSpace(`
Show a resolved workflow template by name.

Usage:
  show template <name> [--verbose]`),
		},
		"template run": {
			title: "run template",
			body: strings.TrimSpace(`
Run or validate a workflow template.

Usage:
  run template <name> [--preview] [--verbose] [--validate]`),
		},
		"run sql": {
			title: "run sql",
			body: strings.TrimSpace(`
Run raw SQL text or load SQL from a file.

Usage:
  run sql "SELECT 1"
  run sql @schema.sql
  run sql schema.sql`),
		},
		"template validate": {
			title: "validate template",
			body: strings.TrimSpace(`
Validate a workflow template definition.

Usage:
  validate template <name>`),
		},
		"truncate table": {
			title: "truncate table",
			body: strings.TrimSpace(`
Truncate a table in the selected database.

Usage:
  truncate table <table>`),
		},
		"rename table": {
			title: "rename table",
			body: strings.TrimSpace(`
Rename a table in the selected database.

Usage:
  rename table <from> <to>`),
		},
		"use": {
			title: "use database",
			body: strings.TrimSpace(`
Select the current database.

Usage:
  use database <name>`),
		},
		"context": {
			title: "show context",
			body: strings.TrimSpace(`
Show the current connection, database, and dry-run mode.

Usage:
  show context`),
		},
		"doctor": {
			title: "doctor",
			body: strings.TrimSpace(`
Inspect the current connection, or pick one when running interactively.

Usage:
  doctor
  doctor connection <name>`),
		},
		"status": {
			title: "status",
			body: strings.TrimSpace(`
Legacy status summary for internal compatibility.

Prefer:
  context`),
		},
		"exit": {
			title: "exit",
			body: strings.TrimSpace(`
Exit the REPL.

Aliases:
  quit
  q`),
		},
		"clear": {
			title: "clear",
			body: strings.TrimSpace(`
Clear terminal output.`),
		},
	}

	entries["show connections"] = entries["connections"]
	entries["create connection"] = entries["connection create"]
	entries["drop connection"] = entries["connection delete"]
	entries["show connection"] = entries["connection show"]
	entries["doctor connection"] = entries["connection doctor"]
	entries["show template"] = entries["describe template"]
	entries["run template"] = entries["template run"]
	entries["use database"] = entries["use"]
	entries["show context"] = entries["context"]

	return entries
}()

func printHelpTopic(prompt printer, topic string) error {
	topic = normalizeHelpTopic(topic)
	if topic == "" {
		topic = ""
	}

	if entry, ok := helpEntries[topic]; ok {
		if entry.title != "" {
			prompt.Println(entry.title)
		}
		if entry.body != "" {
			prompt.Println(entry.body)
		}
		return nil
	}

	if spec, ok := commandSpecByPath(topic); ok {
		if spec.Path != "" {
			prompt.Println(spec.Path)
		}
		if spec.Description != "" {
			prompt.Println(spec.Description)
		}
		return nil
	}

	return fmt.Errorf("unknown help topic %q; use help", topic)
}

func normalizeHelpTopic(topic string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(topic)), " ")
}

type printer interface {
	Println(args ...any)
}
