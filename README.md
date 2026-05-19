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
- Lightweight TAB completion for commands and saved connection names
- Explicit command aliases without changing canonical help output
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

REPL commands:

```text
/
help
help <command>
help aliases
connect
connect <name>
connections
connection create
connection edit <name>
connection delete <name>
connection show <name>
status
create database
list databases
drop database
dry-run on
dry-run off
exit
```

`/` is reserved for command discovery. Operational commands do not use a `/` prefix.

Non-interactive CLI commands:

```text
dbx connect <name>
dbx connections

dbx connection create <name> [flags]
dbx connection edit <name> [flags]
dbx connection delete <name> [flags]
dbx connection show <name>

dbx create database <name> [flags]
dbx list databases [flags]
dbx drop database <name> [flags]

dbx status
dbx help
dbx help <command>
```

Global CLI flags:

```text
--connection <name>
--config-dir <path>
--dry-run
--yes
--format text|json
```

## REPL Ergonomics

TAB completion is intentionally lightweight. It does not implement full readline-style editing, but it does support command and saved-connection completion:

```text
dbx> conn<TAB>
connect
connections
connection create
connection edit
connection delete
connection show

dbx> connection <TAB>
create
edit
delete
show

dbx> connect <TAB>
dev
prod
```

Supported aliases stay intentionally small and explicit:

```text
q             -> exit
quit          -> exit
conn          -> connect
cx            -> connect
conns         -> connections
ls db         -> list databases
show dbs      -> list databases
create db     -> create database
drop db       -> drop database
dry on        -> dry-run on
dry off       -> dry-run off
```

Use `help aliases` inside the REPL to display the alias list.

Running `dbx` enters the interactive shell:

```bash
dbx
```

Any supported REPL operation can also run as a one-shot CLI command. This is useful for scripts, CI jobs, and release automation.

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
  "transaction": true,
  "match": {
    "command": "create database",
    "driver": "mysql"
  },
  "inputs": [
    {
      "name": "database",
      "type": "identifier",
      "prompt": "Database name"
    },
    {
      "name": "charset",
      "type": "select",
      "prompt": "Charset",
      "default": "utf8mb4",
      "options": ["utf8mb4", "utf8"]
    },
    {
      "name": "create_user",
      "type": "confirm",
      "prompt": "Create same-name user?",
      "default": true
    },
    {
      "name": "port",
      "type": "int",
      "prompt": "Port",
      "default": 3306
    },
    {
      "name": "password",
      "type": "secret",
      "prompt": "New user password"
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

dbx> /
Available commands
...

dbx> dry-run on
Dry-run mode is on.

dbx> create database
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

Connection selection with `connect` and no argument:

```text
dbx> connect
1) prod     mysql ssh    10.0.1.20:3306 via bastion.example.com
2) dev      mysql direct 127.0.0.1:3306
Select connection by number or name: 1
Connected to prod.
```

Connection creation flow:

```text
dbx> connection create
Connection name: prod
  1. direct
  2. ssh
Connection mode [direct]: ssh
Database host: 10.0.1.20
Database port [3306]:
Database user: root
  1. prompt every time
  2. env variable
  3. save password
Password handling [prompt every time]: env variable
Environment variable name [MYSQL_PROD_PASSWORD]:
Connect timeout seconds [10]:
Query timeout seconds [30]:
SSH host: bastion.example.com
SSH port [22]:
SSH user: ubuntu
  1. private key
  2. env variable
SSH auth [private key]:
SSH private key [~/.ssh/id_rsa]:
Test connection? [y/n] [y]:
Connection successful.
Save connection? [y/n] [y]:
Saved: ~/.config/dbx/prod/config.json
Connect now? [y/n] [y]:
```

Connection inspection:

```text
dbx> connection show prod
Name: prod
Driver: mysql
Mode: ssh

Host: 10.0.1.20:3306
User: root

Password:
  env: MYSQL_PROD_PASSWORD

SSH:
  host: bastion.example.com:22
  user: ubuntu
  private_key: ~/.ssh/id_rsa
```

Connection editing and deletion:

```text
dbx> connection edit prod
...interactive update flow...

dbx> connection delete prod
Delete connection "prod"? [y/n] [n]: y
Deleted connection prod.
```

Transactional template execution:

```text
dbx> create database
Database name: appdb
Charset [utf8mb4]:
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
Confirm execution? [y/n] [y]: y
[OK] Create database
[OK] Create user
Committed transaction.
Database appdb created.
```

## Non-Interactive CLI

Create a saved connection:

```bash
dbx connection create prod \
  --mode ssh \
  --host 10.0.1.20 \
  --port 3306 \
  --user root \
  --password-env MYSQL_PROD_PASSWORD \
  --ssh-host bastion.example.com \
  --ssh-port 22 \
  --ssh-user ubuntu \
  --ssh-private-key ~/.ssh/id_rsa \
  --connect-timeout 10 \
  --query-timeout 30
```

Edit only the fields you want to change:

```bash
dbx connection edit prod \
  --host 10.0.1.30 \
  --user admin \
  --query-timeout 60
```

Delete a connection without an interactive confirmation:

```bash
dbx connection delete prod --yes
```

Show a connection in JSON with secrets redacted:

```bash
dbx connection show prod --format json
```

Create a database from a saved connection:

```bash
dbx --connection prod create database app_demo --yes
```

Render a template without executing it:

```bash
dbx --connection prod \
  --dry-run \
  --format json \
  create database app_demo \
  --template create_database_with_user \
  --input password=secret123
```

Example dry-run JSON output:

```json
{
  "ok": true,
  "connection": "prod",
  "command": "create database",
  "template": "create_database_with_user",
  "dry_run": true,
  "actions": [
    {
      "description": "Create database",
      "sql": "CREATE DATABASE IF NOT EXISTS `app_demo`",
      "status": "dry-run"
    },
    {
      "description": "Create user",
      "sql": "CREATE USER IF NOT EXISTS 'app_demo'@'%' IDENTIFIED BY '***'",
      "status": "dry-run"
    }
  ]
}
```

List databases for scripts:

```bash
dbx --connection prod list databases --format json
```

Drop a database safely:

```bash
dbx --connection prod drop database app_demo --yes
```

Inspect status using an explicit saved connection instead of the persisted session:

```bash
dbx status --connection prod --format json
```

For CI and shell scripts, prefer `--format json`, `--yes`, and `--dry-run` where appropriate.

## Installation

Build locally:

```bash
make build
```

Install a release artifact with the helper script:

```bash
sh scripts/install.sh
```

Override the repository, version, or install directory if needed:

```bash
REPO="OWNER/dbx" VERSION="v0.2.0" INSTALL_DIR="$HOME/.local/bin" sh scripts/install.sh
```

Create release artifacts locally:

```bash
sh scripts/release.sh
```

This writes platform archives and `checksums.txt` to `dist/`.

CI or release automation example:

```bash
dbx --config-dir "$PWD/.dbx" connection create ci \
  --mode direct \
  --host 127.0.0.1 \
  --port 3306 \
  --user root \
  --password-env MYSQL_CI_PASSWORD

dbx --config-dir "$PWD/.dbx" --connection ci --dry-run --format json create database ci_demo
```

## Security Notes

- User-facing workflows never require raw SQL input.
- Database identifiers are validated against `[a-zA-Z_][a-zA-Z0-9_]*`.
- Password prompts are hidden when stdin is a terminal.
- Secret template values are redacted from previews.
- `type: "secret"` and legacy `secret: true` inputs are both redacted.
- SSH access is native through Go SSH libraries, not `exec.Command("ssh")`.
- SSH host verification uses `known_hosts`.
- `DBX_KNOWN_HOSTS` can point to alternate `known_hosts` files if needed.

## Developer Workflow

```bash
make fmt
make vet
make test
make build
make check
make release
```

## Known Limitations

- MySQL is the only supported database in the MVP.
- REPL history is persisted, but arrow-key navigation is intentionally not implemented.
- TAB completion is lightweight and does not provide full shell-style line editing.
- Dry-run is session-scoped and not persisted.
- A saved connection that uses `password_prompt` still needs an interactive terminal when a command must actually open the database.
- SSH verification expects a prepared `known_hosts` file.
- MySQL can implicitly commit around statements such as `CREATE DATABASE` and `CREATE USER`, so `transaction: true` is best-effort for those workflows.
- One-shot CLI commands reuse the same core services as the REPL, but the product direction remains REPL-first.

## Future Roadmap

- More guided database operations within the same template-driven model
- Better history ergonomics on top of the persisted history file
- Additional safe introspection commands like table listing and schema description
- Stronger SSH verification configuration options while keeping the default path simple

## Examples

Sample configs and templates are available in [examples/](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/examples).
