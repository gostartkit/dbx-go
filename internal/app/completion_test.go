package app

import (
	"testing"

	"pkg.gostartkit.com/dbx/internal/commandlang"
	"pkg.gostartkit.com/dbx/internal/ui"
)

func TestCalculateCompletionRootCommands(t *testing.T) {
	t.Parallel()

	values := suggestionValues(calculateCompletion("", CompletionContext{}))
	assertSuggestionsContainAll(t, values, []string{
		"connect",
		"create",
		"doctor",
		"drop",
		"exec",
		"help",
		"show",
		"use",
		"audit",
		"exit",
	})
}

func TestCalculateCompletionDynamicValues(t *testing.T) {
	t.Parallel()

	connectCompletion := calculateCompletion("connect ", CompletionContext{
		Connections: []CompletionConnection{
			{Name: "prod", Driver: "mysql", Mode: "proxy-ssh"},
			{Name: "dev", Driver: "mysql", Mode: "direct"},
		},
	})
	if len(connectCompletion.Suggestions) < 2 {
		t.Fatalf("connect suggestions = %#v", connectCompletion.Suggestions)
	}
	if connectCompletion.Suggestions[0].Value != "dev" {
		t.Fatalf("first connect suggestion = %q", connectCompletion.Suggestions[0].Value)
	}
	if connectCompletion.Suggestions[1].Description == "" {
		t.Fatalf("expected connection description, got %#v", connectCompletion.Suggestions[1])
	}

	useCompletion := calculateCompletion("use ", CompletionContext{
		Databases: []string{"app_demo", "app_prod"},
	})
	assertSuggestionsContainAll(t, suggestionValues(useCompletion), []string{"app_demo", "app_prod"})

	rowCompletion := calculateCompletion("show rows ", CompletionContext{
		Tables: []string{"users", "orders"},
	})
	assertSuggestionsContainAll(t, suggestionValues(rowCompletion), []string{"orders", "users"})

	templateCompletion := calculateCompletion("exec ", CompletionContext{
		Templates: []string{"readonly_user", "create_database_with_user"},
	})
	assertSuggestionsContainAll(t, suggestionValues(templateCompletion), []string{"create_database_with_user", "readonly_user"})

	createCompletion := calculateCompletion("create ", CompletionContext{})
	assertSuggestionsContainAll(t, suggestionValues(createCompletion), []string{"connection", "database", "user"})

	dropCompletion := calculateCompletion("drop ", CompletionContext{})
	assertSuggestionsContainAll(t, suggestionValues(dropCompletion), []string{"connection", "database", "user"})
}

func TestCalculateCompletionHelpTopics(t *testing.T) {
	t.Parallel()

	values := suggestionValues(calculateCompletion("help ", CompletionContext{}))
	assertSuggestionsContainAll(t, values, []string{"doctor", "show templates", "exec", "show rows", "show users"})
	assertSuggestionsMissingAll(t, values, []string{"run"})
	assertSuggestionsMissingAll(t, values, []string{"run template"})
	assertSuggestionsMissingAll(t, values, []string{"show user"})
	assertSuggestionsMissingAll(t, values, []string{"describe", "show template", "doctor connection"})
}

func TestCalculateCompletionOmitsRemovedCommands(t *testing.T) {
	t.Parallel()

	values := suggestionValues(calculateCompletion("", CompletionContext{}))
	assertSuggestionsMissingAll(t, values, []string{"count", "peek", "sample", "truncate", "rename", "validate", "edit", "test", "context", "clear", "describe"})
}

func TestCalculateCompletionOmitsRemovedSubcommands(t *testing.T) {
	t.Parallel()

	showValues := suggestionValues(calculateCompletion("show ", CompletionContext{}))
	assertSuggestionsContainAll(t, showValues, []string{"users"})
	assertSuggestionsMissingAll(t, showValues, []string{"user"})
	assertSuggestionsMissingAll(t, showValues, []string{"template"})

	runValues := suggestionValues(calculateCompletion("exec ", CompletionContext{
		Templates: []string{"readonly_user", "create_database_with_user"},
	}))
	assertSuggestionsContainAll(t, runValues, []string{"readonly_user", "create_database_with_user"})
	assertSuggestionsMissingAll(t, runValues, []string{"template", "sql"})

	doctorValues := suggestionValues(calculateCompletion("doctor ", CompletionContext{}))
	assertSuggestionsMissingAll(t, doctorValues, []string{"connection"})
}

func TestCalculateCompletionReplacementRanges(t *testing.T) {
	t.Parallel()

	completion := calculateCompletion("connect pr", CompletionContext{
		Connections: []CompletionConnection{{Name: "prod", Driver: "mysql", Mode: "ssh"}},
	})
	if len(completion.Suggestions) == 0 {
		t.Fatalf("expected connect suggestions")
	}
	got := completion.Suggestions[0]
	if got.Value != "prod" {
		t.Fatalf("value = %q, want prod", got.Value)
	}
	if len(got.Result.Edits) != 1 {
		t.Fatalf("completion edits = %#v", got.Result.Edits)
	}
	edit := got.Result.Edits[0]
	if edit.StartRune != 8 || edit.EndRune != 10 {
		t.Fatalf("replacement range = %d:%d", edit.StartRune, edit.EndRune)
	}
	if edit.Text != "prod" {
		t.Fatalf("replacement = %q", edit.Text)
	}

	hint := calculateCompletion("con", CompletionContext{}).Hint
	if hint == "" {
		t.Fatalf("expected inline hint")
	}
}

func TestCalculateCompletionASTFlagAndValueContexts(t *testing.T) {
	t.Parallel()

	flagCompletion := calculateCompletion("exec create_user --dr", CompletionContext{})
	assertSuggestionsContainAll(t, suggestionValues(flagCompletion), []string{"--dry-run"})

	tagCompletion := calculateCompletion("show templates --tag ", CompletionContext{
		TemplateTags: []string{"readonly", "grant"},
	})
	assertSuggestionsContainAll(t, suggestionValues(tagCompletion), []string{"readonly", "grant"})
}

func TestCalculateCompletionASTSubcommandSuggestions(t *testing.T) {
	t.Parallel()

	showCompletion := calculateCompletion("show ", CompletionContext{})
	assertSuggestionsContainAll(t, suggestionValues(showCompletion), []string{"columns", "connections", "context", "rows", "tables", "templates", "users"})
}

func TestBuildProviderContextFlagValueSyntax(t *testing.T) {
	t.Parallel()

	app := newREPLBuilder(nil, nil).buildApp()
	request := ui.NewSingleLineCompletionRequest("show templates --tag ", len([]rune("show templates --tag ")))
	ctx := buildProviderContext(app, request, staticCompletionResolver{
		ctx: CompletionContext{TemplateTags: []string{"readonly"}},
	})
	if !ctx.syntaxContext.InFlagValue {
		t.Fatalf("expected flag value context, got %+v", ctx.syntaxContext)
	}
	if ctx.expectingFlag != "--tag" {
		t.Fatalf("expecting flag = %q", ctx.expectingFlag)
	}
	suggestions, err := flagValueProvider{}.Complete(ctx)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	assertSuggestionsContainAll(t, suggestionValues(ui.Completion{Suggestions: suggestions}), []string{"readonly"})
}

func TestFlagCompletionUsesSchemaFlags(t *testing.T) {
	t.Parallel()

	completion := calculateCompletion("exec --dr", CompletionContext{})
	assertSuggestionsContainAll(t, suggestionValues(completion), []string{"--dry-run"})
}

func TestEnumFlagValueCompletionUsesSchemaEnumValues(t *testing.T) {
	t.Parallel()

	ctx := &providerContext{
		syntaxContext: commandlang.SyntaxContext{
			InFlagValue: true,
			FlagSpec: &commandlang.FlagSpec{
				Name:       "--mode",
				ValueType:  commandlang.ValueEnum,
				EnumValues: []string{"readonly", "admin"},
			},
		},
		expectingFlag: "--mode",
		replaceStart:  0,
		replaceEnd:    0,
	}

	suggestions, err := flagValueProvider{}.Complete(ctx)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	assertSuggestionsContainAll(t, suggestionValues(ui.Completion{Suggestions: suggestions}), []string{"readonly", "admin"})
}

func assertSuggestionsMissingAll(t *testing.T, values []string, missing []string) {
	t.Helper()
	have := make(map[string]struct{}, len(values))
	for _, value := range values {
		have[value] = struct{}{}
	}
	for _, value := range missing {
		if _, ok := have[value]; ok {
			t.Fatalf("unexpected suggestion %q in %#v", value, values)
		}
	}
}

func assertSuggestionsContainAll(t *testing.T, values []string, want []string) {
	t.Helper()
	have := make(map[string]struct{}, len(values))
	for _, value := range values {
		have[value] = struct{}{}
	}
	for _, value := range want {
		if _, ok := have[value]; !ok {
			t.Fatalf("missing suggestion %q in %#v", value, values)
		}
	}
}

func suggestionValues(completion ui.Completion) []string {
	values := make([]string, 0, len(completion.Suggestions))
	for _, suggestion := range completion.Suggestions {
		values = append(values, suggestion.Value)
	}
	return values
}
