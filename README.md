# dbx

[中文文档](README.zh-CN.md)
[Architecture walkthrough](docs/ARCHITECTURE.md)

`dbx` is a REPL-first MySQL operator CLI written in Go. It keeps day-to-day database work guided and template-driven, shares one command tree between the REPL and one-shot CLI mode, and supports direct, SSH, SOCKS5, and SOCKS5 -> SSH transport paths without shelling out to external SSH tools.

`dbx` does not expose unrestricted user SQL entrypoints. Instead, commands collect typed inputs, validate identifiers, resolve an executable operation spec, preview the plan, and execute safely.

## Highlights

- REPL-first workflow with a session-aware `dbx>` prompt
- Shared command tree between interactive and non-interactive usage
- Canonical verb-first commands: `show`, `create`, `drop`, `use`, `exec`
- Native MySQL, SSH, SOCKS5, and SOCKS5 -> SSH connectivity
- Template precedence: connection > global > builtin
- Dry-run previews, confirmation gates, redacted output, and local audit logs
- Small dependency set and classic Go project layout

## Command Surface

The current user-facing command surface is:

```text
connect [name]

show connections
show connection <name>
show databases
show tables
show table <name>
show columns <table>
show rows <table> [--limit n]
show users
show templates [query] [--tag value]
show context

create connection <name>
create database <name>
create user <name>

drop connection <name>
drop database <name>
drop user <name>

use <name>

exec <operation> [--preview] [--verbose] [--validate]

doctor
audit log
help
exit
```

Notes:

- `show table <name>` prints the `SHOW CREATE TABLE` DDL for that table.
- `show rows <table>` previews rows with a default limit of `10` and a maximum of `100`.
- `exec <operation>` runs a named operation. Today the only operation provider is `template`, but the command surface stays provider-neutral.

In non-interactive mode, prepend `dbx`:

```bash
dbx show connections
dbx --connection prod show tables
dbx --connection prod exec create_database_with_user --preview
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

A typical guided flow:

```text
dbx> create connection prod
dbx> connect prod
dbx(prod)> show databases
dbx(prod)> use app_prod
dbx(prod/app_prod)> show tables
dbx(prod/app_prod)> show table users
dbx(prod/app_prod)> show rows users --limit 20
dbx(prod/app_prod)> create user analytics_ro
dbx(prod/app_prod)> audit log
```

## Non-Interactive CLI

Create a saved direct connection:

```bash
dbx create connection dev \
  --mode direct \
  --host 127.0.0.1 \
  --port 3306 \
  --user root \
  --password-env MYSQL_DEV_PASSWORD \
  --yes
```

Create a saved proxy -> SSH connection and test it:

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
  --ssh-private-key ~/.ssh/id_rsa \
  --test \
  --yes
```

Inspect saved state and operational context:

```bash
dbx show connections
dbx show connection prod
dbx --connection prod show databases
dbx --connection prod --database app_prod show tables
dbx --connection prod --database app_prod show table users
dbx --connection prod --database app_prod show columns users
dbx --connection prod --database app_prod show rows users --limit 20
dbx --connection prod show context --format json
```

Run guided operations:

```bash
dbx --connection prod create database app_demo --yes
dbx --connection prod drop database app_demo --dry-run
dbx --connection prod --database app_prod create user analytics_ro \
  --grant readonly \
  --password-env ANALYTICS_RO_PASSWORD \
  --yes
dbx --connection prod --database app_prod drop user analytics_ro --yes
```

Use named operations explicitly:

```bash
dbx --connection prod show templates
dbx --connection prod show templates database --tag tenant
dbx --connection prod exec create_database_with_user --validate
dbx --connection prod exec create_database_with_user \
  --input database=greenhn_prod \
  --input user_host=% \
  --input password=super-secret \
  --preview
dbx --connection prod create database greenhn_prod \
  --template create_database_with_user \
  --input user_host=% \
  --input password=super-secret \
  --yes
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

Only SOCKS5 proxying is supported. Validation is mode-specific:

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

Password sources may be stored inline, loaded from environment variables, or prompted at runtime:

- database password: `password`, `password_env`, `password_prompt`
- SSH auth: `ssh.private_key`, `ssh.password_env`, `ssh.password`

`doctor` warns on insecure inline secrets and missing password environment variables, but it does not rewrite config for you.

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

Current template input types:

- `string`
- `secret`
- `select`
- `confirm`
- `identifier`
- `int`

Template facts:

- Input keys passed with `--input key=value` must match template input names exactly.
- Secret inputs are redacted from previews, logs, audit records, and JSON output.
- If multiple templates match the same command at the same layer, the REPL asks you to choose and the CLI requires `--template <name>` or `exec <name>`.

Built-in variables include:

```text
{{database}}
{{connection.name}}
{{connection.host}}
{{connection.user}}
```

The repository includes an example template at [examples/templates/create_database_with_user.json](examples/templates/create_database_with_user.json). It matches the `create database` command and adds extra inputs for `user_host` and `password`.

For a detailed end-to-end template walkthrough, including layering, input types, secret handling, and command-to-template matching notes, see [TEMPLATE_STARTKIT.md](docs/TEMPLATE_STARTKIT.md). A Chinese version is available at [TEMPLATE_STARTKIT.zh-CN.md](docs/TEMPLATE_STARTKIT.zh-CN.md).

To try it locally, copy it into your config directory:

```bash
mkdir -p ~/.config/dbx/templates
cp examples/templates/create_database_with_user.json ~/.config/dbx/templates/
```

Then validate or preview it:

```bash
dbx --connection prod exec create_database_with_user --validate
dbx --connection prod create database app_demo \
  --template create_database_with_user \
  --input user_host=% \
  --input password=super-secret \
  --preview
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

Key paths:

- connection config: `~/.config/dbx/{connection}/config.json`
- connection templates: `~/.config/dbx/{connection}/templates/`
- global templates: `~/.config/dbx/templates/`
- session file: `~/.config/dbx/session.json`
- history file: `~/.config/dbx/history`
- audit log: `~/.config/dbx/logs/audit.jsonl`

The session file persists the selected connection and selected database. Command history is persisted locally and trimmed to the most recent `1000` entries.

## Diagnostics And Safety

- `doctor` is static. It checks config shape, password sources, proxy URL shape, SSH auth settings, key file permissions, and `known_hosts` presence without dialing the network path.
- `create connection ... --test` saves the config, then performs a live connectivity check and reports any warning without deleting the saved config.
- Mutating commands require confirmation in the REPL and `--yes` in non-interactive mode unless `--dry-run` or `--preview` is in effect for that command.
- Passwords, proxy passwords, and secret template inputs are redacted from user-facing output and audit logs.
- `audit log` reads the local JSONL audit file and shows the most recent entries instead of failing the original command when audit append fails.

## Architecture

`dbx` stays intentionally small and split by responsibility:

- [cmd/dbx/main.go](cmd/dbx/main.go): process startup, signal-aware shutdown, CLI root
- [internal/app/](internal/app): shared command tree, REPL handlers, one-shot CLI flow, execution orchestration, reporting
- [internal/commandlang/](internal/commandlang): syntax model used by completion and command-aware help
- [internal/config/](internal/config): config loading, session file, history file, audit log, timeout defaults
- [internal/connect/](internal/connect): timeout-aware connection helpers
- [internal/driver/](internal/driver): MySQL, SSH, and SOCKS5 transport implementation
- [internal/template/](internal/template): template resolution and rendering
- [internal/ui/](internal/ui): lightweight prompt helpers and completion-facing UI types
- [internal/ui/editor/](internal/ui/editor): line buffer and completion edit primitives
- [internal/util/](internal/util): validation, paths, layered errors, redaction helpers

The active REPL runtime is currently provided by `pkg.gostartkit.com/cmd`; `internal/repl/` exists in the repository but is not part of the active execution path today.

For a package-by-package code walkthrough, execution flow analysis, and notes on the current implementation shape, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Project Layout

```text
dbx/
├── cmd/
│   └── dbx/
│       └── main.go
├── docs/
│   ├── ARCHITECTURE.md
│   ├── ARCHITECTURE.zh-CN.md
│   ├── CONTRIBUTING.md
│   ├── CONTRIBUTING.zh-CN.md
│   ├── TEMPLATE_STARTKIT.md
│   └── TEMPLATE_STARTKIT.zh-CN.md
├── internal/
│   ├── app/
│   ├── commandlang/
│   ├── config/
│   ├── connect/
│   ├── driver/
│   ├── repl/
│   ├── template/
│   ├── ui/
│   └── util/
├── examples/
├── AGENTS.md
├── LICENSE
├── Makefile
├── README.md
├── README.zh-CN.md
└── go.mod
```

## Installation And Development

Requirements:

- Go `1.25+`
- MySQL reachable through one of the supported transport modes

Common developer commands:

```bash
make fmt
make test
make build
make check
```

Release packaging lives in [scripts/release.sh](scripts/release.sh). The installer helper in [scripts/install.sh](scripts/install.sh) follows the release artifact naming convention and still expects the final GitHub release repo path to be wired through its `REPO` value before publishing.

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for development and review expectations.
