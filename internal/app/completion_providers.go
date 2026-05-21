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
	program         *commandlang.Program
	localProgram    *commandlang.Program
	syntaxContext   commandlang.SyntaxContext
	localSyntax     commandlang.SyntaxContext
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
	registry        *commandlang.Registry
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
	fullPrefix := logicalCompletionPrefix(request.Buffer, request.Cursor)
	localPrefix := request.CurrentLinePrefix()
	registry := commandlang.DefaultRegistry()
	fullTokens := commandlang.Lex(fullPrefix)
	localTokens := commandlang.Lex(localPrefix)
	fullContext := commandlang.BuildCommandContext(fullTokens, len([]rune(fullPrefix)))
	localContext := commandlang.BuildCommandContext(localTokens, len([]rune(localPrefix)))
	fullProgram := commandlang.ParseTokens(fullTokens)
	localProgram := commandlang.ParseTokens(localTokens)
	fullSyntax := commandlang.BuildSyntaxContextWithRegistry(fullProgram, len([]rune(fullPrefix)), registry)
	localSyntax := commandlang.BuildSyntaxContextWithRegistry(localProgram, len([]rune(localPrefix)), registry)
	replaceStart, replaceEnd, currentValue, currentFlag := completionEditRangeFromSyntax(localSyntax, len([]rune(localPrefix)))
	if replaceStart == replaceEnd && currentValue == "" && currentFlag == "" {
		replaceStart, replaceEnd, currentValue, currentFlag = completionEditRange(localContext, len([]rune(localPrefix)))
	}
	commandPath := append([]string(nil), fullSyntax.CommandPath...)
	parentPath := append([]string(nil), fullSyntax.ParentPath...)
	positionalIndex := fullSyntax.ArgIndex
	expectingFlag := ""
	if fullSyntax.InFlagValue {
		expectingFlag = fullSyntax.CurrentFlag
	} else {
		expectingFlag = fullContext.ExpectingValueForFlag
	}

	return &providerContext{
		request:         request,
		fullPrefix:      fullPrefix,
		localPrefix:     localPrefix,
		fullTokens:      fullTokens,
		localTokens:     localTokens,
		commandContext:  fullContext,
		localContext:    localContext,
		program:         fullProgram,
		localProgram:    localProgram,
		syntaxContext:   fullSyntax,
		localSyntax:     localSyntax,
		commandPath:     commandPath,
		parentCommand:   parentPath,
		positionalIndex: positionalIndex,
		currentValue:    currentValue,
		currentFlag:     currentFlag,
		expectingFlag:   expectingFlag,
		replaceStart:    replaceStart,
		replaceEnd:      replaceEnd,
		resolver:        resolver,
		app:             app,
		application:     applicationFromResolver(resolver),
		registry:        registry,
	}
}

func (p commandProvider) Complete(ctx *providerContext) ([]ui.Suggestion, error) {
	if schemaCompletionRoute(ctx) == "topic" {
		suggestions := make([]ui.Suggestion, 0)
		for _, suggestion := range helpCompletionTopics() {
			if ctx.currentValue != "" && !strings.HasPrefix(suggestion.Value, ctx.currentValue) {
				continue
			}
			suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, suggestion.Value, suggestion.Value, suggestion.Description, suggestion.Category))
		}
		return suggestions, nil
	}
	if !ctx.syntaxContext.InCommandName && !ctx.syntaxContext.InSubcommand {
		if ctx.syntaxContext.CommandSpec == nil && len(ctx.commandPath) > 0 {
			children := directChildCommands(ctx.commandPath)
			if len(children) > 0 {
				suggestions := make([]ui.Suggestion, 0, len(children))
				for _, child := range children {
					if ctx.currentValue != "" && !strings.HasPrefix(child.Value, ctx.currentValue) {
						continue
					}
					suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, child.Value, child.Value, child.Description, "command"))
				}
				return suggestions, nil
			}
		}
		return nil, nil
	}

	parent := ctx.parentCommand
	if ctx.syntaxContext.InCommandName {
		parent = nil
	}
	children := mergeCommandSuggestions(schemaChildCommands(ctx.registry, parent), directChildCommands(parent))
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
	if schemaCompletionRoute(ctx) != "operation" && (!slices.Equal(ctx.commandPath, []string{"exec"}) || !ctx.syntaxContext.InArg || ctx.positionalIndex != 0) {
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
	if schemaCompletionRoute(ctx) != "template" {
		switch {
		case slices.Equal(ctx.commandPath, []string{"describe", "template"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		case slices.Equal(ctx.commandPath, []string{"template", "validate"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		case slices.Equal(ctx.commandPath, []string{"template", "render"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		default:
			return nil, nil
		}
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
	if schemaCompletionRoute(ctx) != "connection" {
		switch {
		case slices.Equal(ctx.commandPath, []string{"connect"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		case slices.Equal(ctx.commandPath, []string{"show", "connection"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		case slices.Equal(ctx.commandPath, []string{"drop", "connection"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		default:
			return nil, nil
		}
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
	if schemaCompletionRoute(ctx) != "database" {
		switch {
		case slices.Equal(ctx.commandPath, []string{"use"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		default:
			return nil, nil
		}
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
	if schemaCompletionRoute(ctx) != "table" {
		switch {
		case slices.Equal(ctx.commandPath, []string{"show", "table"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		case slices.Equal(ctx.commandPath, []string{"show", "columns"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		case slices.Equal(ctx.commandPath, []string{"show", "rows"}) && ctx.syntaxContext.InArg && ctx.positionalIndex == 0:
		default:
			return nil, nil
		}
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
	if schemaCompletionRoute(ctx) != "user" && (!slices.Equal(ctx.commandPath, []string{"drop", "user"}) || !ctx.syntaxContext.InArg || ctx.positionalIndex != 0) {
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
	flags := schemaFlagNames(ctx.syntaxContext.CommandSpec)
	if len(flags) == 0 {
		flags = flagsForCommandPath(ctx.commandPath)
	}
	if len(flags) == 0 {
		return nil, nil
	}
	if ctx.expectingFlag != "" || ctx.syntaxContext.InFlagValue {
		return nil, nil
	}
	if !ctx.syntaxContext.InFlagName {
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
	if ctx.expectingFlag == "" || !ctx.syntaxContext.InFlagValue {
		return nil, nil
	}
	if ctx.syntaxContext.FlagSpec != nil && len(ctx.syntaxContext.FlagSpec.EnumValues) > 0 {
		suggestions := make([]ui.Suggestion, 0, len(ctx.syntaxContext.FlagSpec.EnumValues))
		for _, value := range ctx.syntaxContext.FlagSpec.EnumValues {
			if ctx.currentValue != "" && !strings.HasPrefix(value, ctx.currentValue) {
				continue
			}
			suggestions = append(suggestions, newCompletionSuggestion(ctx.replaceStart, ctx.replaceEnd, value, value, "enum", "enum"))
		}
		return suggestions, nil
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

func schemaChildCommands(registry *commandlang.Registry, parent []string) []Suggestion {
	if registry == nil {
		return nil
	}
	children := registry.LookupVisibleSubcommands(parent)
	suggestions := make([]Suggestion, 0, len(children))
	for _, child := range children {
		suggestions = append(suggestions, Suggestion{
			Value:       child.Name,
			Description: child.Description,
			Category:    "command",
		})
	}
	return suggestions
}

func mergeCommandSuggestions(left []Suggestion, right []Suggestion) []Suggestion {
	merged := make([]Suggestion, 0, len(left)+len(right))
	seen := make(map[string]struct{}, len(left)+len(right))
	for _, candidate := range append(append([]Suggestion(nil), left...), right...) {
		if _, ok := seen[candidate.Value]; ok {
			continue
		}
		seen[candidate.Value] = struct{}{}
		merged = append(merged, candidate)
	}
	return merged
}

func schemaFlagNames(spec *commandlang.CommandSpec) []string {
	if spec == nil {
		return nil
	}
	values := make([]string, 0, len(spec.Flags))
	for _, flag := range spec.Flags {
		values = append(values, flag.Name)
	}
	return values
}

func schemaCompletionRoute(ctx *providerContext) string {
	if ctx == nil {
		return ""
	}
	if ctx.syntaxContext.CompletionProvider != "" {
		return ctx.syntaxContext.CompletionProvider
	}
	switch ctx.syntaxContext.ExpectedValueType {
	case commandlang.ValueOperation:
		return "operation"
	case commandlang.ValueTemplate:
		return "template"
	case commandlang.ValueConnection:
		return "connection"
	case commandlang.ValueDatabase:
		return "database"
	case commandlang.ValueTable:
		return "table"
	case commandlang.ValueUser:
		return "user"
	case commandlang.ValueSchema:
		return "schema"
	}
	return ""
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

func completionEditRangeFromSyntax(ctx commandlang.SyntaxContext, cursor int) (int, int, string, string) {
	if ctx.InFlagValue {
		switch node := ctx.Node.(type) {
		case *commandlang.ArgNode:
			return node.Range().StartRune, node.Range().EndRune, node.Value, ctx.CurrentFlag
		default:
			return cursor, cursor, "", ctx.CurrentFlag
		}
	}
	if ctx.InFlagName {
		if node, ok := ctx.Node.(*commandlang.FlagNode); ok {
			return node.Range().StartRune, node.Range().EndRune, node.Name, node.Name
		}
	}
	switch node := ctx.Node.(type) {
	case *commandlang.ArgNode:
		return node.Range().StartRune, node.Range().EndRune, node.Value, ctx.CurrentFlag
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

func logicalCompletionPrefix(buffer ui.Buffer, cursor ui.Position) string {
	lineIndex := min(cursor.Line, len(buffer.Lines)-1)
	parts := make([]string, 0, lineIndex+1)
	for idx := 0; idx < lineIndex; idx++ {
		line := strings.TrimRight(string(buffer.Lines[idx]), " \t\r")
		if strings.HasSuffix(line, "\\") {
			line = strings.TrimSpace(strings.TrimSuffix(line, "\\"))
		} else {
			line = strings.TrimSpace(line)
		}
		if line != "" {
			parts = append(parts, line)
		}
	}
	current := ""
	if lineIndex >= 0 && lineIndex < len(buffer.Lines) {
		line := buffer.Lines[lineIndex]
		column := min(cursor.Column, len(line))
		current = string(line[:column])
		current = strings.TrimLeft(current, " \t\r")
	}
	if len(parts) == 0 {
		return current
	}
	if current == "" {
		return strings.Join(parts, " ") + " "
	}
	return strings.Join(append(parts, current), " ")
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
