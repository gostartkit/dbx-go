# Contributing

## Development

- Use `make fmt`, `make vet`, `make test`, and `make build` before submitting changes.
- Keep the project REPL-first and avoid expanding beyond the documented MVP without discussion.
- Prefer small files, small functions, explicit error handling, and standard library solutions first.

## Scope Guardrails

- Do not add raw-SQL user workflows.
- Do not add large CLI or prompt frameworks.
- Do not add ORM, migrations, proxy chains, autocomplete, or AI SQL features.

## Review Expectations

- Preserve the configuration layout under `~/.config/dbx/`.
- Avoid logging or printing secrets.
- Keep SSH access native through Go SSH libraries.
