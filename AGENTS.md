# AGENTS.md

## Project Overview

`dbx` is a REPL-first MySQL database CLI written in Go.

Current product direction:

- Guided operations instead of raw SQL from users
- Interactive REPL as the primary entrypoint
- One-shot CLI commands as secondary automation surfaces
- Native SSH support
- SOCKS5 proxy support
- Template-driven database operations
- Minimal third-party dependencies
- Classic Go project layout
- `pkg.gostartkit.com/cmd` for non-interactive CLI registration

## Core Principles

### 1. REPL First

Preferred usage:

```bash
dbx
```

This enters:

```text
dbx>
```

One-shot CLI commands such as `dbx connection show prod` and `dbx create database appdb` are supported, but the product should stay optimized for the interactive flow first.

### 2. No SQL From Users

Users should not provide unrestricted SQL.

`dbx` is responsible for:

- collecting parameters
- validating identifiers and typed inputs
- generating SQL
- previewing execution plans
- executing safely

Examples:

```text
create database
drop database
```

Not:

```sql
CREATE DATABASE ...
DROP DATABASE ...
```

### 3. Native Transport Support

SSH must use Go libraries, not shelling out:

```go
golang.org/x/crypto/ssh
```

Forbidden:

```go
exec.Command("ssh")
```

SOCKS5 proxy support must use:

```go
golang.org/x/net/proxy
```

Supported connection paths today:

```text
direct    -> db
ssh       -> ssh -> db
proxy     -> proxy -> db
proxy-ssh -> proxy -> ssh -> db
```

Only SOCKS5 is supported. Do not add HTTP CONNECT or proxy chains unless explicitly requested in future scope.

### 4. Minimal Dependencies

Allowed dependencies:

```text
pkg.gostartkit.com/cmd
github.com/go-sql-driver/mysql
golang.org/x/crypto/ssh
golang.org/x/net/proxy
golang.org/x/term
```

Avoid introducing:

```text
cobra
viper
promptui
survey
readline
gorm
xorm
tablewriter
```

Prefer the standard library whenever practical.

## Project Layout

Use the classic Go structure already present in the repository:

```text
dbx/
├── cmd/
│   └── dbx/
│       └── main.go
├── internal/
│   ├── app/
│   ├── repl/
│   ├── config/
│   ├── connect/
│   ├── template/
│   ├── driver/
│   ├── ui/
│   └── util/
├── examples/
├── AGENTS.md
├── README.md
└── go.mod
```

Keep files and functions small. Prefer extending the current packages over introducing new architectural layers unless there is a clear need.

## REPL Design

Commands no longer use a `/` prefix.

`/` is reserved only for command discovery.

Current REPL commands:

```text
/
help
help <command>
help aliases

connect
connect <name>
connections
audit log

connection create
connection edit <name>
connection delete <name>
connection show <name>
connection test [name] [verbose]
connection doctor [name]

create database
show databases
drop database
create user
count rows <table>
peek rows <table>
sample rows <table>
show users
drop user
show tables
show columns <table>
describe [table]
show foreign keys <table>
show indexes [table]
show create table <table>
show table status [table]
show grants <user> [host]
show processlist
show triggers
show variables [name|pattern]
show views
show templates
describe template <name>
template run <name>
truncate table <table>
rename table <from> <to>
use <database>
context

status
dry-run on
dry-run off
exit
```

### REPL Input UX

Current interactive behavior includes:

- lightweight TAB completion
- persisted history
- Up/Down history navigation
- hidden password input when stdin is a terminal
- graceful Ctrl+C handling

Do not replace the existing lightweight prompt approach with a readline-style framework.

## Interactive UX Rules

Interactive commands should:

- ask step-by-step
- provide defaults
- provide constrained choices where possible
- preview execution plans
- confirm before execution
- redact secrets in previews and output

Example:

```text
Database name:
Charset:
Collation:
Confirm execution?
```

## Non-Interactive CLI

`dbx` without arguments enters the REPL.

`dbx <command> ...` runs non-interactive mode through `pkg.gostartkit.com/cmd`.

REPL and one-shot CLI commands should continue to share the same underlying services and execution paths. Avoid duplicating business logic across interactive and non-interactive entrypoints.

Current non-interactive command families include:

```text
dbx connect <name>
dbx connections
dbx audit log

dbx connection create <name> [flags]
dbx connection edit <name> [flags]
dbx connection delete <name> [flags]
dbx connection show <name>
dbx connection test <name> [--verbose]
dbx connection doctor <name>

dbx create database <name> [flags]
dbx show databases [flags]
dbx show dbs [flags]
dbx list databases [flags]
dbx drop database <name> [flags]
dbx create user <name> [flags]
dbx count rows <table> [flags]
dbx peek rows <table> [flags]
dbx sample rows <table> [flags]
dbx show users [flags]
dbx drop user <name> [flags]
dbx show tables [flags]
dbx show columns <table> [flags]
dbx describe <table> [flags]
dbx show foreign keys <table> [flags]
dbx show indexes <table> [flags]
dbx show create table <table> [flags]
dbx show table status [table] [flags]
dbx show grants <user> [host] [flags]
dbx show processlist [flags]
dbx show triggers [flags]
dbx show variables [name|pattern] [flags]
dbx show views [flags]
dbx show templates [flags]
dbx describe template <name> [flags]
dbx template run <name> [flags]
dbx truncate table <table> [flags]
dbx rename table <from> <to> [flags]

dbx status
dbx context
dbx help
dbx help <command>
```

Global CLI flags currently supported:

```text
--connection <name>
--database <name>
--config-dir <path>
--dry-run
--yes
--format text|json
```

## Configuration Directory

All user state lives under:

```text
~/.config/dbx/
```

Current layout:

```text
~/.config/dbx/
  history
  logs/
    audit.jsonl
  session.json
  templates/

  dev/
    config.json
    templates/

  prod/
    config.json
    templates/
```

Connection configs are stored at:

```text
~/.config/dbx/{connection}/config.json
```

## Connection Configuration

Current config fields may include:

```json
{
  "version": 1,
  "name": "prod-proxy",
  "driver": "mysql",
  "mode": "proxy-ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "password_env": "MYSQL_PROD_PASSWORD",
  "proxy": {
    "url": "socks5://proxy_user:proxy_password@127.0.0.1:1080"
  },
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  },
  "timeout": {
    "connect_seconds": 10,
    "query_seconds": 30
  }
}
```

Validation rules must stay strict and mode-specific:

- `direct` rejects proxy and SSH config
- `ssh` requires SSH config and rejects proxy config
- `proxy` requires proxy config and rejects SSH config
- `proxy-ssh` requires both proxy and SSH config

Secrets must never be printed raw. Proxy URLs must redact inline proxy passwords in all user-facing output.

## Template System

Three layers exist:

```text
connection template
> global template
> builtin template
```

Directories:

```text
~/.config/dbx/templates/
~/.config/dbx/{connection}/templates/
```

Current template features include:

- schema version `1`
- typed inputs: `string`, `secret`, `select`, `confirm`, `identifier`, `int`
- optional `required` and `description` metadata
- transaction flag support
- dry-run execution
- secret redaction in previews

Templates are safe operational workflows, not a general scripting language.
Do not add loops, conditionals, embedded scripts, remote registries, or plugin systems.
Secret inputs must never appear in history, audit logs, dry-run SQL, or JSON output.

Built-in variables include:

```text
{{database}}
{{connection.name}}
{{connection.host}}
{{connection.user}}
```

## SQL Safety Rules

Users must never provide unrestricted SQL.

Do not use one global identifier rule for every object type.

Current validation rules are split by purpose:

- database names allow letters, numbers, `_`, and `-`
- MySQL usernames allow letters, numbers, `_`, and `-`
- stricter internal identifiers may still use `[a-zA-Z_][a-zA-Z0-9_]*`

All identifiers must still be validated before SQL rendering, and MySQL object names must be quoted safely in generated SQL.

## Diagnostics

Two different connection diagnostics exist today:

- `connection test`: live connectivity test
- `connection doctor`: static configuration inspection

`connection test` checks layers in order depending on mode:

```text
direct    -> config, mysql
ssh       -> config, ssh, mysql
proxy     -> config, proxy, mysql
proxy-ssh -> config, proxy, ssh, mysql
```

`connection doctor` must stay static:

- no network calls
- no live proxy dialing
- no live SSH dialing
- no live MySQL connection

Static checks may inspect files, environment variables, plain `known_hosts` entries, and config structure.

## UI Rules

Do not introduce readline-style libraries.

Continue using the lightweight prompt approach built on:

```go
bufio.Reader
fmt.Print
golang.org/x/term
```

Keep prompt helpers simple and explicit:

- `Ask`
- `Choose`
- `Confirm`
- `AskPassword`

## Session State

The REPL maintains in-process session state and persists the selected connection and selected database in `session.json`.

The active session concept includes:

- selected connection config
- selected database name for the REPL session
- active `*sql.DB` when connected
- session-scoped dry-run mode
- reconnect candidate on startup

## Driver Strategy

MVP remains MySQL-only.

Do not introduce generic dialect abstractions prematurely.

If transport behavior changes, prefer extending the current MySQL driver integration and registered dialers rather than inventing a large cross-driver abstraction.

## Coding Style

Requirements:

- small files
- small functions
- explicit error handling
- no panic
- no hidden side effects
- composition over abstraction-heavy designs

Preferred:

```go
if err != nil {
    return err
}
```

Forbidden:

```go
panic(err)
```

## Error Handling

Preserve layered errors where possible.

Current layer names in user-facing flows include:

```text
config
validation
proxy
ssh
mysql
template
sql execution
shutdown
```

Non-interactive JSON errors should expose stable error codes and sanitized messages. Current codes include values such as:

```text
CONFIG_NOT_FOUND
VALIDATION_FAILED
UNSUPPORTED_VERSION
SSH_AUTH_FAILED
PROXY_DIAL_FAILED
MYSQL_ACCESS_DENIED
TEMPLATE_NOT_FOUND
SQL_EXECUTION_FAILED
```

Keep audit logging best-effort:

- append JSONL records to `~/.config/dbx/logs/audit.jsonl`
- never log passwords, proxy passwords, or secret template inputs
- do not fail the user command if audit logging itself fails

Keep secrets out of:

- error strings
- previews
- JSON output
- connection summaries
- logs

## Logging

Do not add logging frameworks.

Allowed:

```go
fmt.Println
log.Printf
```

## Current Scope

Current implemented scope includes:

```text
REPL
non-interactive CLI
connection management
direct MySQL
SSH MySQL
proxy MySQL
proxy-SSH MySQL
connection test
connection doctor
create/list/drop database
create/show/drop user
show tables
describe table
show grants
template system
global templates
connection templates
dry-run
audit log
history persistence
lightweight completion
graceful shutdown
README / packaging basics
```

Still out of scope:

```text
new database drivers
ORM
migration system
AI SQL
schema diff
proxy chain
HTTP proxy mode
plugin system
full readline/TUI framework
web UI
```

## Future Extensions

Possible future commands may include:

```text
list tables
create table
table desc
template run
history export
schema inspection
```

Keep the current architecture easy to extend, but do not over-engineer for hypothetical future features.
