# dbx

`dbx` is a REPL-first MySQL database CLI focused on guided operations instead of raw SQL. It connects directly or through native SSH, resolves templates from builtin/global/connection layers, and keeps the user flow centered on safe prompts, previews, and confirmations.

## Goals

- REPL-first UX
- No raw SQL from users
- Native SSH database access
- Template-based operations
- Minimal dependencies
- MySQL-only MVP

## Features

- Interactive `dbx>` prompt
- Direct and SSH MySQL connections
- Hidden password input
- `known_hosts` SSH host verification
- Configurable connect and query timeouts
- Session reconnect prompt
- Session-scoped dry-run mode
- Persisted command history without readline
- Builtin, global, and connection-level templates

## Architecture

`dbx` is intentionally small and split by responsibility:

- [cmd/dbx/main.go](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/cmd/dbx/main.go): process startup, signal-aware shutdown, CLI root
- [internal/app/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/internal/app): REPL command handlers, session flow, reconnect, dry-run, reporting
- [internal/repl/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/internal/repl): minimal REPL loop
- [internal/config/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/internal/config): config loading, session file, history file, timeout defaults
- [internal/connect/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/internal/connect): driver-facing timeout application
- [internal/driver/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/internal/driver): MySQL and SSH transport implementation
- [internal/template/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/internal/template): template resolution and rendering
- [internal/ui/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/internal/ui): lightweight prompt helpers
- [internal/util/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/internal/util): validation, path expansion, layered errors

## Project Layout

```text
dbx/
├── cmd/
│   └── dbx/
│       └── main.go
├── internal/
│   ├── app/
│   ├── config/
│   ├── connect/
│   ├── driver/
│   ├── repl/
│   ├── template/
│   ├── ui/
│   └── util/
├── examples/
├── AGENTS.md
├── CONTRIBUTING.md
├── LICENSE
├── Makefile
├── README.md
└── go.mod
```

## Commands

```text
/
/help
/connect
/connections
/status
/create database
/list databases
/drop database
/dry-run on
/dry-run off
/exit
```

Running `dbx` enters the interactive shell:

```bash
dbx
```

## Configuration

All state lives under:

```text
~/.config/dbx/
  history
  session.json
  templates/
  dev/
    config.json
    templates/
  prod/
    config.json
    templates/
```

Direct MySQL example:

```json
{
  "name": "dev",
  "driver": "mysql",
  "mode": "direct",
  "host": "127.0.0.1",
  "port": 3306,
  "user": "root",
  "password_env": "MYSQL_DEV_PASSWORD",
  "timeout": {
    "connect_seconds": 10,
    "query_seconds": 30
  }
}
```

SSH MySQL example:

```json
{
  "name": "prod",
  "driver": "mysql",
  "mode": "ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "password_env": "MYSQL_PROD_PASSWORD",
  "timeout": {
    "connect_seconds": 10,
    "query_seconds": 30
  },
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  }
}
```

## Template Precedence

Templates resolve in this order:

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

Global template example:

```json
{
  "name": "create_database_with_user",
  "match": {
    "command": "create database",
    "driver": "mysql"
  },
  "inputs": [
    {
      "name": "password",
      "prompt": "New user password",
      "secret": true
    }
  ],
  "actions": [
    {
      "type": "sql",
      "description": "Create database",
      "sql": "CREATE DATABASE IF NOT EXISTS `{{database}}` CHARACTER SET {{charset}} COLLATE {{collation}}"
    },
    {
      "type": "sql",
      "description": "Create user",
      "sql": "CREATE USER IF NOT EXISTS '{{database}}'@'%' IDENTIFIED BY '{{password}}'"
    }
  ]
}
```

Connection template example:

```text
~/.config/dbx/prod/templates/drop_database_guarded.json
```

```json
{
  "name": "drop_database_guarded",
  "match": {
    "command": "drop database",
    "driver": "mysql"
  },
  "actions": [
    {
      "type": "sql",
      "description": "Drop database on production",
      "sql": "DROP DATABASE IF EXISTS `{{database}}`"
    }
  ]
}
```

## Typical Flow

```text
$ dbx
Reconnect previous session "prod"? [y/n]: y
Reconnected to prod.
Available commands:
  /                Show all commands
  /help            Show all commands
  ...

dbx> /dry-run on
Dry-run mode is on.

dbx> /create database
Database name: appdb
  1. utf8mb4
  2. utf8
Charset [utf8mb4]:
  1. utf8mb4_unicode_ci
  2. utf8mb4_general_ci
Collation [utf8mb4_unicode_ci]:
New user password:
Template: create_database_with_user (global)
Source: ~/.config/dbx/templates/create_database_with_user.json
Execution Plan
  1. Create database
  2. Create user
Rendered SQL
  1. CREATE DATABASE IF NOT EXISTS `appdb` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
  2. CREATE USER IF NOT EXISTS 'appdb'@'%' IDENTIFIED BY '***'
Dry-run mode is enabled. SQL will be rendered but not executed.
Confirm execution? [y/n] [y]:
[DRY-RUN] Create database
[DRY-RUN] Create user
```

## Security Notes

- User-facing workflows never require raw SQL input.
- Database identifiers are validated against `[a-zA-Z_][a-zA-Z0-9_]*`.
- Password prompts are hidden when stdin is a terminal.
- Secret template values are redacted from previews.
- SSH access is native through Go SSH libraries, not `exec.Command("ssh")`.
- SSH host verification uses `known_hosts`.
- `DBX_KNOWN_HOSTS` can point to alternate `known_hosts` files if needed.

## Developer Workflow

```bash
make fmt
make vet
make test
make build
```

## Known Limitations

- MySQL is the only supported database in the MVP.
- REPL history is persisted, but arrow-key navigation is intentionally not implemented.
- Dry-run is session-scoped and not persisted.
- SSH verification expects a prepared `known_hosts` file.
- The CLI entrypoint is intentionally REPL-first; one-shot subcommands are not a focus.

## Future Roadmap

- More guided database operations within the same template-driven model
- Better history ergonomics on top of the persisted history file
- Additional safe introspection commands like table listing and schema description
- Stronger SSH verification configuration options while keeping the default path simple

## Examples

Sample configs and templates are available in [examples/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/examples).
