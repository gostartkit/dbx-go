# dbx Architecture

[中文文档](ARCHITECTURE.zh-CN.md)

This document is a code-oriented walkthrough of the current `dbx` implementation. It focuses on how the repo is wired today, which packages own which responsibilities, and which internal behaviors matter when extending the tool.

## Overview

`dbx` is organized around one core idea: the REPL and the one-shot CLI should share the same command tree, the same validation rules, the same template resolution logic, and the same execution pipeline.

The implementation reflects that in a few important ways:

- process startup is thin
- command registration is centralized
- most user actions flow through `internal/app`
- template resolution happens before execution
- transport details are pushed down into the MySQL driver layer
- user state lives in `~/.config/dbx/`

## Runtime Flow

### 1. Startup and mode selection

- [`cmd/dbx/main.go`](../cmd/dbx/main.go) creates a signal-aware context and delegates to `app.NewCommandApp(...)`.
- [`internal/app/cli.go`](../internal/app/cli.go) constructs the root `cmd.App`, enables REPL mode, and registers the global flags and command tree.
- The active REPL loop is currently supplied by `pkg.gostartkit.com/cmd`, not by local code in `internal/repl/`.

### 2. Shared command tree

`internal/app` is the orchestration layer for both interactive and non-interactive usage.

- `cli.go` builds the shared command tree.
- `cli_*_commands.go` files register command groups and CLI flag surfaces.
- `commands.go`, `row_commands.go`, `user_commands.go`, and related files contain the REPL-facing handlers.
- `command_specs.go` derives help/usage metadata from the same command tree so help text, completion topics, and visible command paths stay aligned.

This is the main architectural seam in the repo: the user enters through different surfaces, but both surfaces converge on the same underlying application logic.

### 3. Application lifecycle and session state

[`internal/app/app.go`](../internal/app/app.go) owns the long-lived application state:

- prompt instance
- config store
- connector
- template service
- current session
- local command history
- dry-run mode
- completion caches

On startup, the application loads persisted history and session state. If `session.json` points to a previous connection, `dbx` offers to reconnect and can restore the selected database after validating that it still exists.

### 4. Configuration and persistence

[`internal/config/store.go`](../internal/config/store.go) owns the on-disk layout under `~/.config/dbx/`:

- connection configs
- connection-local template directories
- global templates
- `session.json`
- `history`
- `logs/audit.jsonl`

Important implementation details:

- history is persisted and trimmed to the latest `1000` entries
- audit logging is best-effort
- `audit log` reads the most recent `50` entries
- connection configs are validated on load and on save
- `SessionFile` still accepts legacy JSON field names for compatibility

[`internal/config/types.go`](../internal/config/types.go) applies defaults such as:

- driver `mysql`
- database port `3306`
- SSH port `22`
- connect timeout `10s`
- query timeout `30s`

[`internal/config/diagnostics.go`](../internal/config/diagnostics.go) performs strict mode-specific validation and produces structured diagnostics used by `doctor`.

### 5. Template and operation resolution

`dbx` does not let users submit unrestricted SQL. Instead, commands resolve a template or operation first.

Template responsibilities live in [`internal/template/`](../internal/template/):

- [`builtin.go`](../internal/template/builtin.go) defines builtin templates
- [`service.go`](../internal/template/service.go) loads and caches templates from connection, global, and builtin layers
- [`render.go`](../internal/template/render.go) performs variable rendering
- [`types.go`](../internal/template/types.go) validates schema, input types, and action shape

Resolution order is:

```text
connection template
> global template
> builtin template
```

[`internal/app/operation_runtime.go`](../internal/app/operation_runtime.go) exposes templates as named operations for `exec`. It merges two providers:

- builtin operations
- non-builtin resolved templates

That means `exec <name>` is not a second execution system; it is another entrypoint into the same template-backed execution machinery.

### 6. Command execution pipeline

The execution path is intentionally explicit:

1. resolve connection and optional database context
2. resolve a template or named operation
3. merge CLI/interactive input values with template defaults
4. validate typed inputs
5. build an execution plan and a redacted preview plan
6. confirm if the command is mutating
7. execute statements or return a dry-run result

Key files:

- [`internal/app/context_resolver.go`](../internal/app/context_resolver.go)
- [`internal/app/template_inputs.go`](../internal/app/template_inputs.go)
- [`internal/app/plan_support.go`](../internal/app/plan_support.go)
- [`internal/app/execution.go`](../internal/app/execution.go)

Execution plans carry:

- operation name
- template layer
- source
- transaction flag
- rendered SQL actions

If a plan is transactional, `execution.go` runs every action inside a SQL transaction and records whether it committed or rolled back.

### 7. Connectivity stack

The connectivity path is split cleanly:

- [`internal/connect/connect.go`](../internal/connect/connect.go): timeout-aware driver dispatch
- [`internal/driver/mysql.go`](../internal/driver/mysql.go): MySQL DSN/opening
- [`internal/driver/mysql_transport.go`](../internal/driver/mysql_transport.go): direct, SSH, proxy, and proxy-SSH dialers
- [`internal/driver/mysql_query.go`](../internal/driver/mysql_query.go): query helpers and result shaping

Important transport behaviors:

- SSH uses `golang.org/x/crypto/ssh`
- SOCKS5 uses `golang.org/x/net/proxy`
- SSH host keys are validated through `known_hosts`
- `DBX_KNOWN_HOSTS` can override the default known_hosts search path
- proxy URLs are redacted before user-facing output

`internal/driver/mysql_transport.go` registers custom MySQL dialers instead of shelling out to external tools, which keeps transport support native and testable.

### 8. REPL UX and completion

The interactive UX is intentionally lightweight.

- [`internal/ui/prompt.go`](../internal/ui/prompt.go) implements `Ask`, `Choose`, `Confirm`, and `AskPassword`.
- Password input uses `golang.org/x/term` when stdin is a terminal.
- History and prompt labeling are handled in `internal/app`.

Completion is more sophisticated than the prompt layer suggests:

- [`internal/app/completion*.go`](../internal/app/completion*.go) builds completions
- [`internal/commandlang/`](../internal/commandlang/) lexes and parses the current line
- completion providers combine syntax context, command metadata, and live resolver data

This means completion is syntax-aware rather than simple prefix matching. The `commandlang` package is used for parsing and completion context, not as a second command execution engine.

### 9. Diagnostics, audit, and output contracts

Three subsystems reinforce safety:

- [`internal/app/doctor.go`](../internal/app/doctor.go): static config and filesystem checks
- [`internal/app/audit.go`](../internal/app/audit.go): best-effort JSONL audit trail
- [`internal/util/error_codes.go`](../internal/util/error_codes.go): stable, sanitized JSON error envelopes

Notable behavior:

- `doctor` does not dial the database, proxy, or SSH target
- JSON errors are mapped to stable codes such as `VALIDATION_FAILED`, `SSH_AUTH_FAILED`, and `SQL_EXECUTION_FAILED`
- secrets are excluded from previews, logs, and JSON output

## Package Map

The active package layout today is:

- [`cmd/dbx/`](../cmd/dbx): process startup
- [`internal/app/`](../internal/app): command tree, REPL handlers, CLI handlers, execution orchestration, output shaping
- [`internal/commandlang/`](../internal/commandlang): lexical/syntax model for completion and help-aware parsing
- [`internal/config/`](../internal/config): config types, store, diagnostics, audit/history/session persistence
- [`internal/connect/`](../internal/connect): timeout-aware connector dispatch
- [`internal/driver/`](../internal/driver): MySQL transport and query helpers
- [`internal/template/`](../internal/template): builtin templates, layered resolution, rendering, validation
- [`internal/ui/`](../internal/ui): prompt and completion-facing UI types
- [`internal/ui/editor/`](../internal/ui/editor): buffer and completion edit primitives
- [`internal/util/`](../internal/util): validation helpers, layered errors, path helpers, JSON error codes

Two directories exist but are not part of the active execution path today:

- `internal/repl/` is currently empty
- `internal/commandmeta/` is currently empty

## Current Implementation Notes

These are the main takeaways from the current codebase shape:

- The shared command tree is the strongest architectural decision in the repo. It keeps REPL help, CLI help, validation, and completion synchronized.
- Template-driven execution is enforced consistently. Even built-in verbs like `create database` are routed through execution plans instead of hand-built SQL in the handler layer.
- Safety features are layered rather than centralized in one file: validation, redaction, confirmation, static diagnostics, and audit logging each live close to the behavior they protect.
- Some user-facing commands normalize into internal template command names. For example, `show rows <table>` resolves the `peek rows` template, and `show table <name>` resolves the `show create table` template. Template authors should follow the template command names, not always the literal user-facing verb.
- Completion is one of the more advanced parts of the codebase. The repo already contains a local lexer/parser model to keep suggestions context-aware without adopting a readline framework.

## Testing Shape

The repo has meaningful automated coverage across the main subsystems:

- `internal/app/*_test.go`: command surface, CLI/REPL parity, execution flows, completions, audit behavior
- `internal/template/*_test.go`: template loading, validation, rendering, precedence, performance
- `internal/config/*_test.go`: config defaults, store behavior, persistence rules
- `internal/driver/*_test.go`: proxy and transport behavior
- `internal/commandlang/*_test.go`: lexer/parser/schema behavior

That test shape is consistent with the project direction: keep the UX small, but defend the command surface and safety behavior heavily.
