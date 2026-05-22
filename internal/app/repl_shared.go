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

func (b *cliBuilder) completeOperations(cmd.CompletionContext) []string {
	values := operationNames(b.resolver, b.application)
	results := make([]string, 0, len(values))
	for _, value := range values {
		results = append(results, value.Name)
	}
	return results
}

func (b *cliBuilder) completeTemplateTags(cmd.CompletionContext) []string {
	if b.resolver == nil {
		return nil
	}
	return b.resolver.TemplateTags()
}

func (b *cliBuilder) completeHelpTopics(cmd.CompletionContext) []string {
	topics := helpCompletionTopics()
	results := make([]string, 0, len(topics))
	for _, topic := range topics {
		results = append(results, topic.Value)
	}
	return results
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
	return completionFromApp(replApp, ui.NewSingleLineCompletionRequest(line, len([]rune(line))), staticCompletionResolver{ctx: ctx})
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
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	limit := len(leftRunes)
	if len(rightRunes) < limit {
		limit = len(rightRunes)
	}
	index := 0
	for index < limit && leftRunes[index] == rightRunes[index] {
		index++
	}
	return string(leftRunes[:index])
}

func (a *Application) completeInput(request ui.CompletionRequest) ui.Completion {
	return completionFromApp(a.replCommandApp(), request, a)
}

func resolverConnections(resolver completionResolver) []CompletionConnection {
	if resolver == nil {
		return nil
	}
	return resolver.Connections()
}

func resolverTemplates(resolver completionResolver) []string {
	if resolver == nil {
		return nil
	}
	return resolver.Templates()
}
