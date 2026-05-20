package app

import (
	"slices"
	"strings"

	"pkg.gostartkit.com/dbx/internal/ui"
)

type CompletionContext struct {
	Connection  string
	Database    string
	DryRun      bool
	Connections []CompletionConnection
	Databases   []string
	Tables      []string
	Users       []string
}

type CompletionConnection struct {
	Name   string
	Driver string
	Mode   string
}

type Suggestion struct {
	Value       string
	Description string
	Category    string
}

type CompletionProvider interface {
	Complete(ctx CompletionContext, req completionRequest) []Suggestion
}

type completionRequest struct {
	Line          string
	Fields        []string
	TrailingSpace bool
	Prefix        string
	ReplaceFrom   int
	ReplaceTo     int
}

type completionProviderFunc func(ctx CompletionContext, req completionRequest) []Suggestion

func (f completionProviderFunc) Complete(ctx CompletionContext, req completionRequest) []Suggestion {
	return f(ctx, req)
}

type completionEngine struct {
	providers []CompletionProvider
}

var rootSuggestions = []Suggestion{
	{Value: "connect", Description: "connect to a saved connection", Category: "command"},
	{Value: "connections", Description: "list saved connections", Category: "command"},
	{Value: "audit log", Description: "show recent audit entries", Category: "command"},
	{Value: "connection create", Description: "create a saved connection", Category: "command"},
	{Value: "connection edit", Description: "edit a saved connection", Category: "command"},
	{Value: "connection delete", Description: "delete a saved connection", Category: "command"},
	{Value: "connection show", Description: "show a saved connection", Category: "command"},
	{Value: "connection test", Description: "test a saved connection", Category: "command"},
	{Value: "connection doctor", Description: "inspect a saved connection statically", Category: "command"},
	{Value: "count", Description: "alias for count rows", Category: "alias"},
	{Value: "count rows", Description: "count rows in a table", Category: "command"},
	{Value: "context", Description: "show current REPL context", Category: "command"},
	{Value: "create database", Description: "create a database", Category: "command"},
	{Value: "create user", Description: "create a MySQL user", Category: "command"},
	{Value: "columns", Description: "alias for show columns", Category: "alias"},
	{Value: "show databases", Description: "list databases on the active connection", Category: "command"},
	{Value: "show dbs", Description: "alias for show databases", Category: "alias"},
	{Value: "list databases", Description: "alias for show databases", Category: "alias"},
	{Value: "show columns", Description: "show columns for a table", Category: "command"},
	{Value: "show tables", Description: "list tables in current database", Category: "command"},
	{Value: "show foreign keys", Description: "show foreign keys for a table", Category: "command"},
	{Value: "show fks", Description: "alias for show foreign keys", Category: "alias"},
	{Value: "show indexes", Description: "show indexes for a table", Category: "command"},
	{Value: "show create table", Description: "show CREATE TABLE for a table", Category: "command"},
	{Value: "show table status", Description: "show compact table status", Category: "command"},
	{Value: "show users", Description: "list MySQL users", Category: "command"},
	{Value: "show grants", Description: "show grants for a MySQL user", Category: "command"},
	{Value: "show processlist", Description: "show the active MySQL processlist", Category: "command"},
	{Value: "show triggers", Description: "show triggers in current database", Category: "command"},
	{Value: "show trigger", Description: "alias for show triggers", Category: "alias"},
	{Value: "show variables", Description: "show MySQL system variables", Category: "command"},
	{Value: "show views", Description: "show views in current database", Category: "command"},
	{Value: "show view", Description: "alias for show views", Category: "alias"},
	{Value: "drop database", Description: "drop a database", Category: "command"},
	{Value: "drop user", Description: "drop a MySQL user", Category: "command"},
	{Value: "peek", Description: "alias for peek rows", Category: "alias"},
	{Value: "peek rows", Description: "peek bounded rows from a table", Category: "command"},
	{Value: "sample", Description: "alias for sample rows", Category: "alias"},
	{Value: "sample rows", Description: "sample bounded rows from a table", Category: "command"},
	{Value: "truncate table", Description: "delete all rows from a table", Category: "command"},
	{Value: "rename table", Description: "rename a table", Category: "command"},
	{Value: "describe", Description: "describe a table", Category: "command"},
	{Value: "use", Description: "select the current database", Category: "command"},
	{Value: "status", Description: "show session status", Category: "command"},
	{Value: "dry-run on", Description: "enable dry-run mode", Category: "command"},
	{Value: "dry-run off", Description: "disable dry-run mode", Category: "command"},
	{Value: "help", Description: "show command help", Category: "command"},
	{Value: "exit", Description: "exit the REPL", Category: "command"},
	{Value: "ctx", Description: "alias for context", Category: "alias"},
	{Value: "desc table", Description: "alias for describe table", Category: "alias"},
	{Value: "show index", Description: "alias for show indexes", Category: "alias"},
	{Value: "show processes", Description: "alias for show processlist", Category: "alias"},
	{Value: "show vars", Description: "alias for show variables", Category: "alias"},
	{Value: "list users", Description: "alias for show users", Category: "alias"},
	{Value: "show user accounts", Description: "alias for show users", Category: "alias"},
}

func calculateCompletion(line string, ctx CompletionContext) ui.Completion {
	req := parseCompletionRequest(line)
	engine := newCompletionEngine()
	suggestions := engine.Complete(ctx, req)
	return toUICompletion(req, suggestions)
}

func newCompletionEngine() completionEngine {
	return completionEngine{
		providers: []CompletionProvider{
			completionProviderFunc(completeConnectionCommands),
			completionProviderFunc(completeConnectNames),
			completionProviderFunc(completeUseDatabases),
			completionProviderFunc(completeCountPeekSampleTables),
			completionProviderFunc(completeShowGrantsUsers),
			completionProviderFunc(completeShowColumnsTables),
			completionProviderFunc(completeShowForeignKeysTables),
			completionProviderFunc(completeShowIndexesTables),
			completionProviderFunc(completeShowCreateTables),
			completionProviderFunc(completeShowTableStatusTables),
			completionProviderFunc(completeShowVariablesPatterns),
			completionProviderFunc(completeDropUsers),
			completionProviderFunc(completeRenameTables),
			completionProviderFunc(completeTruncateTables),
			completionProviderFunc(completeDescribeTables),
			completionProviderFunc(completeShowTables),
			completionProviderFunc(completeCreateDropShowListHelpAuditTree),
			completionProviderFunc(completeRootCommands),
		},
	}
}

func (e completionEngine) Complete(ctx CompletionContext, req completionRequest) []Suggestion {
	for _, provider := range e.providers {
		suggestions := provider.Complete(ctx, req)
		if len(suggestions) > 0 {
			return suggestions
		}
	}
	return nil
}

func parseCompletionRequest(line string) completionRequest {
	trailingSpace := strings.HasSuffix(line, " ")
	trimmed := strings.TrimLeft(line, " ")
	fields := strings.Fields(trimmed)
	prefix := ""
	replaceFrom := len(line)
	replaceTo := len(line)
	if len(fields) > 0 {
		prefix = fields[len(fields)-1]
	}
	if trailingSpace {
		prefix = ""
	} else if prefix != "" {
		replaceFrom = len(line) - len(prefix)
	} else {
		replaceFrom = 0
		replaceTo = len(line)
	}
	return completionRequest{
		Line:          line,
		Fields:        fields,
		TrailingSpace: trailingSpace,
		Prefix:        prefix,
		ReplaceFrom:   replaceFrom,
		ReplaceTo:     replaceTo,
	}
}

func toUICompletion(req completionRequest, suggestions []Suggestion) ui.Completion {
	filtered := filterSuggestionsByPrefix(suggestions, req.Prefix)
	uiSuggestions := make([]ui.Suggestion, 0, len(filtered))
	for _, suggestion := range filtered {
		uiSuggestions = append(uiSuggestions, ui.Suggestion{
			Value:       suggestion.Value,
			Description: suggestion.Description,
			Category:    suggestion.Category,
			Replacement: suggestion.Value,
			ReplaceFrom: req.ReplaceFrom,
			ReplaceTo:   req.ReplaceTo,
		})
	}

	return ui.Completion{
		Prefix:      req.Prefix,
		Suggestions: uiSuggestions,
		Hint:        completionHint(req, filtered),
	}
}

func filterSuggestionsByPrefix(suggestions []Suggestion, prefix string) []Suggestion {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return append([]Suggestion(nil), suggestions...)
	}

	filtered := make([]Suggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		if strings.HasPrefix(suggestion.Value, prefix) {
			filtered = append(filtered, suggestion)
		}
	}
	return filtered
}

func completionHint(req completionRequest, suggestions []Suggestion) string {
	if req.TrailingSpace || req.Prefix == "" || len(suggestions) == 0 {
		return ""
	}
	first := suggestions[0].Value
	if !strings.HasPrefix(first, req.Prefix) || len(first) <= len(req.Prefix) {
		return ""
	}
	return first[len(req.Prefix):]
}

func completeRootCommands(_ CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 0 || (len(req.Fields) == 1 && !req.TrailingSpace) {
		return rootSuggestions
	}
	return nil
}

func completeConnectionCommands(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 0 || req.Fields[0] != "connection" {
		return nil
	}

	subcommands := []Suggestion{
		{Value: "create", Description: "create a saved connection", Category: "subcommand"},
		{Value: "edit", Description: "edit a saved connection", Category: "subcommand"},
		{Value: "delete", Description: "delete a saved connection", Category: "subcommand"},
		{Value: "show", Description: "show a saved connection", Category: "subcommand"},
		{Value: "test", Description: "test a saved connection", Category: "subcommand"},
		{Value: "doctor", Description: "inspect a saved connection statically", Category: "subcommand"},
	}

	switch len(req.Fields) {
	case 1:
		return subcommands
	case 2:
		if req.TrailingSpace && req.Fields[1] == "test" {
			suggestions := connectionSuggestions(ctx)
			suggestions = append(suggestions, Suggestion{Value: "verbose", Description: "show detailed timing and targets", Category: "flag"})
			return suggestions
		}
		if !req.TrailingSpace {
			return subcommands
		}
		if slices.Contains([]string{"edit", "delete", "show", "test", "doctor"}, req.Fields[1]) {
			return connectionSuggestions(ctx)
		}
	case 3:
		if slices.Contains([]string{"edit", "delete", "show", "doctor"}, req.Fields[1]) {
			return connectionSuggestions(ctx)
		}
		if req.Fields[1] == "test" {
			if req.TrailingSpace {
				return []Suggestion{{Value: "verbose", Description: "show detailed timing and targets", Category: "flag"}}
			}
			suggestions := connectionSuggestions(ctx)
			suggestions = append(suggestions, Suggestion{Value: "verbose", Description: "show detailed timing and targets", Category: "flag"})
			return suggestions
		}
	case 4:
		if req.Fields[1] == "test" {
			return []Suggestion{{Value: "verbose", Description: "show detailed timing and targets", Category: "flag"}}
		}
	}
	return nil
}

func completeConnectNames(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 0 || req.Fields[0] != "connect" {
		return nil
	}
	if len(req.Fields) == 1 && !req.TrailingSpace {
		return nil
	}
	return connectionSuggestions(ctx)
}

func completeUseDatabases(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 0 || req.Fields[0] != "use" {
		return nil
	}
	if len(req.Fields) == 1 && req.TrailingSpace {
		return stringSuggestions(ctx.Databases, "database")
	}
	if len(req.Fields) == 2 {
		return stringSuggestions(ctx.Databases, "database")
	}
	return nil
}

func completeDropUsers(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "drop" || req.Fields[1] != "user" {
		return nil
	}
	if len(req.Fields) == 2 && req.TrailingSpace {
		return stringSuggestions(ctx.Users, "user")
	}
	if len(req.Fields) == 3 {
		return stringSuggestions(ctx.Users, "user")
	}
	return nil
}

func completeShowGrantsUsers(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "show" || req.Fields[1] != "grants" {
		return nil
	}
	if len(req.Fields) == 2 && req.TrailingSpace {
		return stringSuggestions(ctx.Users, "user")
	}
	if len(req.Fields) == 3 {
		return stringSuggestions(ctx.Users, "user")
	}
	return nil
}

func completeShowColumnsTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 0 {
		return nil
	}
	if req.Fields[0] == "columns" {
		if len(req.Fields) == 1 && req.TrailingSpace {
			return stringSuggestions(ctx.Tables, "table")
		}
		if len(req.Fields) == 2 {
			return stringSuggestions(ctx.Tables, "table")
		}
		return nil
	}
	if len(req.Fields) < 2 || req.Fields[0] != "show" || req.Fields[1] != "columns" {
		return nil
	}
	if len(req.Fields) == 2 && !req.TrailingSpace {
		return []Suggestion{{Value: "columns", Description: "show columns for a table", Category: "subcommand"}}
	}
	if len(req.Fields) == 2 && req.TrailingSpace {
		return stringSuggestions(ctx.Tables, "table")
	}
	if len(req.Fields) == 3 {
		return stringSuggestions(ctx.Tables, "table")
	}
	return nil
}

func completeShowForeignKeysTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "show" {
		return nil
	}
	if req.Fields[1] == "fks" {
		if len(req.Fields) == 2 && !req.TrailingSpace {
			return []Suggestion{{Value: "fks", Description: "alias for show foreign keys", Category: "alias"}}
		}
		if len(req.Fields) == 2 && req.TrailingSpace {
			return stringSuggestions(ctx.Tables, "table")
		}
		if len(req.Fields) == 3 {
			return stringSuggestions(ctx.Tables, "table")
		}
		return nil
	}
	if req.Fields[1] != "foreign" {
		return nil
	}
	if len(req.Fields) == 2 && !req.TrailingSpace {
		return []Suggestion{{Value: "foreign", Description: "show foreign key details", Category: "subcommand"}}
	}
	if len(req.Fields) == 2 && req.TrailingSpace {
		return []Suggestion{{Value: "keys", Description: "show foreign keys for a table", Category: "subcommand"}}
	}
	if len(req.Fields) == 3 && req.Fields[2] == "keys" {
		if req.TrailingSpace {
			return stringSuggestions(ctx.Tables, "table")
		}
		return []Suggestion{{Value: "keys", Description: "show foreign keys for a table", Category: "subcommand"}}
	}
	if len(req.Fields) == 4 && req.Fields[2] == "keys" {
		return stringSuggestions(ctx.Tables, "table")
	}
	return nil
}

func completeCountPeekSampleTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 0 {
		return nil
	}
	switch req.Fields[0] {
	case "count":
		if len(req.Fields) == 1 && req.TrailingSpace {
			return stringSuggestions(ctx.Tables, "table")
		}
		if len(req.Fields) == 2 {
			if req.Fields[1] == "rows" {
				if req.TrailingSpace {
					return stringSuggestions(ctx.Tables, "table")
				}
				return []Suggestion{{Value: "rows", Description: "count rows in a table", Category: "subcommand"}}
			}
			return stringSuggestions(ctx.Tables, "table")
		}
		if len(req.Fields) == 3 && req.Fields[1] == "rows" {
			return stringSuggestions(ctx.Tables, "table")
		}
	case "peek", "sample":
		if len(req.Fields) == 1 && req.TrailingSpace {
			return stringSuggestions(ctx.Tables, "table")
		}
		if len(req.Fields) == 2 {
			if req.Fields[1] == "rows" {
				if req.TrailingSpace {
					return stringSuggestions(ctx.Tables, "table")
				}
				return []Suggestion{{Value: "rows", Description: req.Fields[0] + " bounded rows from a table", Category: "subcommand"}}
			}
			return stringSuggestions(ctx.Tables, "table")
		}
		if len(req.Fields) == 3 && req.Fields[1] == "rows" {
			return stringSuggestions(ctx.Tables, "table")
		}
	}
	return nil
}

func completeDescribeTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 0 || req.Fields[0] != "describe" {
		return nil
	}
	switch len(req.Fields) {
	case 1:
		if req.TrailingSpace {
			return stringSuggestions(ctx.Tables, "table")
		}
	case 2:
		if req.Fields[1] == "table" && req.TrailingSpace {
			return stringSuggestions(ctx.Tables, "table")
		}
		if req.Fields[1] == "table" {
			return []Suggestion{{Value: "table", Description: "describe a table", Category: "subcommand"}}
		}
		return stringSuggestions(ctx.Tables, "table")
	case 3:
		if req.Fields[1] == "table" {
			return stringSuggestions(ctx.Tables, "table")
		}
	}
	return nil
}

func completeShowTables(_ CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 2 && req.Fields[0] == "show" && !req.TrailingSpace && req.Fields[1] == "tables" {
		return []Suggestion{{Value: "tables", Description: "list tables in current database", Category: "subcommand"}}
	}
	return nil
}

func completeShowIndexesTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "show" {
		return nil
	}
	if req.Fields[1] != "indexes" && req.Fields[1] != "index" {
		return nil
	}

	switch len(req.Fields) {
	case 2:
		if req.TrailingSpace {
			suggestions := []Suggestion{{Value: "on", Description: "show indexes on a table", Category: "keyword"}}
			return append(suggestions, stringSuggestions(ctx.Tables, "table")...)
		}
		return []Suggestion{{Value: req.Fields[1], Description: "show indexes for a table", Category: "subcommand"}}
	case 3:
		if req.Fields[2] == "on" {
			if req.TrailingSpace {
				return stringSuggestions(ctx.Tables, "table")
			}
			return []Suggestion{{Value: "on", Description: "show indexes on a table", Category: "keyword"}}
		}
		return stringSuggestions(ctx.Tables, "table")
	case 4:
		if req.Fields[2] == "on" {
			return stringSuggestions(ctx.Tables, "table")
		}
	}
	return nil
}

func completeShowCreateTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "show" {
		return nil
	}
	if len(req.Fields) == 2 && !req.TrailingSpace && req.Fields[1] == "create" {
		return []Suggestion{{Value: "create", Description: "show CREATE statements", Category: "subcommand"}}
	}
	if len(req.Fields) == 2 && req.TrailingSpace && req.Fields[1] == "create" {
		return []Suggestion{{Value: "table", Description: "show CREATE TABLE for a table", Category: "subcommand"}}
	}
	if len(req.Fields) == 3 && req.Fields[1] == "create" {
		if !req.TrailingSpace && req.Fields[2] == "table" {
			return []Suggestion{{Value: "table", Description: "show CREATE TABLE for a table", Category: "subcommand"}}
		}
		if req.TrailingSpace && req.Fields[2] == "table" {
			return stringSuggestions(ctx.Tables, "table")
		}
	}
	if len(req.Fields) == 4 && req.Fields[1] == "create" && req.Fields[2] == "table" {
		return stringSuggestions(ctx.Tables, "table")
	}
	return nil
}

func completeShowTableStatusTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "show" {
		return nil
	}
	if len(req.Fields) == 2 && !req.TrailingSpace && req.Fields[1] == "table" {
		return []Suggestion{{Value: "table", Description: "show table details", Category: "subcommand"}}
	}
	if len(req.Fields) == 2 && req.TrailingSpace && req.Fields[1] == "table" {
		return []Suggestion{{Value: "status", Description: "show table status", Category: "subcommand"}}
	}
	if len(req.Fields) == 3 && req.Fields[1] == "table" {
		if !req.TrailingSpace && req.Fields[2] == "status" {
			return []Suggestion{{Value: "status", Description: "show table status", Category: "subcommand"}}
		}
		if req.TrailingSpace && req.Fields[2] == "status" {
			return stringSuggestions(ctx.Tables, "table")
		}
	}
	if len(req.Fields) == 4 && req.Fields[1] == "table" && req.Fields[2] == "status" {
		return stringSuggestions(ctx.Tables, "table")
	}
	return nil
}

func completeShowVariablesPatterns(_ CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "show" {
		return nil
	}
	if req.Fields[1] != "variables" && req.Fields[1] != "vars" {
		return nil
	}
	if len(req.Fields) == 2 && !req.TrailingSpace {
		return []Suggestion{{Value: req.Fields[1], Description: "show MySQL system variables", Category: "subcommand"}}
	}
	if (len(req.Fields) == 2 && req.TrailingSpace) || len(req.Fields) == 3 {
		return commonVariableSuggestions()
	}
	return nil
}

func completeTruncateTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "truncate" || req.Fields[1] != "table" {
		return nil
	}
	if len(req.Fields) == 2 && req.TrailingSpace {
		return stringSuggestions(ctx.Tables, "table")
	}
	if len(req.Fields) == 3 {
		return stringSuggestions(ctx.Tables, "table")
	}
	return nil
}

func completeRenameTables(ctx CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) < 2 || req.Fields[0] != "rename" || req.Fields[1] != "table" {
		return nil
	}
	if len(req.Fields) == 2 && req.TrailingSpace {
		return stringSuggestions(ctx.Tables, "table")
	}
	if len(req.Fields) == 3 {
		return stringSuggestions(ctx.Tables, "table")
	}
	return nil
}

func completeCreateDropShowListHelpAuditTree(_ CompletionContext, req completionRequest) []Suggestion {
	if len(req.Fields) == 0 {
		return nil
	}

	switch req.Fields[0] {
	case "create":
		return []Suggestion{
			{Value: "database", Description: "create a database", Category: "subcommand"},
			{Value: "user", Description: "create a MySQL user", Category: "subcommand"},
		}
	case "count":
		return []Suggestion{{Value: "rows", Description: "count rows in a table", Category: "subcommand"}}
	case "drop":
		if len(req.Fields) == 1 || (len(req.Fields) == 2 && req.Fields[1] != "user") {
			return []Suggestion{
				{Value: "database", Description: "drop a database", Category: "subcommand"},
				{Value: "user", Description: "drop a MySQL user", Category: "subcommand"},
			}
		}
	case "list":
		return []Suggestion{
			{Value: "databases", Description: "alias for show databases", Category: "alias"},
			{Value: "users", Description: "alias for show users", Category: "alias"},
		}
	case "peek":
		return []Suggestion{{Value: "rows", Description: "peek bounded rows from a table", Category: "subcommand"}}
	case "sample":
		return []Suggestion{{Value: "rows", Description: "sample bounded rows from a table", Category: "subcommand"}}
	case "show":
		if len(req.Fields) == 1 {
			return []Suggestion{
				{Value: "columns", Description: "show columns for a table", Category: "subcommand"},
				{Value: "create", Description: "show CREATE statements", Category: "subcommand"},
				{Value: "databases", Description: "list databases on the active connection", Category: "subcommand"},
				{Value: "dbs", Description: "alias for show databases", Category: "alias"},
				{Value: "foreign", Description: "show foreign key details", Category: "subcommand"},
				{Value: "fks", Description: "alias for show foreign keys", Category: "alias"},
				{Value: "view", Description: "alias for show views", Category: "alias"},
				{Value: "views", Description: "show views in current database", Category: "subcommand"},
				{Value: "index", Description: "alias for show indexes", Category: "alias"},
				{Value: "indexes", Description: "show indexes for a table", Category: "subcommand"},
				{Value: "processes", Description: "alias for show processlist", Category: "alias"},
				{Value: "processlist", Description: "show the active MySQL processlist", Category: "subcommand"},
				{Value: "table", Description: "show table details", Category: "subcommand"},
				{Value: "tables", Description: "list tables in current database", Category: "subcommand"},
				{Value: "trigger", Description: "alias for show triggers", Category: "alias"},
				{Value: "triggers", Description: "show triggers in current database", Category: "subcommand"},
				{Value: "users", Description: "list MySQL users", Category: "subcommand"},
				{Value: "grants", Description: "show grants for a MySQL user", Category: "subcommand"},
				{Value: "user", Description: "show user aliases", Category: "alias"},
				{Value: "variables", Description: "show MySQL system variables", Category: "subcommand"},
				{Value: "vars", Description: "alias for show variables", Category: "alias"},
			}
		}
		if len(req.Fields) == 2 && req.Fields[1] == "user" {
			return []Suggestion{{Value: "accounts", Description: "alias for show users", Category: "alias"}}
		}
		if len(req.Fields) == 2 {
			return []Suggestion{
				{Value: "columns", Description: "show columns for a table", Category: "subcommand"},
				{Value: "create", Description: "show CREATE statements", Category: "subcommand"},
				{Value: "databases", Description: "list databases on the active connection", Category: "subcommand"},
				{Value: "dbs", Description: "alias for show databases", Category: "alias"},
				{Value: "foreign", Description: "show foreign key details", Category: "subcommand"},
				{Value: "fks", Description: "alias for show foreign keys", Category: "alias"},
				{Value: "view", Description: "alias for show views", Category: "alias"},
				{Value: "views", Description: "show views in current database", Category: "subcommand"},
				{Value: "index", Description: "alias for show indexes", Category: "alias"},
				{Value: "indexes", Description: "show indexes for a table", Category: "subcommand"},
				{Value: "processes", Description: "alias for show processlist", Category: "alias"},
				{Value: "processlist", Description: "show the active MySQL processlist", Category: "subcommand"},
				{Value: "table", Description: "show table details", Category: "subcommand"},
				{Value: "tables", Description: "list tables in current database", Category: "subcommand"},
				{Value: "trigger", Description: "alias for show triggers", Category: "alias"},
				{Value: "triggers", Description: "show triggers in current database", Category: "subcommand"},
				{Value: "users", Description: "list MySQL users", Category: "subcommand"},
				{Value: "grants", Description: "show grants for a MySQL user", Category: "subcommand"},
				{Value: "user", Description: "show user aliases", Category: "alias"},
				{Value: "variables", Description: "show MySQL system variables", Category: "subcommand"},
				{Value: "vars", Description: "alias for show variables", Category: "alias"},
			}
		}
	case "truncate":
		return []Suggestion{{Value: "table", Description: "delete all rows from a table", Category: "subcommand"}}
	case "rename":
		return []Suggestion{{Value: "table", Description: "rename a table", Category: "subcommand"}}
	case "help":
		return []Suggestion{
			{Value: "connect", Description: "connect help", Category: "topic"},
			{Value: "connections", Description: "connections help", Category: "topic"},
			{Value: "audit log", Description: "audit log help", Category: "topic"},
			{Value: "connection", Description: "connection command help", Category: "topic"},
			{Value: "connection create", Description: "connection create help", Category: "topic"},
			{Value: "connection edit", Description: "connection edit help", Category: "topic"},
			{Value: "connection delete", Description: "connection delete help", Category: "topic"},
			{Value: "connection show", Description: "connection show help", Category: "topic"},
			{Value: "connection test", Description: "connection test help", Category: "topic"},
			{Value: "connection doctor", Description: "connection doctor help", Category: "topic"},
			{Value: "count rows", Description: "count rows help", Category: "topic"},
			{Value: "create database", Description: "create database help", Category: "topic"},
			{Value: "create user", Description: "create user help", Category: "topic"},
			{Value: "peek rows", Description: "peek rows help", Category: "topic"},
			{Value: "sample rows", Description: "sample rows help", Category: "topic"},
			{Value: "show databases", Description: "show databases help", Category: "topic"},
			{Value: "show columns", Description: "show columns help", Category: "topic"},
			{Value: "show create table", Description: "show create table help", Category: "topic"},
			{Value: "show foreign keys", Description: "show foreign keys help", Category: "topic"},
			{Value: "show tables", Description: "show tables help", Category: "topic"},
			{Value: "show indexes", Description: "show indexes help", Category: "topic"},
			{Value: "show table status", Description: "show table status help", Category: "topic"},
			{Value: "show users", Description: "show users help", Category: "topic"},
			{Value: "show grants", Description: "show grants help", Category: "topic"},
			{Value: "show processlist", Description: "show processlist help", Category: "topic"},
			{Value: "show triggers", Description: "show triggers help", Category: "topic"},
			{Value: "show variables", Description: "show variables help", Category: "topic"},
			{Value: "show views", Description: "show views help", Category: "topic"},
			{Value: "drop database", Description: "drop database help", Category: "topic"},
			{Value: "drop user", Description: "drop user help", Category: "topic"},
			{Value: "describe", Description: "describe help", Category: "topic"},
			{Value: "truncate table", Description: "truncate table help", Category: "topic"},
			{Value: "rename table", Description: "rename table help", Category: "topic"},
			{Value: "use", Description: "use help", Category: "topic"},
			{Value: "status", Description: "status help", Category: "topic"},
			{Value: "context", Description: "context help", Category: "topic"},
			{Value: "dry-run", Description: "dry-run help", Category: "topic"},
			{Value: "aliases", Description: "alias help", Category: "topic"},
			{Value: "exit", Description: "exit help", Category: "topic"},
		}
	case "audit":
		return []Suggestion{{Value: "log", Description: "show recent audit entries", Category: "subcommand"}}
	case "dry-run", "dry":
		return []Suggestion{
			{Value: "on", Description: "enable dry-run mode", Category: "subcommand"},
			{Value: "off", Description: "disable dry-run mode", Category: "subcommand"},
		}
	}
	return nil
}

func connectionSuggestions(ctx CompletionContext) []Suggestion {
	connections := append([]CompletionConnection(nil), ctx.Connections...)
	slices.SortFunc(connections, func(a CompletionConnection, b CompletionConnection) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		default:
			return 0
		}
	})

	suggestions := make([]Suggestion, 0, len(connections))
	for _, connection := range connections {
		description := strings.TrimSpace(connection.Driver + " " + connection.Mode)
		suggestions = append(suggestions, Suggestion{
			Value:       connection.Name,
			Description: description,
			Category:    "connection",
		})
	}
	return suggestions
}

func stringSuggestions(values []string, category string) []Suggestion {
	sorted := append([]string(nil), values...)
	slices.Sort(sorted)

	suggestions := make([]Suggestion, 0, len(sorted))
	for _, value := range sorted {
		suggestions = append(suggestions, Suggestion{
			Value:    value,
			Category: category,
		})
	}
	return suggestions
}

func commonVariableSuggestions() []Suggestion {
	suggestions := make([]Suggestion, 0, len(commonVariableNames))
	for _, name := range commonVariableNames {
		suggestions = append(suggestions, Suggestion{
			Value:    name,
			Category: "variable",
		})
	}
	return suggestions
}
