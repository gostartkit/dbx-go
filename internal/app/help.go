package app

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"pkg.gostartkit.com/dbx/internal/commandmeta"
)

type helpEntry struct {
	title string
	body  string
}

type helpCommandOverride struct {
	summary string
	extra   string
}

type helpCommandSpec struct {
	path    []string
	command *commandmeta.Command
}

type helpCommandCatalog struct {
	byPath map[string]helpCommandSpec
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

func commandHelpSpec(topic string) (helpCommandSpec, bool) {
	spec, ok := helpCommandSpecCatalog().byPath[canonicalHelpTopic(topic)]
	return spec, ok
}

func helpCommandSpecCatalog() *helpCommandCatalog {
	helpCommandCatalogOnce.Do(func() {
		resolved := commandmeta.FlattenCommands(commandmeta.DefaultManifest())
		byPath := make(map[string]helpCommandSpec, len(resolved))
		for _, command := range resolved {
			key := normalizeHelpTopic(strings.Join(command.Path, " "))
			if key == "" || command.Command == nil {
				continue
			}
			byPath[key] = helpCommandSpec{
				path:    append([]string(nil), command.CanonicalPath...),
				command: command.Command,
			}
		}
		helpCommandCatalogData = helpCommandCatalog{byPath: byPath}
	})
	return &helpCommandCatalogData
}

func renderCommandHelp(topic string, spec helpCommandSpec) helpEntry {
	title := normalizeHelpTopic(strings.Join(spec.path, " "))
	if title == "" {
		title = topic
	}

	sections := make([]string, 0, 6)
	override := helpCommandOverrides[title]
	summary := strings.TrimSpace(override.summary)
	if summary == "" {
		summary = strings.TrimSpace(spec.command.Description)
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
	if usage := strings.TrimSpace(spec.command.UsageLine); usage != "" {
		sections = append(sections, "Usage:\n  "+usage)
	}

	aliases := visibleAliases(spec, topic)
	if len(aliases) > 0 {
		sections = append(sections, "Aliases:\n  "+strings.Join(aliases, "\n  "))
	}

	subcommands := visibleSubcommands(spec.command.Subcommands)
	if len(subcommands) > 0 {
		lines := make([]string, 0, len(subcommands))
		for _, sub := range subcommands {
			line := sub.Name
			if desc := strings.TrimSpace(sub.Description); desc != "" {
				line += "  " + desc
			}
			lines = append(lines, line)
		}
		sections = append(sections, "Subcommands:\n  "+strings.Join(lines, "\n  "))
	}

	flags := visibleFlags(spec.command.Flags)
	if len(flags) > 0 {
		lines := make([]string, 0, len(flags))
		for _, flag := range flags {
			line := flag.Name
			if flag.ValueType != commandmeta.ValueBool {
				line += " <value>"
			}
			if desc := strings.TrimSpace(flag.Description); desc != "" {
				line += "  " + desc
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

func visibleAliases(spec helpCommandSpec, topic string) []string {
	aliases := make([]string, 0, len(spec.command.Aliases))
	for _, alias := range spec.command.Aliases {
		alias = normalizeHelpTopic(alias)
		if alias == "" || alias == topic {
			continue
		}
		aliases = append(aliases, alias)
	}
	return aliases
}

func visibleSubcommands(subcommands []*commandmeta.Command) []*commandmeta.Command {
	visible := make([]*commandmeta.Command, 0, len(subcommands))
	for _, sub := range subcommands {
		if sub == nil || sub.Hidden {
			continue
		}
		visible = append(visible, sub)
	}
	return visible
}

func visibleFlags(flags []*commandmeta.Flag) []*commandmeta.Flag {
	visible := make([]*commandmeta.Flag, 0, len(flags))
	for _, flag := range flags {
		if flag == nil {
			continue
		}
		visible = append(visible, flag)
	}
	return visible
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
