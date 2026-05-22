package app

import (
	"fmt"
	"io"
	"strings"
	"sync"

	cmdpkg "pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/commandlang"
)

type helpEntry struct {
	title string
	body  string
}

type helpCommandOverride struct {
	summary string
	extra   string
}

type helpCommandCatalog struct {
	byPath map[string]cmdpkg.CommandSpec
}

var (
	helpCommandCatalogOnce sync.Once
	helpCommandCatalogData helpCommandCatalog
)

var helpEntries = map[string]helpEntry{
	"": {
		title: "dbx commands",
		body: strings.TrimSpace(`
Core commands:
  connect <name>       Connect to a saved connection
  use <name>           Select the current database
  doctor               Inspect the selected connection
  audit log            Show audit history
  exit                 Exit the shell

Examples:
  connect
  connect prod

  show
  show connections
  show tables
  show table users
  show templates --tag readonly

  create
  create connection prod --host 10.0.1.20 --user root
  create database appdb

  exec
  exec create_database_with_user --validate`),
	},
	"connection": {
		title: "connection commands",
		body: strings.TrimSpace(`
Connection commands use a single verb-first style.

Commands:
  show connections
  show connection <name>
  create connection <name>
  drop connection <name>`),
	},
}

var helpTopicAliases = map[string]string{
	"connection create": "create connection",
	"connection delete": "drop connection",
	"connection show":   "show connection",
	"connections":       "show connections",
	"context":           "show context",
}

var helpExtraBodies = map[string]string{
	"audit log": strings.TrimSpace(`
Show recent audit entries from:
  ~/.config/dbx/logs/audit.jsonl`),
}

var helpCommandOverrides = map[string]helpCommandOverride{
	"create connection": {
		summary: "Create a saved connection.",
		extra: strings.TrimSpace(`
This command writes:
  ~/.config/dbx/{name}/config.json`),
	},
	"create database": {
		summary: "Create a database from the resolved operation spec.",
	},
	"create user": {
		summary: "Create a MySQL user from the resolved operation spec.",
	},
	"doctor": {
		summary: "Inspect the selected connection statically without opening the network path.",
		extra: strings.TrimSpace(`
Checks:
  config structure
  password sources
  proxy URL shape
  SSH auth settings
  known_hosts presence`),
	},
	"drop database": {
		summary: "Drop a database from the resolved operation spec.",
	},
	"drop user": {
		summary: "Drop a MySQL user from the resolved operation spec.",
	},
	"exec": {
		summary: "Execute a named operation.",
	},
	"show table": {
		summary: "Show CREATE TABLE output for one table.",
	},
	"show templates": {
		summary: "Show resolved workflow templates.",
	},
}

func printHelpTopic(prompt printer, topic string) error {
	topic = canonicalHelpTopic(topic)

	if entry, ok := helpEntries[topic]; ok {
		printHelpEntry(prompt, entry)
		return nil
	}

	if spec, ok := commandHelpSpec(topic); ok {
		printHelpEntry(prompt, renderCommandHelp(topic, spec))
		return nil
	}

	if doc, ok := commandlang.DefaultRegistry().Help(topic); ok {
		printHelpEntry(prompt, helpEntry{title: doc.Title, body: doc.Body})
		return nil
	}

	if spec, ok := commandSpecByPath(topic); ok {
		printHelpEntry(prompt, helpEntry{title: spec.Path, body: spec.Description})
		return nil
	}

	return fmt.Errorf("unknown help topic %q; use help", topic)
}

func printHelpEntry(prompt printer, entry helpEntry) {
	if entry.title != "" {
		prompt.Println(entry.title)
	}
	if entry.body != "" {
		prompt.Println(entry.body)
	}
}

func helpLong(topic string) string {
	topic = canonicalHelpTopic(topic)
	if entry, ok := helpEntries[topic]; ok {
		return entry.body
	}
	if override, ok := helpCommandOverrides[topic]; ok {
		sections := make([]string, 0, 2)
		if summary := strings.TrimSpace(override.summary); summary != "" {
			sections = append(sections, summary)
		}
		if extra := strings.TrimSpace(override.extra); extra != "" {
			sections = append(sections, extra)
		}
		return strings.Join(sections, "\n\n")
	}
	return helpExtraBodies[topic]
}

func commandHelpSpec(topic string) (cmdpkg.CommandSpec, bool) {
	spec, ok := helpCommandSpecCatalog().byPath[canonicalHelpTopic(topic)]
	return spec, ok
}

func helpCommandSpecCatalog() *helpCommandCatalog {
	helpCommandCatalogOnce.Do(func() {
		app := (&cliBuilder{
			mode:    ModeREPL,
			out:     io.Discard,
			err:     io.Discard,
			globals: &cliGlobals{Format: "text"},
		}).buildApp()
		spec := app.SpecFor(cmdpkg.SurfaceREPL)

		byPath := make(map[string]cmdpkg.CommandSpec, len(spec.Commands)*2)
		for _, command := range spec.Commands {
			indexHelpCommandSpec(byPath, nil, command)
		}
		helpCommandCatalogData = helpCommandCatalog{byPath: byPath}
	})
	return &helpCommandCatalogData
}

func indexHelpCommandSpec(dst map[string]cmdpkg.CommandSpec, parent []string, spec cmdpkg.CommandSpec) {
	path := append(append([]string(nil), parent...), spec.Name)
	key := normalizeHelpTopic(strings.Join(path, " "))
	if key != "" {
		spec.Path = append([]string(nil), path...)
		dst[key] = spec
		for _, alias := range spec.Aliases {
			aliasKey := normalizeHelpTopic(strings.Join(append(append([]string(nil), parent...), alias), " "))
			if aliasKey != "" {
				dst[aliasKey] = spec
			}
		}
	}

	for _, sub := range spec.SubCommands {
		indexHelpCommandSpec(dst, path, sub)
	}
}

func renderCommandHelp(topic string, spec cmdpkg.CommandSpec) helpEntry {
	title := normalizeHelpTopic(strings.Join(spec.Path, " "))
	if title == "" {
		title = topic
	}

	sections := make([]string, 0, 6)
	override := helpCommandOverrides[title]
	summary := strings.TrimSpace(override.summary)
	if summary == "" {
		summary = strings.TrimSpace(spec.Short)
	}
	if summary != "" {
		sections = append(sections, summary)
	}
	extra := strings.TrimSpace(override.extra)
	if extra == "" {
		extra = strings.TrimSpace(helpExtraBodies[title])
	}
	if extra != "" {
		sections = append(sections, extra)
	}
	if usage := strings.TrimSpace(spec.UsageLine); usage != "" {
		sections = append(sections, "Usage:\n  "+usage)
	}

	aliases := visibleAliases(spec, topic)
	if len(aliases) > 0 {
		sections = append(sections, "Aliases:\n  "+strings.Join(aliases, "\n  "))
	}

	subcommands := visibleSubcommands(spec.SubCommands)
	if len(subcommands) > 0 {
		lines := make([]string, 0, len(subcommands))
		for _, sub := range subcommands {
			line := sub.Name
			if len(sub.Positionals) > 0 {
				line += " " + formatPositionals(sub.Positionals)
			}
			lines = append(lines, line)
		}
		sections = append(sections, "Subcommands:\n  "+strings.Join(lines, "\n  "))
	}

	flags := visibleFlags(spec.Flags)
	if len(flags) > 0 {
		lines := make([]string, 0, len(flags))
		for _, flag := range flags {
			line := "--" + flag.Name
			if flag.Type != "" && flag.Type != "bool" {
				line += " <" + flag.Type + ">"
			}
			if usage := strings.TrimSpace(flag.Usage); usage != "" {
				line += "  " + usage
			}
			lines = append(lines, line)
		}
		sections = append(sections, "Flags:\n  "+strings.Join(lines, "\n  "))
	}

	return helpEntry{
		title: title,
		body:  strings.Join(sections, "\n\n"),
	}
}

func visibleAliases(spec cmdpkg.CommandSpec, topic string) []string {
	aliases := make([]string, 0, len(spec.Aliases))
	for _, alias := range spec.Aliases {
		alias = normalizeHelpTopic(alias)
		if alias == "" || alias == topic {
			continue
		}
		aliases = append(aliases, alias)
	}
	return aliases
}

func visibleSubcommands(subcommands []cmdpkg.CommandSpec) []cmdpkg.CommandSpec {
	visible := make([]cmdpkg.CommandSpec, 0, len(subcommands))
	for _, sub := range subcommands {
		if sub.Hidden {
			continue
		}
		visible = append(visible, sub)
	}
	return visible
}

func visibleFlags(flags []cmdpkg.FlagSpec) []cmdpkg.FlagSpec {
	visible := make([]cmdpkg.FlagSpec, 0, len(flags))
	for _, flag := range flags {
		if flag.Hidden {
			continue
		}
		visible = append(visible, flag)
	}
	return visible
}

func formatPositionals(positionals []cmdpkg.PositionalSpec) string {
	parts := make([]string, 0, len(positionals))
	for _, positional := range positionals {
		name := positional.Name
		if positional.Variadic {
			name += "..."
		}
		if positional.Required {
			parts = append(parts, "<"+name+">")
			continue
		}
		parts = append(parts, "["+name+"]")
	}
	return strings.Join(parts, " ")
}

func normalizeHelpTopic(topic string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(topic)), " ")
}

func canonicalHelpTopic(topic string) string {
	topic = normalizeHelpTopic(topic)
	if alias, ok := helpTopicAliases[topic]; ok {
		return alias
	}
	return topic
}

type printer interface {
	Println(args ...any)
}

type writerPrinter struct {
	w io.Writer
}

func (p writerPrinter) Println(args ...any) {
	fmt.Fprintln(p.w, args...)
}
