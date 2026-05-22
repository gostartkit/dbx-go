package app

import "strings"

type printer interface {
	Println(args ...any)
}

var commandLongText = map[string]string{
	"": strings.TrimSpace(`
dbx is a lightweight MySQL shell with shared CLI and REPL commands.

Examples:
  connect prod
  show tables
  create database appdb
  exec create_database_with_user --validate`),
	"audit": strings.TrimSpace(`
Inspect local audit records stored under:
  ~/.config/dbx/logs/`),
	"audit log": strings.TrimSpace(`
Show recent audit entries from:
  ~/.config/dbx/logs/audit.jsonl`),
	"connection create": strings.TrimSpace(`
This command writes:
  ~/.config/dbx/{name}/config.json`),
	"connections": strings.TrimSpace(`
Show saved connections, including invalid configurations when present.`),
	"doctor": strings.TrimSpace(`
Checks:
  config structure
  password sources
  proxy URL shape
  SSH auth settings
  known_hosts presence`),
}

func commandLong(topic string) string {
	return commandLongText[normalizeHelpTopic(topic)]
}

func normalizeHelpTopic(topic string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(topic)), " ")
}
