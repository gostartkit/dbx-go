package editor

import "testing"

func TestApplyCompletionPreservesSuffixAndCursor(t *testing.T) {
	t.Parallel()

	line, cursor := ApplyCompletion("show us where id = 1", CompletionResult{
		Edits: []CompletionEdit{{
			StartRune: 5,
			EndRune:   7,
			Text:      "users",
		}},
		Cursor: len([]rune("show users")),
	})
	if line != "show users where id = 1" {
		t.Fatalf("line = %q", line)
	}
	if cursor != len([]rune("show users")) {
		t.Fatalf("cursor = %d", cursor)
	}
}

func TestCommonSuggestionResultPreservesSuffix(t *testing.T) {
	t.Parallel()

	result, ok := CommonSuggestionResult("use da tail", []Suggestion{
		{Value: "database", Result: CompletionResult{Edits: []CompletionEdit{{StartRune: 4, EndRune: 6, Text: "database"}}, Cursor: len([]rune("use database"))}},
		{Value: "databases", Result: CompletionResult{Edits: []CompletionEdit{{StartRune: 4, EndRune: 6, Text: "databases"}}, Cursor: len([]rune("use databases"))}},
	})
	if !ok {
		t.Fatalf("expected common result")
	}
	line, cursor := ApplyCompletion("use da tail", result)
	if line != "use database tail" {
		t.Fatalf("line = %q", line)
	}
	if cursor != len([]rune("use database")) {
		t.Fatalf("cursor = %d", cursor)
	}
}
