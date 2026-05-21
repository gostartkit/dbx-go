package app

import (
	"errors"
	"io"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/ui"
)

var errREPLExit = errors.New("repl exit")

type staticCompletionResolver struct {
	ctx CompletionContext
}

func (r staticCompletionResolver) Connections() []CompletionConnection {
	return append([]CompletionConnection(nil), r.ctx.Connections...)
}

func (r staticCompletionResolver) Databases() []string {
	return append([]string(nil), r.ctx.Databases...)
}

func (r staticCompletionResolver) Tables() []string {
	return append([]string(nil), r.ctx.Tables...)
}

func (r staticCompletionResolver) Templates() []string {
	return append([]string(nil), r.ctx.Templates...)
}

func (r staticCompletionResolver) TemplateTags() []string {
	return append([]string(nil), r.ctx.TemplateTags...)
}

func (r staticCompletionResolver) Users() []string {
	return append([]string(nil), r.ctx.Users...)
}

func (a *Application) replCommandApp() *cmd.App {
	if a == nil {
		return nil
	}
	if a.replApp == nil {
		a.replApp = newREPLBuilder(a, a).buildApp()
	}
	return a.replApp
}

func (a *Application) Connections() []CompletionConnection {
	connections, err := a.store.ListConnections()
	if err != nil {
		return nil
	}
	results := make([]CompletionConnection, 0, len(connections))
	for _, connection := range connections {
		results = append(results, CompletionConnection{
			Name:   connection.Name,
			Driver: connection.Driver,
			Mode:   connection.Mode,
		})
	}
	return results
}

func (a *Application) Databases() []string {
	return a.currentCompletionDatabases()
}

func (a *Application) Tables() []string {
	return a.currentCompletionTables()
}

func (a *Application) Templates() []string {
	return a.currentCompletionTemplates()
}

func (a *Application) TemplateTags() []string {
	return a.currentCompletionTemplateTags()
}

func (a *Application) Users() []string {
	return a.currentCompletionUsers()
}

func (b *cliBuilder) completeConnections(cmd.CompletionContext) []string {
	if b.resolver == nil {
		return nil
	}
	connections := b.resolver.Connections()
	values := make([]string, 0, len(connections))
	for _, connection := range connections {
		values = append(values, connection.Name)
	}
	return values
}

func (b *cliBuilder) completeDatabases(cmd.CompletionContext) []string {
	if b.resolver == nil {
		return nil
	}
	return b.resolver.Databases()
}

func (b *cliBuilder) completeTables(cmd.CompletionContext) []string {
	if b.resolver == nil {
		return nil
	}
	return b.resolver.Tables()
}

func (b *cliBuilder) completeUsers(cmd.CompletionContext) []string {
	if b.resolver == nil {
		return nil
	}
	return b.resolver.Users()
}

func (b *cliBuilder) completeTemplates(cmd.CompletionContext) []string {
	if b.resolver == nil {
		return nil
	}
	return b.resolver.Templates()
}

func (b *cliBuilder) completeTemplateTags(cmd.CompletionContext) []string {
	if b.resolver == nil {
		return nil
	}
	return b.resolver.TemplateTags()
}

func (b *cliBuilder) completeVariables(cmd.CompletionContext) []string {
	return []string{"max_connections", "wait_timeout", "innodb_buffer_pool_size"}
}

func calculateCompletion(line string, ctx CompletionContext) ui.Completion {
	replApp := (&cliBuilder{
		mode:     ModeREPL,
		out:      io.Discard,
		err:      io.Discard,
		globals:  &cliGlobals{Format: "text"},
		resolver: staticCompletionResolver{ctx: ctx},
	}).buildApp()
	return completionFromApp(replApp, line, staticCompletionResolver{ctx: ctx})
}

func completionFromApp(app *cmd.App, line string, resolver completionResolver) ui.Completion {
	if app == nil {
		return ui.Completion{}
	}

	if strings.HasPrefix(strings.TrimSpace(line), "help") {
		return helpTopicCompletion(line)
	}

	results := app.CompleteLineDetailed(line, len(line))
	args, current, _ := cmd.SplitLineForCompletion(line)
	replaceFrom := len(line) - len(current)
	replaceTo := len(line)

	suggestions := make([]ui.Suggestion, 0, len(results))
	for _, result := range results {
		if !includeCompletionResult(result) {
			continue
		}
		description := result.Description
		if description == "" {
			description = dynamicCompletionDescription(args, result.Value, resolver)
		}
		suggestions = append(suggestions, ui.Suggestion{
			Value:       result.Value,
			Description: description,
			Category:    result.Kind,
			Replacement: result.Value,
			ReplaceFrom: replaceFrom,
			ReplaceTo:   replaceTo,
		})
	}

	return ui.Completion{
		Prefix:      current,
		Suggestions: suggestions,
		Hint:        completionHint(current, suggestions),
	}
}

func helpTopicCompletion(line string) ui.Completion {
	args, current, _ := cmd.SplitLineForCompletion(line)
	replaceFrom := len(line) - len(current)
	replaceTo := len(line)
	suggestions := make([]ui.Suggestion, 0)
	for _, suggestion := range helpCompletionTopics() {
		if current != "" && !strings.HasPrefix(suggestion.Value, current) {
			continue
		}
		suggestions = append(suggestions, ui.Suggestion{
			Value:       suggestion.Value,
			Description: suggestion.Description,
			Category:    suggestion.Category,
			Replacement: suggestion.Value,
			ReplaceFrom: replaceFrom,
			ReplaceTo:   replaceTo,
		})
	}
	prefix := current
	if len(args) == 0 {
		prefix = ""
	}
	return ui.Completion{
		Prefix:      prefix,
		Suggestions: suggestions,
		Hint:        completionHint(current, suggestions),
	}
}

func includeCompletionResult(result cmd.CompletionResult) bool {
	if result.Kind != "builtin" {
		return true
	}
	return result.Value == "help"
}

func dynamicCompletionDescription(args []string, value string, resolver completionResolver) string {
	if len(args) == 0 {
		return ""
	}

	switch {
	case len(args) == 1 && args[0] == "connect":
		for _, connection := range resolverConnections(resolver) {
			if connection.Name == value {
				return strings.TrimSpace(connection.Driver + " " + connection.Mode)
			}
		}
	case len(args) == 2 && args[0] == "show" && args[1] == "connection":
		for _, connection := range resolverConnections(resolver) {
			if connection.Name == value {
				return strings.TrimSpace(connection.Driver + " " + connection.Mode)
			}
		}
	case len(args) == 2 && args[0] == "drop" && args[1] == "connection":
		for _, connection := range resolverConnections(resolver) {
			if connection.Name == value {
				return strings.TrimSpace(connection.Driver + " " + connection.Mode)
			}
		}
	case len(args) == 2 && args[0] == "use" && args[1] == "database":
		if value != "" {
			return "database"
		}
	case len(args) == 2 && args[0] == "show" && args[1] == "context":
		return "context"
	case len(args) == 2 && args[0] == "run" && args[1] == "template":
		if value != "" {
			return "template"
		}
	case len(args) == 2 && args[0] == "show" && (args[1] == "columns" || args[1] == "rows" || args[1] == "table"):
		if value != "" {
			return "table"
		}
	}

	return ""
}

func completionHint(prefix string, suggestions []ui.Suggestion) string {
	if prefix == "" || len(suggestions) == 0 {
		return ""
	}

	common := suggestions[0].Value
	for _, suggestion := range suggestions[1:] {
		common = commonPrefix(common, suggestion.Value)
		if common == "" {
			return ""
		}
	}
	if len(common) <= len(prefix) {
		if strings.HasPrefix(suggestions[0].Value, prefix) && len(suggestions[0].Value) > len(prefix) {
			return suggestions[0].Value[len(prefix):]
		}
		return ""
	}
	return common[len(prefix):]
}

func commonPrefix(left string, right string) string {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	index := 0
	for index < limit && left[index] == right[index] {
		index++
	}
	return left[:index]
}

func (a *Application) completeInput(line string) ui.Completion {
	return completionFromApp(a.replCommandApp(), line, a)
}

func resolverConnections(resolver completionResolver) []CompletionConnection {
	if resolver == nil {
		return nil
	}
	return resolver.Connections()
}
