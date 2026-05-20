package app

import (
	"testing"

	"pkg.gostartkit.com/dbx/internal/ui"
)

func TestCalculateCompletionRootCommands(t *testing.T) {
	t.Parallel()

	values := suggestionValues(calculateCompletion("", CompletionContext{}))
	assertSuggestionsContainAll(t, values, []string{
		"connect",
		"create",
		"describe",
		"doctor",
		"drop",
		"run",
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

	useCompletion := calculateCompletion("use database ", CompletionContext{
		Databases: []string{"app_demo", "app_prod"},
	})
	assertSuggestionsContainAll(t, suggestionValues(useCompletion), []string{"app_demo", "app_prod"})

	rowCompletion := calculateCompletion("show rows ", CompletionContext{
		Tables: []string{"users", "orders"},
	})
	assertSuggestionsContainAll(t, suggestionValues(rowCompletion), []string{"orders", "users"})

	templateCompletion := calculateCompletion("run template ", CompletionContext{
		Templates: []string{"readonly_user", "create_database_with_user"},
	})
	assertSuggestionsContainAll(t, suggestionValues(templateCompletion), []string{"create_database_with_user", "readonly_user"})
}

func TestCalculateCompletionHelpTopics(t *testing.T) {
	t.Parallel()

	values := suggestionValues(calculateCompletion("help ", CompletionContext{}))
	assertSuggestionsContainAll(t, values, []string{"doctor connection", "show templates", "run template", "show rows"})
}

func TestCalculateCompletionOmitsRemovedCommands(t *testing.T) {
	t.Parallel()

	values := suggestionValues(calculateCompletion("", CompletionContext{}))
	assertSuggestionsMissingAll(t, values, []string{"count", "peek", "sample", "truncate", "rename", "validate", "edit", "test", "context", "clear"})
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
	if got.ReplaceFrom != 8 || got.ReplaceTo != 10 {
		t.Fatalf("replacement range = %d:%d", got.ReplaceFrom, got.ReplaceTo)
	}
	if got.Replacement != "prod" {
		t.Fatalf("replacement = %q", got.Replacement)
	}

	hint := calculateCompletion("con", CompletionContext{}).Hint
	if hint == "" {
		t.Fatalf("expected inline hint")
	}
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
