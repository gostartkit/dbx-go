package app

import (
	"testing"

	"pkg.gostartkit.com/dbx/internal/ui"
)

func TestCalculateCompletionCommands(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		saved     []string
		databases []string
		tables    []string
		users     []string
		wantFirst string
		wantCount int
	}{
		{name: "conn prefix", input: "conn", wantFirst: "connect", wantCount: 8},
		{name: "connection subcommands", input: "connection ", wantFirst: "create", wantCount: 6},
		{name: "create subcommand", input: "create ", wantFirst: "database", wantCount: 2},
		{name: "list subcommand", input: "list ", wantFirst: "databases", wantCount: 2},
		{name: "show subcommand", input: "show ", wantFirst: "databases", wantCount: 6},
		{name: "show user alias subcommand", input: "show user ", wantFirst: "accounts", wantCount: 1},
		{name: "drop subcommand", input: "drop ", wantFirst: "database", wantCount: 2},
		{name: "connect saved", input: "connect ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 2},
		{name: "connection test saved", input: "connection test ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 3},
		{name: "connection test verbose", input: "connection test prod ", saved: []string{"prod", "dev"}, wantFirst: "verbose", wantCount: 1},
		{name: "connection test verbose prefix", input: "connection test prod v", saved: []string{"prod", "dev"}, wantFirst: "verbose", wantCount: 1},
		{name: "connection doctor saved", input: "connection doctor ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 2},
		{name: "use databases", input: "use ", databases: []string{"app_prod", "app_demo"}, wantFirst: "app_demo", wantCount: 2},
		{name: "drop user users", input: "drop user ", users: []string{"analytics-ro", "app_user"}, wantFirst: "analytics-ro", wantCount: 2},
		{name: "describe tables", input: "describe ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show grants users", input: "show grants ", wantFirst: "analytics-ro", wantCount: 2, users: []string{"analytics-ro", "app_user"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := CompletionContext{
				Connections: make([]CompletionConnection, 0, len(tc.saved)),
				Databases:   tc.databases,
				Tables:      tc.tables,
				Users:       tc.users,
			}
			for _, name := range tc.saved {
				ctx.Connections = append(ctx.Connections, CompletionConnection{Name: name, Driver: "mysql", Mode: "direct"})
			}

			completion := calculateCompletion(tc.input, ctx)
			values := suggestionValues(completion)
			if len(values) != tc.wantCount {
				t.Fatalf("candidate count = %d, want %d (%#v)", len(values), tc.wantCount, values)
			}
			if tc.wantCount > 0 && values[0] != tc.wantFirst {
				t.Fatalf("first candidate = %q, want %q", values[0], tc.wantFirst)
			}
		})
	}
}

func TestCalculateCompletionIncludesConnectionDescriptionsAndHint(t *testing.T) {
	t.Parallel()

	completion := calculateCompletion("connect ", CompletionContext{
		Connections: []CompletionConnection{
			{Name: "prod", Driver: "mysql", Mode: "proxy-ssh"},
		},
	})
	if len(completion.Suggestions) != 1 {
		t.Fatalf("suggestions = %#v", completion.Suggestions)
	}
	if completion.Suggestions[0].Description != "mysql proxy-ssh" {
		t.Fatalf("description = %q", completion.Suggestions[0].Description)
	}

	hintCompletion := calculateCompletion("crea", CompletionContext{})
	if hintCompletion.Hint == "" {
		t.Fatalf("expected inline hint, got empty")
	}
}

func suggestionValues(completion ui.Completion) []string {
	values := make([]string, 0, len(completion.Suggestions))
	for _, suggestion := range completion.Suggestions {
		values = append(values, suggestion.Value)
	}
	return values
}
