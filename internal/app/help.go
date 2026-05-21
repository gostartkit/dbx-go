package app

import (
	"fmt"
	"io"
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
Core commands:
  connect <name>       Connect to a saved connection
  use database <name>  Select the current database
  doctor               Inspect the selected connection
  audit log            Show audit history
  exit                 Exit the shell

Examples:
  connect
  connect prod

  show
  show connections
  show tables
  show templates --tag readonly

  create
  create connection prod --host 10.0.1.20 --user root
  create database appdb

  run
  run template seed-users --validate
  run sql @schema.sql`),
		},
		"connect": {
			title: "connect",
			body: strings.TrimSpace(`
Connect to a saved connection.

Usage:
  connect <name>`),
		},
		"show": {
			title: "show",
			body: strings.TrimSpace(`
Inspect saved configuration and database state.

Usage:
  dbx show <subcommand>

Subcommands:
  connections
  connection <name>
  context
  databases
  tables
  table <name>
  columns <table>
  rows <table>
  templates [query] [--tag value]`),
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
  drop connection <name>`),
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
		"create": {
			title: "create",
			body: strings.TrimSpace(`
Create saved connections and database resources.

Usage:
  dbx create <subcommand>

Subcommands:
  connection <name>
  database <name>
  user <name>`),
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
		"drop": {
			title: "drop",
			body: strings.TrimSpace(`
Drop saved connections and database resources.

Usage:
  dbx drop <subcommand>

Subcommands:
  connection <name>
  database <name>
  user <name>`),
		},
		"drop user": {
			title: "drop user",
			body: strings.TrimSpace(`
Drop a MySQL user from the resolved template.

Usage:
  drop user <name> [flags]`),
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
		"show templates": {
			title: "show templates",
			body: strings.TrimSpace(`
Show resolved workflow templates.

Usage:
  show templates [query] [--tag value]`),
		},
		"run": {
			title: "run",
			body: strings.TrimSpace(`
Run database workflows and SQL entrypoints.

Usage:
  dbx run <subcommand>

Subcommands:
  template <name>
  sql <sql-or-file>`),
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
Inspect the selected connection statically without opening the network path.

Checks:
  config structure
  password sources
  proxy URL shape
  SSH auth settings
  known_hosts presence

Usage:
  doctor`),
		},
		"exit": {
			title: "exit",
			body: strings.TrimSpace(`
Exit the REPL.

Aliases:
  quit
  q`),
		},
	}

	entries["show connections"] = entries["connections"]
	entries["create connection"] = entries["connection create"]
	entries["drop connection"] = entries["connection delete"]
	entries["show connection"] = entries["connection show"]
	entries["run template"] = entries["template run"]
	entries["use database"] = entries["use"]
	entries["show context"] = entries["context"]

	return entries
}()

func printHelpTopic(prompt printer, topic string) error {
	topic = normalizeHelpTopic(topic)

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

type writerPrinter struct {
	w io.Writer
}

func (p writerPrinter) Println(args ...any) {
	fmt.Fprintln(p.w, args...)
}
