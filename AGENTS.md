````markdown
# AGENTS.md

## Project Overview

`dbx` is an interactive database command line tool written in Go.

Goals:

- No SQL required from users
- Interactive REPL-first experience
- SSH-native database access
- Template-based database operations
- Global and connection-level templates
- Minimal third-party dependencies
- Classic Go project layout
- Use `pkg.gostartkit.com/cmd` as the CLI framework

---

# Core Principles

## 1. REPL First

The primary entrypoint is interactive mode.

Preferred usage:

```bash
dbx
````

Enter:

```text
dbx>
```

NOT:

```bash
dbx create database xxx
```

One-shot CLI commands are secondary.

---

## 2. No SQL From Users

Users should not write SQL directly.

dbx is responsible for:

* collecting parameters
* validating inputs
* generating SQL
* executing SQL safely

Example:

```text
/create database
```

NOT:

```sql
CREATE DATABASE ...
```

---

## 3. Native SSH Support

SSH must be implemented using Go SSH libraries.

Forbidden:

```go
exec.Command("ssh")
```

Required:

```go
golang.org/x/crypto/ssh
```

Database connections must work over native SSH connections.

---

## 4. Minimal Dependencies

Allowed dependencies:

```text
pkg.gostartkit.com/cmd
database drivers
golang.org/x/crypto/ssh
```

Avoid:

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

Prefer Go standard library whenever possible.

---

# Project Layout

Must use classic Go project structure.

```text
dbx/
├── cmd/
│   └── dbx/
│       └── main.go
│
├── internal/
│   ├── app/
│   ├── repl/
│   ├── config/
│   ├── connect/
│   ├── template/
│   ├── driver/
│   ├── ui/
│   └── util/
│
├── examples/
│
├── go.mod
└── README.md
```

---

# REPL Design

REPL is the core entrypoint.

Supported commands:

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

Typing:

```text
/
```

must display all available commands.

---

# Interactive UX Rules

Commands must:

* ask parameters step-by-step
* provide default values
* provide selectable options
* preview execution plans
* ask for confirmation before execution

Example:

```text
Database name:
Charset:
Collation:
Confirm execution?
```

---

# Configuration Directory

All configuration must be stored under:

```text
~/.config/dbx/
```

Structure:

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

---

# Connection Configuration

Each connection has its own directory.

Example:

```text
~/.config/dbx/prod/config.json
```

Example config:

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

---

# Connection Modes

Supported in MVP:

```text
direct
ssh
```

Deferred:

```text
proxy
jump host
proxy chain
```

---

# Template System

Three template layers are supported.

Priority:

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

Example template:

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
      "sql": "CREATE DATABASE IF NOT EXISTS `{{database}}` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
    },
    {
      "type": "sql",
      "description": "Create user",
      "sql": "CREATE USER IF NOT EXISTS '{{database}}'@'%' IDENTIFIED BY '{{password}}'"
    },
    {
      "type": "sql",
      "description": "Grant privileges",
      "sql": "GRANT ALL PRIVILEGES ON `{{database}}`.* TO '{{database}}'@'%'"
    }
  ]
}
```

---

# Template Variables

Built-in variables:

```text
{{database}}
{{connection.name}}
{{connection.host}}
{{connection.user}}
```

User-provided variables:

```text
{{password}}
{{username}}
```

---

# SQL Safety Rules

Users must never provide raw unrestricted SQL.

Identifiers must match:

```text
[a-zA-Z_][a-zA-Z0-9_]*
```

Forbidden:

```text
drop database xxx; rm -rf /
```

All identifiers must be validated before rendering SQL.

---

# Driver Strategy

MVP only supports:

```text
mysql
```

Future expansion:

```text
postgres
sqlite
```

Do not over-engineer dialect abstractions initially.

---

# UI Rules

Do not use readline-like libraries.

Use standard library:

```go
bufio.Reader
fmt.Print
```

Implement lightweight UI helpers:

* Ask
* Choose
* Confirm
* AskPassword

---

# Session State

REPL must maintain session state.

```go
type Session struct {
    Connection *config.ConnectConfig
    DB         *sql.DB
}
```

---

# SSH Requirements

SSH must use:

```go
golang.org/x/crypto/ssh
```

Forbidden:

```go
exec.Command("ssh")
```

Preferred approach:

```go
mysql.RegisterDialContext(...)
```

Database connections should work through native SSH dialers.

---

# Coding Style

Requirements:

* small files
* small functions
* explicit error handling
* no panic
* no hidden side effects
* composition over abstraction-heavy designs

---

# Error Handling

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

---

# Logging

Do not introduce logging frameworks initially.

Allowed:

```go
fmt.Println
log.Printf
```

---

# MVP Scope

Must implement:

```text
REPL
connection management
SSH MySQL
/create database
/list databases
/drop database
template system
global templates
connection templates
```

Must NOT implement in MVP:

```text
ORM
migration system
complex query builders
AI SQL
schema diff
proxy chain
autocomplete
```

---

# Future Extensions

Future commands may include:

```text
/list tables
/create table
/table desc
/query
/template run
/history
/schema sync
```

Current architecture must support future expansion cleanly.

```
```
