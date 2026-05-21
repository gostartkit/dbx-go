# Contributing

[中文版本](CONTRIBUTING.zh-CN.md)

## Development

- Use `make check` before submitting changes. It verifies formatting, `go vet`, tests, and a full build.
- If you are iterating locally, the common loop is `make fmt`, `make test`, and `make build`.
- Keep the project REPL-first and avoid expanding beyond the documented MVP without discussion.
- Prefer small files, small functions, explicit error handling, and standard library solutions first.

## Scope Guardrails

- Keep guided operations as the primary UX. Direct SQL should remain an explicit escape hatch, not the default experience.
- Do not add unrestricted SQL entrypoints or reintroduce `run sql`-style workflows without discussion.
- Do not add large CLI or prompt frameworks.
- Do not add ORM, migrations, proxy chains, autocomplete, or AI SQL features.

## Transport And Dependencies

- Keep SSH support native through `golang.org/x/crypto/ssh`; do not shell out to `ssh`.
- Keep proxy support limited to SOCKS5 through `golang.org/x/net/proxy`.
- Prefer the standard library unless a dependency is already part of the allowed project set.
- Avoid introducing frameworks such as Cobra, Viper, PromptUI, Survey, readline, GORM, or tablewriter.

## Review Expectations

- Preserve the configuration layout under `~/.config/dbx/`.
- Avoid logging or printing secrets.
- Keep SSH access native through Go SSH libraries.
- Keep embedded help text, command examples, and Markdown docs in sync when command behavior changes.
- Update `README.md`, `README.zh-CN.md`, `CONTRIBUTING.md`, `CONTRIBUTING.zh-CN.md`, and `AGENTS.md` together when user-facing command surfaces or workflows change.
