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
show connections      Show saved connections
show connection <name> Show one saved connection
create connection <name> Create a saved connection
edit connection <name> Edit a saved connection
drop connection <name> Drop a saved connection
test connection <name> Test a saved connection
doctor connection <name> Inspect a saved connection statically

show databases        Show databases on the active connection
show tables           Show tables in the selected database
show columns <table>  Show table columns
describe <table>      Describe a table
show indexes <table>  Show table indexes
show foreign keys <table> Show table foreign keys
show create table <table> Show CREATE TABLE output
show table status [table] Show table status
show grants <user> [host] Show grants for a MySQL user
show processlist      Show active MySQL processes
show variables [pattern] Show MySQL variables
show triggers         Show triggers
show views            Show views
show users            Show MySQL users

create database <name> Create a database
drop database <name>  Drop a database
create user <name>    Create a MySQL user
drop user <name>      Drop a MySQL user
count rows <table>    Count table rows
peek rows <table>     Peek table rows
sample rows <table>   Sample table rows
truncate table <table> Truncate a table
rename table <from> <to> Rename a table

show templates        Show available templates
show template <name>  Show one template
run template <name>   Run a template
validate template <name> Validate a template

use database <name>   Select the current database
context               Show the current operational context
clear                 Clear terminal output
exit                  Exit the REPL`),
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
  edit connection <name>
  drop connection <name>
  test connection <name>
  doctor connection <name>`),
		},
		"connection create": {
			title: "create connection",
			body: strings.TrimSpace(`
Create a saved connection.

This command writes:
  ~/.config/dbx/{name}/config.json

Usage:
  create connection <name> [flags]`),
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
Run a workflow template.

Usage:
  run template <name> [--preview] [--verbose]`),
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
			title: "context",
			body: strings.TrimSpace(`
Show the current connection, database, and dry-run mode.

Usage:
  context`),
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
	entries["edit connection"] = entries["connection edit"]
	entries["drop connection"] = entries["connection delete"]
	entries["show connection"] = entries["connection show"]
	entries["test connection"] = entries["connection test"]
	entries["doctor connection"] = entries["connection doctor"]
	entries["show template"] = entries["describe template"]
	entries["run template"] = entries["template run"]
	entries["validate template"] = entries["template validate"]
	entries["use database"] = entries["use"]

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
