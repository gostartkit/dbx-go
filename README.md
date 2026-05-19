# dbx

`dbx` is an interactive REPL-first database CLI for MySQL with native SSH support.

The tool is designed around guided operations instead of raw SQL. Users start `dbx`, choose a configured connection, and run slash commands such as `/create database` or `/list databases`.

## Features

- REPL-first workflow
- No raw SQL from users
- Native SSH MySQL access with `golang.org/x/crypto/ssh`
- MySQL support for the MVP
- Template-based operations
- Builtin, global, and connection-level template priority
- Minimal dependencies
- `pkg.gostartkit.com/cmd v0.1.9`

## Commands

Inside the REPL, `dbx` supports:

```text
/
/help
/connect
/connections
/status
/create database
/list databases
/drop database
/exit
```

Running `dbx` starts the REPL directly:

```bash
dbx
```

## Configuration

All configuration lives under:

```text
~/.config/dbx/
  session.json
  templates/
  dev/
    config.json
    templates/
  prod/
    config.json
    templates/
```

Example connection config:

```json
{
  "name": "prod",
  "driver": "mysql",
  "mode": "ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "password_env": "MYSQL_PROD_PASSWORD",
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  }
}
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
  "password_env": "MYSQL_DEV_PASSWORD"
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
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  }
}
```

## Template Priority

Templates are resolved in this order:

```text
connection template
> global template
> builtin template
```

Supported directories:

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

Typical `/create database` flow:

```text
dbx> /connect
  1. dev
Connection name [dev]:
Confirm execution? [y/n] [y]:
Connected to dev.

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
Execution plan:
  1. Create database
     CREATE DATABASE IF NOT EXISTS `appdb` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
  2. Create user
     CREATE USER IF NOT EXISTS 'appdb'@'%' IDENTIFIED BY '***'
Confirm execution? [y/n] [y]:
```

## Notes

- `dbx` validates database identifiers with `[a-zA-Z_][a-zA-Z0-9_]*`.
- SSH uses a native Go SSH client and does not shell out to `ssh`.
- SSH host verification uses `known_hosts`. If `~/.ssh/known_hosts` is missing, `dbx` returns a first-run error telling you how to add the host key.
- Secret prompts use hidden terminal input when stdin is a terminal.

## Examples

Sample configs and templates are available in [`examples/`](/Users/sam/Dev/work/gostartkit/stub/golang/dbx/examples).
