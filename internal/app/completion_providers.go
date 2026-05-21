package app

import (
	"context"
	"slices"
	"sort"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/commandlang"
	"pkg.gostartkit.com/dbx/internal/ui"
)

type completionProvider interface {
	Name() string
	Complete(*providerContext) ([]ui.Suggestion, error)
}

type providerContext struct {
	request         ui.CompletionRequest
	fullPrefix      string
	localPrefix     string
	fullTokens      []commandlang.Token
	localTokens     []commandlang.Token
	commandContext  commandlang.CommandContext
	localContext    commandlang.CommandContext
	commandPath     []string
	parentCommand   []string
	positionalIndex int
	currentValue    string
	currentFlag     string
	expectingFlag   string
	replaceStart    int
	replaceEnd      int
	resolver        completionResolver
	app             *cmd.App
	application     *Application
}

type commandProvider struct{}
type operationProvider struct{}
type templateProvider struct{}
type connectionProvider struct{}
type databaseProvider struct{}
type tableProvider struct{}
type userProvider struct{}
type schemaProvider struct{}
type flagProvider struct{}
type flagValueProvider struct{}

func (commandProvider) Name() string    { return "command" }
func (operationProvider) Name() string  { return "operation" }
func (templateProvider) Name() string   { return "template" }
func (connectionProvider) Name() string { return "connection" }
func (databaseProvider) Name() string   { return "database" }
func (tableProvider) Name() string      { return "table" }
func (userProvider) Name() string       { return "user" }
func (schemaProvider) Name() string     { return "schema" }
func (flagProvider) Name() string       { return "flag" }
func (flagValueProvider) Name() string  { return "flag-value" }

func completionProviders() []completionProvider {
	return []completionProvider{
		flagValueProvider{},
		flagProvider{},
		commandProvider{},
		operationProvider{},
		templateProvider{},
		connectionProvider{},
		databaseProvider{},
		tableProvider{},
		userProvider{},
		schemaProvider{},
	}
}

func completionFromApp(app *cmd.App, request ui.CompletionRequest, resolver completionResolver) ui.Completion {
	if app == nil {
		return ui.Completion{}
	}

	ctx := buildProviderContext(app, request, resolver)
	suggestions := make([]ui.Suggestion, 0)
	seen := make(map[string]struct{})
	for _, provider := range completionProviders() {
		results, err := provider.Complete(ctx)
		if err != nil {
			continue
		}
		for _, suggestion := range results {
			key := suggestion.Value + "|" + suggestion.Description + "|" + suggestion.Category
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			suggestions = append(suggestions, suggestion)
		}
		if len(suggestions) > 0 {
			break
		}
	}

	sort.SliceStable(suggestions, func(i int, j int) bool {
		return suggestions[i].Value < suggestions[j].Value
	})

	return ui.Completion{
		Prefix:      ctx.currentValue,
		Suggestions: suggestions,
		Hint:        completionHint(ctx.currentValue, suggestions),
	}
}

func buildProviderContext(app *cmd.App, request ui.CompletionRequest, resolver completionResolver) *providerContext {
	fullPrefix := commandlang.JoinLogicalLines(bufferPrefixString(request.Buffer, request.Cursor))
	localPrefix := request.CurrentLinePrefix()
	fullTokens := commandlang.Lex(fullPrefix)
	localTokens := commandlang.Lex(localPrefix)
	fullContext := commandlang.BuildCommandContext(fullTokens, len([]rune(fullPrefix)))
	localContext := commandlang.BuildCommandContext(localTokens, len([]rune(localPrefix)))
	replaceStart, replaceEnd, currentValue, currentFlag := completionEditRange(localContext, len([]rune(localPrefix)))
	commandPath, parentPath, positionalIndex := resolveCompletionCommandPath(fullTokens)

	return &providerContext{
		request:         request,
		fullPrefix:      fullPrefix,
		localPrefix:     localPrefix,
		fullTokens:      fullTokens,
		localTokens:     localTokens,
		commandContext:  fullContext,
		localContext:    localContext,
		commandPath:     commandPath,
		parentCommand:   parentPath,
		positionalIndex: positionalIndex,
		currentValue:    currentValue,
		currentFlag:     currentFlag,
		expectingFlag:   fullContext.ExpectingValueForFlag,
		replaceStart:    replaceStart,
		replaceEnd:      replaceEnd,
		resolver:        resolver,
		app:             app,
		application:     applicationFromResolver(resolver),
	}
}

func (p commandProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	if strings.HasPrefix(strings.TrimSpace(ctx.fullPrefix), "help") {
		suggestions := make([]ui.Suggestion, 0)
		for _, suggestion := range helpCompletionTopics() {
			if ctx.currentValue != "" && !strings.HasPrefix(suggestion.Value, ctx.currentValue) {
				continue
			}
			suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, suggestion.Value, suggestion.Value, suggestion.Description, suggestion.Category))
		}
		return suggestions, nil
	}

	parent := ctx.parentCommand
	if len(ctx.commandPath) == 0 {
		parent = nil
	}
	children := directChildCommands(parent)
	if len(children) == 0 {
		return nil, nil
	}
	suggestions := make([]ui.Suggestion, 0)
	for _, child := range children {
		if ctx.currentValue != "" && !strings.HasPrefix(child.Value, ctx.currentValue) {
			continue
		}
		suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, child.Value, child.Value, child.Description, "command"))
	}
	return suggestions, nil
}

func (p operationProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	if !slices.Equal(ctx.commandPath, []string{"exec"}) || ctx.positionalIndex != 0 {
		return nil, nil
	}
	values := operationNames(ctx.resolver, ctx.application)
	suggestions := make([]ui.Suggestion, 0, len(values))
	for _, value := range values {
		if ctx.currentValue != "" && !strings.HasPrefix(value.Name, ctx.currentValue) {
			continue
		}
		suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, value.Name, value.Name, value.Description, "operation"))
	}
	return suggestions, nil
}

func (p templateProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	switch {
	case slices.Equal(ctx.commandPath, []string{"describe", "template"}) && ctx.positionalIndex == 0:
	case slices.Equal(ctx.commandPath, []string{"template", "validate"}) && ctx.positionalIndex == 0:
	default:
		return nil, nil
	}
	values := resolverTemplates(ctx.resolver)
	suggestions := make([]ui.Suggestion, 0, len(values))
	for _, value := range values {
		if ctx.currentValue != "" && !strings.HasPrefix(value, ctx.currentValue) {
			continue
		}
		suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, value, value, "template", "template"))
	}
	return suggestions, nil
}

func (p connectionProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	switch {
	case slices.Equal(ctx.commandPath, []string{"connect"}) && ctx.positionalIndex == 0:
	case slices.Equal(ctx.commandPath, []string{"show", "connection"}) && ctx.positionalIndex == 0:
	case slices.Equal(ctx.commandPath, []string{"drop", "connection"}) && ctx.positionalIndex == 0:
	default:
		return nil, nil
	}
	suggestions := make([]ui.Suggestion, 0)
	for _, connection := range resolverConnections(ctx.resolver) {
		if ctx.currentValue != "" && !strings.HasPrefix(connection.Name, ctx.currentValue) {
			continue
		}
		suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, connection.Name, connection.Name, strings.TrimSpace(connection.Driver+" "+connection.Mode), "connection"))
	}
	return suggestions, nil
}

func (p databaseProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	switch {
	case slices.Equal(ctx.commandPath, []string{"use", "database"}) && ctx.positionalIndex == 0:
	default:
		return nil, nil
	}
	values := ctx.resolver.Databases()
	suggestions := make([]ui.Suggestion, 0, len(values))
	for _, value := range values {
		if ctx.currentValue != "" && !strings.HasPrefix(value, ctx.currentValue) {
			continue
		}
		suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, value, value, "database", "database"))
	}
	return suggestions, nil
}

func (p tableProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	switch {
	case slices.Equal(ctx.commandPath, []string{"show", "table"}) && ctx.positionalIndex == 0:
	case slices.Equal(ctx.commandPath, []string{"show", "columns"}) && ctx.positionalIndex == 0:
	case slices.Equal(ctx.commandPath, []string{"show", "rows"}) && ctx.positionalIndex == 0:
	default:
		return nil, nil
	}
	values := ctx.resolver.Tables()
	suggestions := make([]ui.Suggestion, 0, len(values))
	for _, value := range values {
		if ctx.currentValue != "" && !strings.HasPrefix(value, ctx.currentValue) {
			continue
		}
		suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, value, value, "table", "table"))
	}
	return suggestions, nil
}

func (p userProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	if !slices.Equal(ctx.commandPath, []string{"drop", "user"}) || ctx.positionalIndex != 0 {
		return nil, nil
	}
	values := ctx.resolver.Users()
	suggestions := make([]ui.Suggestion, 0, len(values))
	for _, value := range values {
		if ctx.currentValue != "" && !strings.HasPrefix(value, ctx.currentValue) {
			continue
		}
		suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, value, value, "user", "user"))
	}
	return suggestions, nil
}

func (p schemaProvider) Complete(*providerContext) ([]ui.Suggestion, error) {
	return nil, nil
}

func (p flagProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	flags := flagsForCommandPath(ctx.commandPath)
	if len(flags) == 0 {
		return nil, nil
	}
	if ctx.expectingFlag != "" {
		return nil, nil
	}
	if ctx.currentValue == "" && ctx.currentFlag == "" {
		return nil, nil
	}
	if ctx.currentValue != "" && !strings.HasPrefix(ctx.currentValue, "--") && ctx.currentFlag == "" {
		return nil, nil
	}
	suggestions := make([]ui.Suggestion, 0, len(flags))
	for _, flag := range flags {
		if ctx.currentValue != "" && !strings.HasPrefix(flag, ctx.currentValue) {
			continue
		}
		suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, flag, flag, "flag", "flag"))
	}
	return suggestions, nil
}

func (p flagValueProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	if ctx.expectingFlag == "" {
		return nil, nil
	}
	switch ctx.expectingFlag {
	case "--tag":
		values := ctx.resolver.TemplateTags()
		suggestions := make([]ui.Suggestion, 0, len(values))
		for _, value := range values {
			if ctx.currentValue != "" && !strings.HasPrefix(value, ctx.currentValue) {
				continue
			}
			suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, value, value, "tag", "tag"))
		}
		return suggestions, nil
	}
	return nil, nil
}

func directChildCommands(parent []string) []Suggestion {
	specs := replCommandSpecs()
	seen := make(map[string]struct{})
	suggestions := make([]Suggestion, 0)
	for _, spec := range specs {
		if spec.Hidden || (spec.Category != "command" && spec.Category != "alias") {
			continue
		}
		tokens := commandlangTokensForPath(spec.Path)
		if len(tokens) != len(parent)+1 {
			continue
		}
		if !slices.Equal(tokens[:len(parent)], parent) {
			continue
		}
		value := tokens[len(tokens)-1]
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		suggestions = append(suggestions, Suggestion{Value: value, Description: spec.Description, Category: "command"})
	}
	return suggestions
}

func completionEditRange(ctx commandlang.CommandContext, cursor int) (int, int, string, string) {
	if ctx.CursorToken == nil {
		return cursor, cursor, "", ""
	}
	switch ctx.CursorToken.Type {
	case commandlang.TokenWord, commandlang.TokenString, commandlang.TokenFlag:
		return ctx.CursorToken.StartRune, ctx.CursorToken.EndRune, ctx.CursorToken.Literal, flagLiteral(*ctx.CursorToken)
	default:
		return cursor, cursor, "", ""
	}
}

func flagLiteral(token commandlang.Token) string {
	if token.Type == commandlang.TokenFlag {
		return token.Literal
	}
	return ""
}

func resolveCompletionCommandPath(tokens []commandlang.Token) ([]string, []string, int) {
	words := positionalWords(tokens)
	matched := longestCommandPath(words)
	parent := matched
	if len(words) > len(matched) {
		parent = matched
	}
	return matched, parent, max(0, len(words)-len(matched)-1)
}

func longestCommandPath(words []string) []string {
	best := []string(nil)
	for _, spec := range replCommandSpecs() {
		if spec.Hidden || (spec.Category != "command" && spec.Category != "alias") {
			continue
		}
		tokens := commandlangTokensForPath(spec.Path)
		if len(tokens) == 0 || len(tokens) > len(words) {
			continue
		}
		if slices.Equal(tokens, words[:len(tokens)]) && len(tokens) > len(best) {
			best = append([]string(nil), tokens...)
		}
	}
	return best
}

func positionalWords(tokens []commandlang.Token) []string {
	values := make([]string, 0)
	var activeFlag bool
	for idx := 0; idx < len(tokens); idx++ {
		token := tokens[idx]
		switch token.Type {
		case commandlang.TokenEOF, commandlang.TokenNewline, commandlang.TokenPipe:
			return values
		case commandlang.TokenFlag:
			activeFlag = true
		case commandlang.TokenEquals:
			continue
		case commandlang.TokenWord, commandlang.TokenString:
			if activeFlag {
				activeFlag = false
				continue
			}
			values = append(values, token.Literal)
		}
	}
	return values
}

func commandlangTokensForPath(path string) []string {
	tokens := commandlang.Lex(path)
	values := make([]string, 0)
	for _, token := range tokens {
		if token.Type == commandlang.TokenWord || token.Type == commandlang.TokenString {
			values = append(values, token.Literal)
		}
	}
	return values
}

func bufferPrefixString(buffer ui.Buffer, cursor ui.Position) string {
	lineIndex := min(cursor.Line, len(buffer.Lines)-1)
	parts := make([]string, 0, lineIndex+1)
	for idx := 0; idx < lineIndex; idx++ {
		parts = append(parts, string(buffer.Lines[idx]))
	}
	if lineIndex >= 0 && lineIndex < len(buffer.Lines) {
		line := buffer.Lines[lineIndex]
		column := min(cursor.Column, len(line))
		parts = append(parts, string(line[:column]))
	}
	return strings.Join(parts, "\n")
}

func flagsForCommandPath(path []string) []string {
	switch {
	case slices.Equal(path, []string{"exec"}):
		return []string{"--preview", "--verbose", "--dry-run", "--validate"}
	case slices.Equal(path, []string{"show", "templates"}):
		return []string{"--tag"}
	default:
		return nil
	}
}

func newCompletionSuggestion(start int, end int, value string, text string, description string, category string) ui.Suggestion {
	return ui.Suggestion{
		Value:       value,
		Description: description,
		Category:    category,
		Result: ui.CompletionResult{
			Edits: []ui.CompletionEdit{{
				StartRune: start,
				EndRune:   end,
				Text:      text,
			}},
			Cursor: start + len([]rune(text)),
		},
	}
}

type operationNameEntry struct {
	Name        string
	Description string
}

func operationNames(resolver completionResolver, app *Application) []operationNameEntry {
	if app != nil {
		values, err := app.listOperationNames(context.Background())
		if err == nil {
			return values
		}
	}
	templates := resolverTemplates(resolver)
	results := make([]operationNameEntry, 0, len(templates))
	for _, value := range templates {
		results = append(results, operationNameEntry{Name: value, Description: "operation"})
	}
	return results
}

func applicationFromResolver(resolver completionResolver) *Application {
	if application, ok := resolver.(*Application); ok {
		return application
	}
	return nil
}
