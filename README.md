# dbx

`dbx` is a REPL-first MySQL database CLI focused on guided operations instead of raw SQL. It connects directly, through native SSH, through a SOCKS5 proxy to MySQL, or through a SOCKS5 proxy feeding native SSH, resolves templates from builtin/global/connection layers, and keeps the user flow centered on safe prompts, previews, and confirmations.

## Goals

- REPL-first UX
- No raw SQL from users
- Native SSH database access
- Template-based operations
- Minimal dependencies
- MySQL-only MVP

## Features

- Interactive `dbx>` prompt
- Context-aware TAB completion for commands, connections, databases, tables, and MySQL users
- Lightweight inline hints for common command prefixes
- Explicit command aliases without changing canonical help output
- Direct, proxy, SSH, and proxy-SSH MySQL connections
- Hidden password input
- `known_hosts` SSH host verification
- Configurable connect and query timeouts
- Session reconnect prompt
- Session-scoped dry-run mode
- Persisted command history without readline
- Local JSONL audit log under `~/.config/dbx/logs/audit.jsonl`
- Live `connection test` and static `connection doctor` diagnostics
- Structured JSON error codes for non-interactive mode
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
audit log
connection create
connection edit <name>
connection delete <name>
connection show <name>
connection test [name] [verbose]
connection doctor [name]
status
context
create database
list databases
drop database
create user
show users
drop user
show tables
describe [table]
show grants <user> [host]
use <database>
dry-run on
dry-run off
exit
```

`/` is reserved for command discovery. Operational commands do not use a `/` prefix.

Non-interactive CLI commands:

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
dbx list databases [flags]
dbx show databases [flags]
dbx show dbs [flags]
dbx drop database <name> [flags]
dbx create user <name> [flags]
dbx show users [flags]
dbx drop user <name> [flags]
dbx show tables [flags]
dbx describe <table> [flags]
dbx show grants <user> [host] [flags]

dbx status
dbx context
dbx help
dbx help <command>
```

Global CLI flags:

```text
--connection <name>
--database <name>
--config-dir <path>
--dry-run
--yes
--format text|json
```

## REPL Ergonomics

TAB completion is intentionally lightweight, but it is now operationally context-aware. It does not implement full readline-style editing, but it does understand command position, saved connections, current database context, and lightweight inline hints. Up and Down arrows navigate persisted command history in the interactive REPL.

```text
dbx> conn<TAB>
connect
connections
connection create
connection doctor
connection edit
connection delete
connection show
connection test

dbx> connection <TAB>
create
doctor
edit
delete
show
test

dbx> connect <TAB>
dev
prod

dbx> use <TAB>
app_demo
app_prod

dbx(prod/app_prod)> describe <TAB>
orders
users

dbx> drop user <TAB>
analytics-ro
app_user

dbx(prod)> show grants <TAB>
analytics-ro
app_user
```

Common command-tree examples:

```text
dbx> create <TAB>
database
user

dbx> drop <TAB>
database
user

dbx> show <TAB>
databases
dbs
users
tables
grants

dbx> help <TAB>
connection
create
drop
use
context
```

Supported aliases stay intentionally small and explicit:

```text
q             -> exit
quit          -> exit
conn          -> connect
cx            -> connect
conns         -> connections
ls db         -> list databases
show databases -> list databases
show dbs      -> list databases
create db     -> create database
drop db       -> drop database
list users    -> show users
show user accounts -> show users
describe      -> describe
desc table    -> describe table
ctx           -> context
test conn     -> connection test
doctor conn   -> connection doctor
dry on        -> dry-run on
dry off       -> dry-run off
```

Use `help aliases` inside the REPL to display the alias list.

## Confirmation Behavior

Read-only commands run immediately. This includes commands such as `status`, `connections`, `connection show`, `connection test`, `connection doctor`, `list databases`, `show databases`, `show dbs`, and `show users`.

Mutating commands require confirmation in the REPL unless dry-run is active. For one-shot CLI commands, mutating operations require `--yes` unless `--dry-run` is active for SQL execution commands.

Examples:

```text
dbx(prod)> status
dbx(prod)> show databases
dbx(prod)> connection test prod
```

```bash
dbx --connection prod drop database greenhn-dev --yes
dbx --connection prod drop database greenhn-dev --dry-run
```

Running `dbx` enters the interactive shell:

```bash
dbx
```

Any supported REPL operation can also run as a one-shot CLI command. This is useful for scripts, CI jobs, and release automation.

Session-scoped database selection is REPL-only:

```text
dbx> connect prod
Connected to prod.

dbx(prod)> use analytics_v2
Using database: analytics_v2

dbx(prod/analytics_v2)> status
Connection: prod
Database: analytics_v2
...

dbx(prod/analytics_v2)> context
Connection: prod
Database: analytics_v2
Mode: proxy-ssh
Dry-run: off
```

`use <database>` updates the active REPL session and persists the selected database in `session.json`. One-shot CLI commands stay stateless and instead accept `--database <name>` for a single invocation.

Database names accept letters, numbers, `_`, and `-`. Examples:

```text
greenhn-dev
prod-db
analytics_v2
```

MySQL usernames accept letters, numbers, `_`, and `-`. Examples:

```text
app_user
analytics-ro
service_v2
```

## Configuration

All state lives under:

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

`session.json` stores the last selected connection and, when set from the REPL, the last selected database:

```json
{
  "connection": "prod",
  "database": "app_prod"
}
```

Direct MySQL example:

```json
{
  "version": 1,
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
  "version": 1,
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

Proxy -> SSH -> MySQL example:

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
  }
}
```

Proxy -> MySQL example:

```json
{
  "version": 1,
  "name": "prod-proxy",
  "driver": "mysql",
  "mode": "proxy",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "password_env": "MYSQL_PROD_PASSWORD",
  "proxy": {
    "url": "socks5://proxy_user:proxy_password@127.0.0.1:1080"
  }
}
```

## Operational Inspection

These commands keep `dbx` in its operational REPL lane without turning it into a raw SQL shell:

```text
show tables
describe users
show grants analytics-ro
context
```

`show tables` and `describe` require a selected database context. If none is selected, `dbx` returns:

```text
no database selected; use: use <database>
```

`show grants` defaults the MySQL host to `%` unless a second argument is provided.

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
  "version": 1,
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
  "version": 1,
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
Database name: greenhn-dev
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
  1. CREATE DATABASE IF NOT EXISTS `greenhn-dev` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
  2. CREATE USER IF NOT EXISTS 'greenhn-dev'@'%' IDENTIFIED BY '***'
Dry-run mode is enabled. SQL will be rendered but not executed.
Confirm execution? [y/n] [y]:
[DRY-RUN] Create database
[DRY-RUN] Create user
```

User creation with a database-aware grant:

```text
dbx(prod/app_prod)> create user
Username: analytics-ro
Host [%]:
  1. prompt
  2. env variable
  3. generated password
Password mode [prompt]: generated password
Grant access to current database app_prod? [y/n] [y]:
  1. all
  2. readonly
Privileges [all]: readonly
Template: builtin_create_user (builtin)
Execution Plan
  1. Create MySQL user 'analytics-ro'@'%'
  2. Grant SELECT on `app_prod`.*
Rendered SQL
  1. CREATE USER 'analytics-ro'@'%' IDENTIFIED BY '***'
  2. GRANT SELECT ON `app_prod`.* TO 'analytics-ro'@'%'
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
  3. proxy
  4. proxy-ssh
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

If you choose `proxy` or `proxy-ssh`, `dbx` asks for `Proxy URL` and stores it under `proxy.url`. `proxy` mode does not prompt for SSH settings.

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

Connection diagnostics:

```text
dbx> connection test prod-proxy
[OK] config
[OK] proxy
[OK] mysql
Connection successful.
```

Proxy -> SSH diagnostics:

```text
dbx> connection test prod-bastion
[OK] config
[OK] proxy
[OK] ssh
[OK] mysql
Connection successful.
```

Verbose connection test:

```text
dbx> connection test prod-bastion verbose
[OK] config
     mode: proxy-ssh
     driver: mysql
     config_path: ~/.config/dbx/prod-bastion/config.json
[OK] proxy
     url: socks5://127.0.0.1:1080
     target: bastion.example.com:22
     duration: 81ms
[OK] ssh
     host: bastion.example.com:22
     user: ubuntu
     duration: 130ms
[OK] mysql
     target: 10.0.1.20:3306
     ping: 45ms
Connection successful.
```

Static connection doctor:

```text
dbx> connection doctor prod-proxy
Connection doctor: prod-proxy

[OK] config file exists
[OK] config JSON can be parsed
[OK] mode proxy
[OK] proxy scheme socks5
[WARN] proxy URL contains inline password
       suggestion: avoid saving inline proxy passwords in config
```

Operational inspection in the REPL:

```text
dbx> connect prod
Connected to prod.

dbx(prod)> use app_prod
Using database: app_prod

dbx(prod/app_prod)> show tables
users
orders
audit_logs

dbx(prod/app_prod)> describe users
id               bigint
email            varchar(255)
created_at       datetime

dbx(prod/app_prod)> show grants analytics-ro
GRANT SELECT ON `app_prod`.* TO 'analytics-ro'@'%'
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
[OK] Create database (124ms)
[OK] Create user (92ms)
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

Create a saved proxy -> SSH connection:

```bash
dbx connection create prod-proxy \
  --mode proxy-ssh \
  --host 10.0.1.20 \
  --port 3306 \
  --user root \
  --password-env MYSQL_PROD_PASSWORD \
  --proxy-url socks5://proxy_user:proxy_password@127.0.0.1:1080 \
  --ssh-host bastion.example.com \
  --ssh-port 22 \
  --ssh-user ubuntu \
  --ssh-private-key ~/.ssh/id_rsa
```

Create a saved proxy -> MySQL connection:

```bash
dbx connection create prod-proxy \
  --mode proxy \
  --host 10.0.1.20 \
  --port 3306 \
  --user root \
  --password-env MYSQL_PROD_PASSWORD \
  --proxy-url socks5://proxy_user:proxy_password@127.0.0.1:1080
```

If you add `--test` and the connection test fails, `dbx` still saves the config and prints a warning so you can fix it later with `connection edit <name>`.

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

Test a saved connection and inspect machine-readable steps:

```bash
dbx connection test prod-proxy --format json
```

Example JSON:

```json
{
  "ok": true,
  "connection": "prod-proxy",
  "steps": [
    {"name": "config", "status": "ok"},
    {"name": "proxy", "status": "ok"},
    {"name": "mysql", "status": "ok"}
  ]
}
```

Verbose JSON diagnostics:

```bash
dbx connection test prod-proxy --verbose --format json
```

```json
{
  "ok": true,
  "connection": "prod-proxy",
  "steps": [
    {
      "name": "config",
      "status": "ok",
      "details": {
        "driver": "mysql",
        "mode": "proxy",
        "config_path": "/Users/sam/.config/dbx/prod-proxy/config.json"
      }
    },
    {
      "name": "proxy",
      "status": "ok",
      "details": {
        "url": "socks5://proxy_user:***@127.0.0.1:1080",
        "target": "10.0.1.20:3306",
        "duration_ms": 81
      }
    }
  ]
}
```

Doctor a saved connection without opening the network path:

```bash
dbx connection doctor prod-proxy --format json
```

Example JSON:

```json
{
  "ok": true,
  "connection": "prod-proxy",
  "checks": [
    {"name": "config file exists", "status": "ok"},
    {"name": "proxy URL contains inline password", "status": "warn", "suggestion": "avoid saving inline proxy passwords in config"}
  ]
}
```

Show tables from the current CLI database context:

```bash
dbx --connection prod --database app_prod show tables
```

Describe a table:

```bash
dbx --connection prod --database app_prod describe users
```

Show grants for a MySQL user:

```bash
dbx --connection prod show grants analytics-ro
dbx --connection prod show grants analytics-ro localhost
```

Context output for scripts or quick checks:

```bash
dbx --connection prod --database app_prod context --format json
```

Example JSON:

```json
{
  "ok": true,
  "connection": "prod",
  "database": "app_prod",
  "mode": "proxy-ssh",
  "dry_run": false
}
```

Show tables JSON:

```bash
dbx --connection prod --database app_prod show tables --format json
```

```json
{
  "ok": true,
  "connection": "prod",
  "database": "app_prod",
  "tables": ["users", "orders", "audit_logs"]
}
```

Describe JSON:

```json
{
  "ok": true,
  "connection": "prod",
  "database": "app_prod",
  "table": "users",
  "columns": [
    {
      "name": "id",
      "type": "bigint"
    }
  ]
}
```

Structured JSON error output:

```json
{
  "ok": false,
  "error": {
    "code": "SSH_AUTH_FAILED",
    "message": "ssh authentication failed",
    "layer": "ssh"
  }
}
```

Show recent audit entries:

```bash
dbx audit log
dbx audit log --format json
```

Create a database from a saved connection:

```bash
dbx --connection prod create database app_demo --yes
```

Create a readonly reporting user for the current database:

```bash
dbx --connection prod --database analytics_v2 create user analytics-ro \
  --password-env ANALYTICS_RO_PASSWORD \
  --grant readonly \
  --yes
```

Create a user with a generated password:

```bash
dbx --connection prod create user app_user \
  --generate-password \
  --yes
```

Show and drop users:

```bash
dbx --connection prod show users
dbx --connection prod drop user analytics-ro --host % --yes
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

Show users for scripts:

```bash
dbx --connection prod show users --format json
```

Drop a database safely:

```bash
dbx --connection prod drop database app_demo --yes
```

Inspect status using an explicit saved connection instead of the persisted session:

```bash
dbx status --connection prod --format json
```

Apply a database only for the current CLI call:

```bash
dbx --connection prod --database analytics_v2 status --format json
```

`use <database>` is not available as a standalone CLI command because CLI mode exits after each invocation.

Validation errors render flush-left in both REPL and CLI output:

```text
Error: validation error: validate database name: invalid database name "greenhn dev"
```

For CI and shell scripts, prefer `--format json`, `--yes`, and `--dry-run` where appropriate.

`connection doctor` is static and does not open proxy, SSH, or MySQL connections. `connection test` is live and verifies the actual connection path.

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

dbx --config-dir "$PWD/.dbx" connection doctor ci
dbx --config-dir "$PWD/.dbx" connection test ci

dbx --config-dir "$PWD/.dbx" --connection ci --dry-run --format json create database ci_demo
```

## Troubleshooting Flow

```text
connection doctor prod
connection show prod
connection edit prod
connection test prod
connect prod
```

## Security Notes

- User-facing workflows never require raw SQL input.
- Database identifiers are validated against `[a-zA-Z_][a-zA-Z0-9_]*`.
- Password prompts are hidden when stdin is a terminal.
- Secret template values are redacted from previews.
- `type: "secret"` and legacy `secret: true` inputs are both redacted.
- SSH access is native through Go SSH libraries, not `exec.Command("ssh")`.
- Proxy passwords in `proxy.url` are redacted in user-facing output and JSON summaries.
- Verbose connection test output also redacts proxy passwords and never prints database or SSH passwords.
- Audit log entries store command names, connection names, modes, dry-run state, success, and duration, but never passwords or secret template input values.
- Generated passwords from `create user --generate-password` are shown once in text mode after a successful create, and are never written to JSON output, dry-run previews, or audit logs.
- SSH host verification uses `known_hosts`.
- `DBX_KNOWN_HOSTS` can point to alternate `known_hosts` files if needed.
- `connection doctor` only performs static `known_hosts` checks against plain host entries; it does not verify hashed entries or make network calls.

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
- Proxy support is limited to SOCKS5 URLs such as `socks5://127.0.0.1:1080`.
- Connection configs and JSON templates use schema `version: 1`; missing versions are treated as version 1 for backward compatibility.
- `connection test` reports the first failing layer and stops there; it is a diagnostic command, not a deep network debugger.
- TAB completion is lightweight and does not provide full shell-style line editing.
- REPL history supports persisted Up/Down navigation, but not reverse search or advanced readline behavior.
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
