package app

import "strings"

type CommandArgSource string

const (
	commandArgNone        CommandArgSource = ""
	commandArgConnection  CommandArgSource = "connection"
	commandArgDatabase    CommandArgSource = "database"
	commandArgTable       CommandArgSource = "table"
	commandArgUser        CommandArgSource = "user"
	commandArgTemplate    CommandArgSource = "template"
	commandArgTemplateTag CommandArgSource = "template-tag"
	commandArgStatic      CommandArgSource = "static"
	commandArgVariable    CommandArgSource = "variable"
)

type ArgSpec struct {
	Name     string
	Source   CommandArgSource
	Optional bool
	Values   []string
}

type CommandSpec struct {
	Path        string
	Description string
	Category    string
	Args        []ArgSpec
	Hidden      bool
}

func replCommandSpecs() []CommandSpec {
	return []CommandSpec{
		{Path: "/", Description: "show all commands", Category: "command"},
		{Path: "help", Description: "show command help", Category: "command"},
		{Path: "connect", Description: "connect to a saved connection", Category: "command", Args: []ArgSpec{{Name: "name", Source: commandArgConnection, Optional: true}}},
		{Path: "connections", Description: "list saved connections", Category: "command"},
		{Path: "audit log", Description: "show recent audit entries", Category: "command"},
		{Path: "connection create", Description: "create a saved connection", Category: "command"},
		{Path: "connection edit", Description: "edit a saved connection", Category: "command", Args: []ArgSpec{{Name: "name", Source: commandArgConnection}}},
		{Path: "connection delete", Description: "delete a saved connection", Category: "command", Args: []ArgSpec{{Name: "name", Source: commandArgConnection}}},
		{Path: "connection show", Description: "show a saved connection", Category: "command", Args: []ArgSpec{{Name: "name", Source: commandArgConnection}}},
		{Path: "connection test", Description: "test a saved connection", Category: "command", Args: []ArgSpec{
			{Name: "name", Source: commandArgConnection, Optional: true},
			{Name: "verbose", Source: commandArgStatic, Optional: true, Values: []string{"verbose"}},
		}},
		{Path: "connection doctor", Description: "inspect a saved connection statically", Category: "command", Args: []ArgSpec{{Name: "name", Source: commandArgConnection, Optional: true}}},
		{Path: "show databases", Description: "list databases on the active connection", Category: "command"},
		{Path: "create database", Description: "create a database", Category: "command"},
		{Path: "drop database", Description: "drop a database", Category: "command"},
		{Path: "use", Description: "select the current database", Category: "command", Args: []ArgSpec{{Name: "database", Source: commandArgDatabase}}},
		{Path: "create user", Description: "create a MySQL user", Category: "command", Args: []ArgSpec{{Name: "user", Source: commandArgNone, Optional: true}}},
		{Path: "show users", Description: "list MySQL users", Category: "command"},
		{Path: "drop user", Description: "drop a MySQL user", Category: "command", Args: []ArgSpec{{Name: "user", Source: commandArgUser, Optional: true}}},
		{Path: "show tables", Description: "list tables in current database", Category: "command"},
		{Path: "describe", Description: "describe a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable, Optional: true}}},
		{Path: "describe table", Description: "describe a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable, Optional: true}}},
		{Path: "show grants", Description: "show grants for a MySQL user", Category: "command", Args: []ArgSpec{
			{Name: "user", Source: commandArgUser, Optional: true},
			{Name: "host", Source: commandArgNone, Optional: true},
		}},
		{Path: "show indexes", Description: "show indexes for a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable, Optional: true}}},
		{Path: "show indexes on", Description: "show indexes for a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable}}, Hidden: true},
		{Path: "show processlist", Description: "show the active MySQL processlist", Category: "command"},
		{Path: "show variables", Description: "show MySQL system variables", Category: "command", Args: []ArgSpec{{Name: "pattern", Source: commandArgVariable, Optional: true}}},
		{Path: "show create table", Description: "show CREATE TABLE for a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable}}},
		{Path: "show table status", Description: "show compact table status", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable, Optional: true}}},
		{Path: "truncate table", Description: "delete all rows from a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable}}},
		{Path: "rename table", Description: "rename a table", Category: "command", Args: []ArgSpec{
			{Name: "from", Source: commandArgTable},
			{Name: "to", Source: commandArgNone, Optional: true},
		}},
		{Path: "show columns", Description: "show columns for a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable}}},
		{Path: "show foreign keys", Description: "show foreign keys for a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable}}},
		{Path: "show triggers", Description: "show triggers in current database", Category: "command"},
		{Path: "show views", Description: "show views in current database", Category: "command"},
		{Path: "count rows", Description: "count rows in a table", Category: "command", Args: []ArgSpec{{Name: "table", Source: commandArgTable, Optional: true}}},
		{Path: "peek rows", Description: "peek bounded rows from a table", Category: "command", Args: []ArgSpec{
			{Name: "table", Source: commandArgTable, Optional: true},
			{Name: "limit", Source: commandArgNone, Optional: true},
		}},
		{Path: "sample rows", Description: "sample bounded rows from a table", Category: "command", Args: []ArgSpec{
			{Name: "table", Source: commandArgTable, Optional: true},
			{Name: "limit", Source: commandArgNone, Optional: true},
		}},
		{Path: "show templates", Description: "list resolved workflow templates", Category: "command"},
		{Path: "show templates tag", Description: "filter templates by tag", Category: "command", Args: []ArgSpec{{Name: "tag", Source: commandArgTemplateTag}}, Hidden: true},
		{Path: "describe template", Description: "describe a workflow template", Category: "command", Args: []ArgSpec{{Name: "name", Source: commandArgTemplate}}},
		{Path: "template run", Description: "run a workflow template", Category: "command", Args: []ArgSpec{{Name: "name", Source: commandArgTemplate}}},
		{Path: "template validate", Description: "validate a workflow template", Category: "command", Args: []ArgSpec{{Name: "name", Source: commandArgTemplate}}},
		{Path: "status", Description: "show session status", Category: "command"},
		{Path: "context", Description: "show current REPL context", Category: "command"},
		{Path: "dry-run on", Description: "enable dry-run mode", Category: "command"},
		{Path: "dry-run off", Description: "disable dry-run mode", Category: "command"},
		{Path: "exit", Description: "exit the REPL", Category: "command"},

		{Path: "q", Description: "alias for exit", Category: "alias"},
		{Path: "quit", Description: "alias for exit", Category: "alias"},
		{Path: "conn", Description: "alias for connect", Category: "alias", Args: []ArgSpec{{Name: "name", Source: commandArgConnection, Optional: true}}},
		{Path: "cx", Description: "alias for connect", Category: "alias", Args: []ArgSpec{{Name: "name", Source: commandArgConnection, Optional: true}}},
		{Path: "conns", Description: "alias for connections", Category: "alias"},
		{Path: "show dbs", Description: "alias for show databases", Category: "alias"},
		{Path: "list databases", Description: "alias for show databases", Category: "alias"},
		{Path: "ls db", Description: "alias for show databases", Category: "alias"},
		{Path: "list users", Description: "alias for show users", Category: "alias"},
		{Path: "show user accounts", Description: "alias for show users", Category: "alias"},
		{Path: "columns", Description: "alias for show columns", Category: "alias", Args: []ArgSpec{{Name: "table", Source: commandArgTable}}},
		{Path: "desc table", Description: "alias for describe table", Category: "alias", Args: []ArgSpec{{Name: "table", Source: commandArgTable, Optional: true}}},
		{Path: "show fks", Description: "alias for show foreign keys", Category: "alias", Args: []ArgSpec{{Name: "table", Source: commandArgTable}}},
		{Path: "show index", Description: "alias for show indexes", Category: "alias", Args: []ArgSpec{{Name: "table", Source: commandArgTable, Optional: true}}},
		{Path: "show index on", Description: "alias for show indexes", Category: "alias", Args: []ArgSpec{{Name: "table", Source: commandArgTable}}, Hidden: true},
		{Path: "show processes", Description: "alias for show processlist", Category: "alias"},
		{Path: "show vars", Description: "alias for show variables", Category: "alias", Args: []ArgSpec{{Name: "pattern", Source: commandArgVariable, Optional: true}}},
		{Path: "show trigger", Description: "alias for show triggers", Category: "alias"},
		{Path: "show view", Description: "alias for show views", Category: "alias"},
		{Path: "count", Description: "alias for count rows", Category: "alias", Args: []ArgSpec{{Name: "table", Source: commandArgTable, Optional: true}}},
		{Path: "peek", Description: "alias for peek rows", Category: "alias", Args: []ArgSpec{
			{Name: "table", Source: commandArgTable, Optional: true},
			{Name: "limit", Source: commandArgNone, Optional: true},
		}},
		{Path: "sample", Description: "alias for sample rows", Category: "alias", Args: []ArgSpec{
			{Name: "table", Source: commandArgTable, Optional: true},
			{Name: "limit", Source: commandArgNone, Optional: true},
		}},
		{Path: "templates", Description: "alias for show templates", Category: "alias"},
		{Path: "templates tag", Description: "alias filter for show templates", Category: "alias", Args: []ArgSpec{{Name: "tag", Source: commandArgTemplateTag}}, Hidden: true},
		{Path: "template show", Description: "alias for describe template", Category: "alias", Args: []ArgSpec{{Name: "name", Source: commandArgTemplate}}},
		{Path: "template describe", Description: "alias for describe template", Category: "alias", Args: []ArgSpec{{Name: "name", Source: commandArgTemplate}}},
		{Path: "run template", Description: "alias for template run", Category: "alias", Args: []ArgSpec{{Name: "name", Source: commandArgTemplate}}},
		{Path: "ctx", Description: "alias for context", Category: "alias"},
		{Path: "dry on", Description: "alias for dry-run on", Category: "alias"},
		{Path: "dry off", Description: "alias for dry-run off", Category: "alias"},
		{Path: "create db", Description: "alias for create database", Category: "alias"},
		{Path: "drop db", Description: "alias for drop database", Category: "alias"},
		{Path: "test conn", Description: "alias for connection test", Category: "alias", Args: []ArgSpec{
			{Name: "name", Source: commandArgConnection, Optional: true},
			{Name: "verbose", Source: commandArgStatic, Optional: true, Values: []string{"verbose"}},
		}},
		{Path: "doctor conn", Description: "alias for connection doctor", Category: "alias", Args: []ArgSpec{{Name: "name", Source: commandArgConnection, Optional: true}}},
	}
}

func commandSpecByPath(path string) (CommandSpec, bool) {
	normalized := normalizeHelpTopic(path)
	for _, spec := range replCommandSpecs() {
		if normalizeHelpTopic(spec.Path) == normalized {
			return spec, true
		}
	}
	return CommandSpec{}, false
}

func helpCompletionTopics() []Suggestion {
	seen := map[string]struct{}{}
	suggestions := make([]Suggestion, 0)
	add := func(path string, description string) {
		path = normalizeHelpTopic(path)
		if path == "" {
			return
		}
		if _, exists := seen[path]; exists {
			return
		}
		seen[path] = struct{}{}
		suggestions = append(suggestions, Suggestion{
			Value:       path,
			Description: description,
			Category:    "topic",
		})
	}

	for _, spec := range replCommandSpecs() {
		if spec.Hidden {
			continue
		}
		add(spec.Path, spec.Description)
	}
	for topic := range helpEntries {
		if topic == "" {
			continue
		}
		description := ""
		if entry, ok := helpEntries[topic]; ok {
			description = entry.title + " help"
		}
		add(topic, description)
	}
	return suggestions
}

func isCommandPrefixRoot(req completionRequest) bool {
	return len(req.Fields) == 0 || (len(req.Fields) == 1 && !req.TrailingSpace)
}

func specTokens(path string) []string {
	return strings.Fields(path)
}
