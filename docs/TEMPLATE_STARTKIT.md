# dbx Template System Startkit

[English README](../README.md) | [中文 README](../README.zh-CN.md) | [中文教程](TEMPLATE_STARTKIT.zh-CN.md)

This guide is for two groups of readers:

- People writing their first `dbx` template
- People who already know the JSON format and want to build templates that are safer, clearer, and easier to maintain

The goal is not just to list fields. The goal is to give you a practical workflow for building templates that work well in both the REPL and CLI.

By the end, you should understand:

1. How `dbx` finds and runs templates
2. When to use a global template vs. a connection-scoped template
3. How to handle secrets, grants, environment variables, and dry-run safely
4. How to design templates that are easy to preview, validate, and maintain

## 1. The Right Mental Model

In `dbx`, a template is not a scripting engine. It is a constrained operational workflow.

A template defines:

- metadata: name, category, tags, description
- a match rule: which command and driver it applies to
- typed inputs: string, secret, select, confirm, identifier, int
- one or more SQL actions that `dbx` renders and executes

The core product idea is:

- users should not provide unrestricted SQL
- users provide parameters and choices
- `dbx` validates, renders, previews, redacts, and executes

This makes templates a good fit for:

- creating databases
- dropping databases
- creating users and grants
- environment-specific operational workflows
- read-only inspection workflows with consistent behavior

Templates are not a good fit for:

- loops
- conditionals
- remote template registries
- arbitrary shelling or scripting
- plugin-style execution

Today, templates support exactly one action type: `sql`.

## 2. Where Templates Live

Templates resolve in this priority order:

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

How to think about the layers:

- global templates: reusable defaults for all connections
- connection templates: only apply to a specific saved connection such as `prod`
- builtin templates: code-defined fallbacks bundled with `dbx`

Resolution always prefers the highest matching layer first:

- if a connection-scoped template matches, lower layers are ignored
- if the connection layer has no match, `dbx` falls back to global
- if global has no match, `dbx` falls back to builtin

The same priority applies to same-name resolved listings. If both global and connection scope contain `shared_workflow`, the resolved template shown for that connection will be the connection-scoped one.

## 3. The Smallest Useful Template

Start with the simplest working form:

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
      "description": "Drop database `{{database}}`",
      "sql": "DROP DATABASE IF EXISTS `{{database}}`"
    }
  ]
}
```

This is valid because it has:

- a `name`
- a `match.command`
- at least one `action`
- only `sql` actions

Save it to:

```text
~/.config/dbx/templates/drop_database_guarded.json
```

Then run:

```bash
dbx --connection prod drop database app_demo --template drop_database_guarded --dry-run
```

What happens:

1. `dbx` resolves the template
2. `database=app_demo` becomes a template variable
3. the SQL is rendered
4. the preview / dry-run result is printed

## 4. Full Template Shape

A fuller template looks like this:

```json
{
  "version": 1,
  "name": "create_database_with_user",
  "category": "database",
  "tags": ["grant", "tenant"],
  "description": "Create a database, create a same-name MySQL user, and grant privileges.",
  "transaction": true,
  "match": {
    "command": "create database",
    "driver": "mysql"
  },
  "inputs": [
    {
      "name": "database",
      "type": "string",
      "prompt": "Database name"
    }
  ],
  "actions": [
    {
      "type": "sql",
      "description": "Create database `{{database}}`",
      "sql": "CREATE DATABASE IF NOT EXISTS `{{database}}`"
    }
  ]
}
```

The fields below are the ones you need to understand well.

### 4.1 `version`

- current schema version: `1`
- omitted values default to `1`
- any other version fails validation

Recommendation: always write `1` explicitly in new templates.

### 4.2 `name`

- required
- used by `exec <name>`
- shown in `show templates`
- used in ambiguity resolution and human-facing output

Good names describe the workflow, not the environment alone.

Recommended examples:

```text
create_database_with_user
readonly_user
drop_database_guarded
prod_app_database
```

### 4.3 `category`

- optional
- defaults to `custom`

Useful for:

- `show templates` organization
- human scanning
- team conventions

Common values:

```text
database
user
inspection
grant
custom
```

### 4.4 `tags`

- optional
- useful for filtering and discovery
- used by `show templates --tag ...`

Example:

```json
"tags": ["tenant", "grant", "readonly"]
```

### 4.5 `description`

- optional
- strongly recommended
- shown in listings and ambiguity messages

If multiple templates match the same command at the same layer, a clear description makes CLI errors much easier to understand.

### 4.6 `transaction`

- optional
- `true` asks `dbx` to run the workflow in a transaction

Good candidates:

- multi-step grant workflows
- operations that should succeed or fail as a unit

Important caveat:

- this is a `dbx` execution setting
- MySQL DDL semantics still behave according to MySQL rules

Use it when you mean it, not by default.

### 4.7 `match`

Example:

```json
"match": {
  "command": "create database",
  "driver": "mysql"
}
```

Rules:

- `command` is required
- `driver` is optional, but the current project only supports `mysql`

This is the routing rule that determines whether a template is considered for automatic resolution.

## 5. Writing the Right `match.command`

This is one of the easiest places to make mistakes.

Not every user-visible command maps directly to the same template command string. The current code uses these hooks:

| User command | Template `match.command` |
| --- | --- |
| `create database` | `create database` |
| `show databases` | `show databases` |
| `drop database` | `drop database` |
| `create user` | `create user` |
| `show users` | `show users` |
| `drop user` | `drop user` |
| `show tables` | `show tables` |
| `show columns <table>` | `show columns` |
| `show table <name>` | `show create table` |
| `show rows <table>` | `peek rows` |

The two most important surprises:

- to override `show table`, match `show create table`
- to override `show rows`, match `peek rows`

So this would not automatically take over `show rows <table>` today:

```json
"match": { "command": "show rows", "driver": "mysql" }
```

## 6. Designing Inputs

Inputs are defined under `inputs`:

```json
{
  "name": "password",
  "type": "secret",
  "prompt": "New user password"
}
```

Current input types:

- `string`
- `secret`
- `select`
- `confirm`
- `identifier`
- `int`

### 6.1 Common Input Fields

An input can use:

- `name`
- `type`
- `prompt`
- `description`
- `required`
- `secret`
- `default`
- `options`
- `choices`
- `identifier`

Behavior to remember:

- if `required` is omitted and there is no `default`, the input is effectively required
- if `prompt` is omitted, `dbx` falls back to `description`, then to `name`
- `options` and `choices` both work for `select`
- `secret: true` implicitly behaves like `type: "secret"`
- `identifier: true` implicitly behaves like `type: "identifier"`

### 6.2 `string`

Example:

```json
{
  "name": "user_host",
  "type": "string",
  "prompt": "User host",
  "default": "%"
}
```

Good for:

- host values
- comments
- labels
- free-form but non-secret operational values

### 6.3 `secret`

Example:

```json
{
  "name": "password",
  "type": "secret",
  "prompt": "Password"
}
```

Behavior:

- hidden input in the REPL
- redacted as `***` in preview / verbose / JSON output
- not logged in cleartext in audit output

Best practice:

- avoid `--input password=super-secret`
- prefer `--input password-env=APP_PASSWORD`

That avoids leaking secrets into shell history.

### 6.4 `select`

Example:

```json
{
  "name": "charset",
  "type": "select",
  "prompt": "Charset",
  "default": "utf8mb4",
  "options": ["utf8mb4", "utf8"]
}
```

Behavior:

- REPL users get constrained choices
- CLI users get validation if they pass a value outside the allowed set

Great for:

- charset
- collation
- privilege mode
- environment or tenant mode

### 6.5 `confirm`

Example:

```json
{
  "name": "really_drop",
  "type": "confirm",
  "prompt": "Drop this database?",
  "default": false
}
```

It renders to string values:

- `true`
- `false`

Useful for additional operator confirmation, but remember that the template system itself does not provide branching. It is better for expressing intent and inputs than for implementing conditional execution.

### 6.6 `identifier`

Example:

```json
{
  "name": "role_name",
  "type": "identifier",
  "prompt": "Role name"
}
```

Current validation rule:

```text
[a-zA-Z_][a-zA-Z0-9_]*
```

This is stricter than the current database-name and MySQL-username rules.

Practical advice:

- if the value needs to allow `-`, do not use `identifier`
- for example, database names currently allow values like `greenhn-prod`
- `identifier` would reject that

So in practice:

- use `identifier` for strict internal symbols
- do not blindly use it for database names or usernames

### 6.7 `int`

Example:

```json
{
  "name": "limit",
  "type": "int",
  "prompt": "Limit",
  "default": 20
}
```

REPL mode collects it as an integer and CLI mode validates it as an integer.

## 7. Writing Actions

Actions are the executable part of a template:

```json
{
  "type": "sql",
  "description": "Grant SELECT on `{{database}}`.*",
  "sql": "GRANT SELECT ON `{{database}}`.* TO '{{username}}'@'{{user_host}}'"
}
```

Current rules:

- `type` must be `sql`
- `sql` must not be empty
- `description` is shown in previews and results

Recommendations:

- write descriptions for humans
- write SQL for the database
- split a 3-step workflow into 3 actions instead of stuffing everything into one long statement

## 8. How Rendering Works

Templates use Mustache-style placeholders:

```text
{{database}}
{{username}}
{{connection.name}}
{{connection.host}}
```

Built-in connection fields include:

- `{{connection.name}}`
- `{{connection.driver}}`
- `{{connection.mode}}`
- `{{connection.host}}`
- `{{connection.port}}`
- `{{connection.user}}`

### 8.1 SQL Strings Are Escaped

When `dbx` renders SQL, normal string values are escaped for MySQL string contexts.

If the input is:

- `pa'ss`

the rendered SQL becomes:

- `pa''ss`

That is why template SQL should typically look like:

```sql
IDENTIFIED BY '{{password}}'
```

instead of manually double-escaping input yourself.

### 8.2 Descriptions Use Raw Values, SQL Uses Escaped Values

This is useful and intentional.

For example:

- description: `Create user {{username}}`
- SQL: `CREATE USER '{{username}}'`

The operator sees a natural description, while the executed SQL remains safe.

### 8.3 `_sql` Variables Are Inserted Raw

This is an advanced but important rule.

If a variable name ends with `_sql`, the current implementation inserts it into SQL without escaping.

Example:

- `grant_sql`

That is appropriate for program-generated SQL fragments such as:

```text
GRANT SELECT ON `app_prod`.* TO 'ro'@'%'
```

Do not put raw user input into `_sql` variables.

Safe rule:

- only use `_sql` for trusted SQL fragments generated by the application logic
- never use it as a shortcut for bypassing normal input safety

## 9. Feeding Inputs from the CLI

There are two main patterns.

### 9.1 Pass the Value Directly

```bash
dbx --connection prod exec create_database_with_user \
  --input database=app_demo \
  --input user_host=% \
  --input password=super-secret \
  --preview
```

### 9.2 Resolve the Value from an Environment Variable

Preferred for secrets:

```bash
export APP_PASSWORD='super-secret'

dbx --connection prod exec create_database_with_user \
  --input database=app_demo \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --preview
```

Rule:

- if a key ends with `-env`
- `dbx` removes that suffix
- the value is treated as an environment variable name

So:

- `password-env=APP_PASSWORD`

becomes template input:

- `password`

This works well for any secret-style template input.

## 10. REPL vs. CLI Input Behavior

In the REPL, missing template inputs can be collected interactively:

- `string`: prompt
- `secret`: hidden prompt
- `select`: constrained choice
- `confirm`: yes/no
- `int`: integer prompt

In the CLI:

- there is no interactive fallback
- required inputs must be present
- defaults are applied automatically when defined

Practical workflow:

- REPL is good for exploration and guided operations
- CLI is better for scripts, explicit parameterization, and repeatability

## 11. The 4 Most Useful Template Commands

### 11.1 List Templates

```bash
dbx --connection prod show templates
dbx --connection prod show templates database
dbx --connection prod show templates --tag tenant
```

Typical output shape:

```text
Templates:
name                       scope       category   command
create_database_with_user  global      database   create database  [grant,tenant]
prod_app_database          connection  custom     create database
```

### 11.2 Validate a Template

```bash
dbx --connection prod exec create_database_with_user --validate
```

This checks:

- template structure
- input type validity
- action type validity

### 11.3 Preview Without Executing

```bash
dbx --connection prod exec create_database_with_user \
  --input database=app_demo \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --preview \
  --verbose
```

Use this to:

- inspect the action list
- inspect redacted SQL
- confirm that all inputs were resolved correctly

### 11.4 Execute Through a Business Command

If a command supports `--template`, you can attach a named template directly:

```bash
dbx --connection prod create database app_demo \
  --template create_database_with_user \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --yes
```

Current major commands with explicit `--template` support:

- `show databases`
- `create database`
- `drop database`
- `create user`
- `drop user`

Current commands without explicit `--template` support:

- `show tables`
- `show columns`
- `show table`
- `show rows`
- `show users`

For those commands, you currently need one of these strategies:

- keep only one automatic match at the selected layer
- or run the named workflow directly with `exec <name>`

## 12. Practical Example 1: Start from the Repository Example

The repository includes:

- [examples/templates/create_database_with_user.json](../examples/templates/create_database_with_user.json)

It does 3 things:

1. creates a database
2. creates a same-name MySQL user
3. grants privileges on that database

Install it into the global template directory:

```bash
mkdir -p ~/.config/dbx/templates
cp examples/templates/create_database_with_user.json ~/.config/dbx/templates/
```

Validate first:

```bash
dbx --connection prod exec create_database_with_user --validate
```

Then preview:

```bash
export APP_PASSWORD='super-secret'

dbx --connection prod exec create_database_with_user \
  --input database=app_demo \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --preview \
  --verbose
```

Then execute through the business command:

```bash
dbx --connection prod create database app_demo \
  --template create_database_with_user \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --yes
```

This is the most reliable Startkit loop:

1. `show templates`
2. `exec --validate`
3. `exec --preview --verbose`
4. attach to the business command and execute

## 13. Practical Example 2: A Connection-Scoped Production Drop Template

Goal:

- production database drops should use a more explicit workflow
- the behavior should apply only to `prod`
- other connections should remain unchanged

Create the directory:

```bash
mkdir -p ~/.config/dbx/prod/templates
```

Create the file:

```json
{
  "version": 1,
  "name": "drop_database_guarded",
  "category": "database",
  "tags": ["danger", "prod"],
  "description": "Production database drop workflow with explicit labeling.",
  "match": {
    "command": "drop database",
    "driver": "mysql"
  },
  "actions": [
    {
      "type": "sql",
      "description": "Drop production database `{{database}}` on {{connection.name}}",
      "sql": "DROP DATABASE IF EXISTS `{{database}}`"
    }
  ]
}
```

Save it to:

```text
~/.config/dbx/prod/templates/drop_database_guarded.json
```

Then:

```bash
dbx --connection prod show templates --tag prod
dbx --connection prod drop database app_demo --template drop_database_guarded --dry-run
```

Why this is useful:

- it only affects `prod`
- it overrides global and builtin behavior for that connection
- it leaves other environments alone

## 14. Practical Example 3: A Readonly User Workflow

This example is a good fit for tenant-level readonly accounts:

```json
{
  "version": 1,
  "name": "readonly_user",
  "category": "user",
  "tags": ["readonly", "grant"],
  "description": "Create a readonly MySQL user for the selected database.",
  "transaction": true,
  "match": {
    "command": "create user",
    "driver": "mysql"
  },
  "inputs": [
    {
      "name": "username",
      "type": "string",
      "prompt": "Username"
    },
    {
      "name": "user_host",
      "type": "string",
      "prompt": "User host",
      "default": "%"
    },
    {
      "name": "password",
      "type": "secret",
      "prompt": "Password"
    }
  ],
  "actions": [
    {
      "type": "sql",
      "description": "Create MySQL user '{{username}}'@'{{user_host}}'",
      "sql": "CREATE USER '{{username}}'@'{{user_host}}' IDENTIFIED BY '{{password}}'"
    },
    {
      "type": "sql",
      "description": "Grant SELECT on `{{database}}`.* to '{{username}}'@'{{user_host}}'",
      "sql": "GRANT SELECT ON `{{database}}`.* TO '{{username}}'@'{{user_host}}'"
    }
  ]
}
```

Recommended preview flow:

```bash
export RO_PASSWORD='replace-me'

dbx --connection prod --database app_prod exec readonly_user \
  --input username=analytics_ro \
  --input user_host=% \
  --input password-env=RO_PASSWORD \
  --preview \
  --verbose
```

Confirm the rendered actions first, then execute.

## 15. When to Choose Global vs. Connection Scope

Use a global template when:

- the logic should be reused across environments
- differences are mostly input values, not behavior
- you want a team-wide default workflow

Use a connection-scoped template when:

- the behavior should only apply to one connection such as `prod`
- production and development need different execution semantics
- you want to override a global default
- you need stronger environment-specific messaging or protections

Short rule:

- shared policy -> global
- environment-specific policy -> connection scope

## 16. How Ambiguity Happens

If multiple templates at the same layer match the same command, for example two global templates that both match `create database`, then:

- the REPL asks you to choose
- the CLI fails with an ambiguity error

The CLI error tells you to choose explicitly with:

- `exec <name>`
- or `--template <name>` when the command supports it

Recommended team convention:

- keep one automatic match per command per layer by default
- if you need multiple variants, treat them as named workflows and run them primarily with `exec <name>`

## 17. Redaction and Safety Boundaries

The template system already helps with:

- redacting `secret` inputs in preview / JSON / verbose SQL
- keeping secret values out of audit logs
- escaping normal string values for MySQL string rendering

But you still need to keep these boundaries:

- do not hardcode secrets inside template JSON
- do not put raw user input into `_sql` variables
- prefer `*-env` inputs for secrets in CLI usage
- do not put production-only workflows into global scope unless that is truly intended

## 18. The 7 Most Important Footguns

### Footgun 1: `identifier` is stricter than database-name rules

If you need values like `greenhn-prod`, do not use `identifier`.

### Footgun 2: `show table` does not match `show table`

Use:

```json
"command": "show create table"
```

### Footgun 3: `show rows` does not match `show rows`

Use:

```json
"command": "peek rows"
```

### Footgun 4: Not every command supports `--template`

Check the current command implementation instead of assuming all command families can pick named templates explicitly.

### Footgun 5: The CLI will not prompt to fill missing required inputs

Scripts must provide the required values up front.

### Footgun 6: `secret` redacts output, but shell history does not

So in production, prefer:

```text
--input password-env=ENV_NAME
```

### Footgun 7: Templates are not a scripting language

Do not try to add:

- branching
- loops
- remote includes
- shell execution

If you need those things, the logic probably belongs in application code rather than in the template layer.

## 19. A Recommended Template Authoring Workflow

Whenever you create a new template, this sequence works well:

1. choose the correct `match.command`
2. choose the correct layer: global or connection scope
3. start with a minimal template and one action
4. run `show templates` to make sure it resolves
5. run `exec <name> --validate`
6. run `exec <name> --preview --verbose`
7. only then add more actions, tags, descriptions, or transaction settings
8. finally attach it to the business command and execute

## 20. A Practical Startkit Checklist

Before shipping a template, check:

- this workflow actually belongs in the template system
- `match.command` matches the real current hook used by the implementation
- the layer choice is intentional
- the `name` is clear
- the `description` is useful in ambiguity messages
- all sensitive inputs use `secret`
- values that need hyphens are not forced through `identifier`
- enum-like fields use `select`
- `_sql` variables only hold trusted SQL fragments
- you validated first
- you previewed before executing
- production execution paths use `--yes` intentionally

## 21. Closing Thought

If you only remember one sentence, make it this:

The best use of the `dbx` template system is not “parameterized SQL”, but “validated, previewable, overridable, redacted operational workflows”.

When you design a template, think in this order:

1. which command hook does it belong to
2. which layer should own it
3. which inputs should be `secret`, `select`, or `int`
4. whether ambiguity is acceptable
5. whether the workflow is safe to preview before execution

That mindset usually produces templates that are much safer and easier to maintain than ad hoc SQL snippets.
