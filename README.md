# dbx

[中文文档](README.zh-CN.md)

`dbx` is a REPL-first MySQL operator CLI. It keeps common database work guided and template-driven, supports direct, SSH, SOCKS5 proxy, and SOCKS5 -> SSH transport paths, and also exposes explicit one-shot entrypoints such as `run template` and `run sql` for automation.

## Highlights

- REPL-first workflow with a session-aware `dbx>` prompt
- Canonical verb-first command families: `show`, `create`, `drop`, `run`, and `use`
- Shared command tree between the interactive REPL and non-interactive CLI
- Native MySQL, SSH, SOCKS5, and SOCKS5 -> SSH connectivity
- Template precedence: connection > global > builtin
- Dry-run previews, confirmation prompts, redacted output, and local audit logging
- Small dependency set and classic Go project layout

## Command Model

`dbx` now documents the verb-first command surface as the canonical interface. Use these forms in new docs, scripts, and examples:

```text
connect [name]

show connections
show connection <name>
show databases
show tables
show table <name>
show columns <table>
show rows <table> [--limit n]
show templates [query] [--tag value]
show context

create connection <name>
create database <name>
create user <name>

drop connection <name>
drop database <name>
drop user <name>

use database <name>

run template <name> [--preview] [--verbose] [--validate]
run sql <sql-or-file>

doctor
audit log
help
exit
```

The REPL and the one-shot CLI share the same command tree. In scripts, prepend `dbx`:

```bash
dbx show connections
dbx --connection prod show tables
dbx --connection prod run template create_database_with_user --preview
```

## Quick Start

Build the binary:

```bash
make build
```

Start the REPL:

```bash
go run ./cmd/dbx
```

Typical interactive flow:

```text
dbx> create connection prod
dbx> connect prod
dbx(prod)> show databases
dbx(prod)> use database app_prod
dbx(prod/app_prod)> show tables
dbx(prod/app_prod)> run template create_database_with_user --preview
dbx(prod/app_prod)> run sql @schema.sql
```

Guided operations remain the primary UX. `run sql` is an explicit escape hatch for direct SQL execution when you need it.

## Non-Interactive Examples

Create a saved direct connection:

```bash
dbx create connection dev \
  --mode direct \
  --host 127.0.0.1 \
  --port 3306 \
  --user root \
  --password-env MYSQL_DEV_PASSWORD
```

Create a saved proxy -> SSH connection:

```bash
dbx create connection prod-proxy \
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

Inspect saved state and current context:

```bash
dbx show connections
dbx --connection prod show connection prod
dbx --connection prod show databases
dbx --connection prod --database app_prod show tables
dbx --connection prod --database app_prod show table users
dbx --connection prod --database app_prod show columns users
dbx --connection prod --database app_prod show rows users --limit 20
dbx --connection prod --database app_prod show context --format json
```

Run guided workflows:

```bash
dbx --connection prod create database app_demo --yes
dbx --connection prod drop database app_demo --dry-run
dbx --connection prod --database app_prod create user analytics_ro --yes
dbx --connection prod show templates
dbx --connection prod show templates --tag tenant
dbx --connection prod run template create_database_with_user \
  --input database=greenhn_prod \
  --input user_host=% \
  --input password-env=GREENHN_PASSWORD \
  --preview
```

Run direct SQL:

```bash
dbx --connection prod run sql "SELECT 1"
dbx --connection prod run sql @schema.sql
dbx --connection prod --database app_prod run sql migrations/bootstrap.sql
```

Inspect diagnostics and audit history:

```bash
dbx --connection prod doctor
dbx audit log
dbx audit log --format json
```

## Connection Modes

Supported transport paths:

```text
direct    -> db
ssh       -> ssh -> db
proxy     -> proxy -> db
proxy-ssh -> proxy -> ssh -> db
```

Only SOCKS5 proxying is supported. Validation stays mode-specific:

- `direct` rejects proxy and SSH settings
- `ssh` requires SSH settings and rejects proxy settings
- `proxy` requires proxy settings and rejects SSH settings
- `proxy-ssh` requires both proxy and SSH settings

Example connection config:

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

## Templates

Templates are safe operational workflows, not a general scripting runtime.

Resolution order:

```text
connection template
> global template
> builtin template
```

Template directories:

```text
~/.config/dbx/templates/
~/.config/dbx/{connection}/templates/
```

Current template inputs include typed values such as `string`, `secret`, `select`, `confirm`, `identifier`, and `int`. Secret inputs are redacted from previews, logs, and JSON output.

Example template file: [examples/templates/create_database_with_user.json](examples/templates/create_database_with_user.json)

That example creates a database, creates a same-name MySQL user, and grants privileges using these inputs:

- `database`
- `charset`
- `collation`
- `user_host`
- `password`

Useful template commands:

```bash
dbx --connection prod show templates
dbx --connection prod show templates --tag grant
dbx --connection prod run template create_database_with_user --validate
dbx --connection prod run template create_database_with_user --preview
dbx --connection prod run template create_database_with_user --verbose --yes
```

## Configuration Layout

All user state lives under `~/.config/dbx/`:

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

Connection configs are stored at `~/.config/dbx/{connection}/config.json`.

## Diagnostics And Safety

- `doctor` is static. It checks config shape, password sources, proxy URL shape, SSH auth settings, and `known_hosts` presence without dialing the network path.
- `create connection ... --test` performs a live connectivity check before returning. If the live test fails, the config is still saved and a warning is reported so you can fix it later.
- Mutating commands require confirmation in the REPL and `--yes` in non-interactive mode unless `--dry-run` or `--preview` is in effect.
- Passwords, proxy passwords, and secret template inputs are redacted from user-facing output and audit logs.

## Architecture

`dbx` stays intentionally small and split by responsibility:

- [cmd/dbx/main.go](cmd/dbx/main.go): process startup, signal-aware shutdown, CLI root
- [internal/app/](internal/app): shared command tree, REPL handlers, one-shot CLI flow, reporting
- [internal/repl/](internal/repl): minimal REPL loop
- [internal/config/](internal/config): config loading, session file, history file, timeout defaults
- [internal/connect/](internal/connect): driver-facing timeout application
- [internal/driver/](internal/driver): MySQL and SSH transport implementation
- [internal/template/](internal/template): template resolution and rendering
- [internal/ui/](internal/ui): lightweight prompt helpers
- [internal/util/](internal/util): validation, paths, layered errors, output helpers

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
├── README.zh-CN.md
└── go.mod
```

## Installation And Development

Requirements:

- Go 1.25+
- MySQL reachable through one of the supported transport modes

Common developer commands:

```bash
make fmt
make test
make build
make check
```

Release packaging lives in [scripts/release.sh](scripts/release.sh). Install helpers live in [scripts/install.sh](scripts/install.sh).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development and review expectations.
