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
		{name: "count alias tables", input: "count ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "count rows tables", input: "count rows ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "create subcommand", input: "create ", wantFirst: "database", wantCount: 2},
		{name: "list subcommand", input: "list ", wantFirst: "databases", wantCount: 2},
		{name: "peek alias tables", input: "peek ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "peek rows tables", input: "peek rows ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "sample alias tables", input: "sample ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "sample rows tables", input: "sample rows ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show subcommand", input: "show ", wantFirst: "columns", wantCount: 21},
		{name: "show partial subcommand", input: "show pro", wantFirst: "processes", wantCount: 2},
		{name: "show user alias subcommand", input: "show user ", wantFirst: "accounts", wantCount: 1},
		{name: "drop subcommand", input: "drop ", wantFirst: "database", wantCount: 2},
		{name: "connect saved", input: "connect ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 2},
		{name: "connection test saved", input: "connection test ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 3},
		{name: "connection test verbose", input: "connection test prod ", saved: []string{"prod", "dev"}, wantFirst: "verbose", wantCount: 1},
		{name: "connection test verbose prefix", input: "connection test prod v", saved: []string{"prod", "dev"}, wantFirst: "verbose", wantCount: 1},
		{name: "connection doctor saved", input: "connection doctor ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 2},
		{name: "use databases", input: "use ", databases: []string{"app_prod", "app_demo"}, wantFirst: "app_demo", wantCount: 2},
		{name: "drop user users", input: "drop user ", users: []string{"analytics-ro", "app_user"}, wantFirst: "analytics-ro", wantCount: 2},
		{name: "columns alias tables", input: "columns ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show columns tables", input: "show columns ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "describe tables", input: "describe ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show foreign keys tables", input: "show foreign keys ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show fks tables", input: "show fks ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show create table tables", input: "show create table ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show table status tables", input: "show table status ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show indexes tables", input: "show indexes ", wantFirst: "on", wantCount: 3, tables: []string{"users", "orders"}},
		{name: "show indexes on tables", input: "show indexes on ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "truncate table tables", input: "truncate table ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "rename table source tables", input: "rename table ", wantFirst: "orders", wantCount: 2, tables: []string{"users", "orders"}},
		{name: "show grants users", input: "show grants ", wantFirst: "analytics-ro", wantCount: 2, users: []string{"analytics-ro", "app_user"}},
		{name: "show variables suggestions", input: "show variables ", wantFirst: "max_connections", wantCount: 6},
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

	showCompletion := calculateCompletion("show ", CompletionContext{})
	if len(showCompletion.Suggestions) == 0 || showCompletion.Suggestions[0].Value != "columns" {
		t.Fatalf("show suggestions = %#v", showCompletion.Suggestions)
	}

	hintCompletion := calculateCompletion("crea", CompletionContext{})
	if hintCompletion.Hint == "" {
		t.Fatalf("expected inline hint, got empty")
	}
}

func TestCalculateCompletionPreservesArgumentPrefixes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       string
		ctx         CompletionContext
		wantValue   string
		wantFrom    int
		wantTo      int
		wantApplied string
	}{
		{
			name:        "use trailing space",
			input:       "use ",
			ctx:         CompletionContext{Databases: []string{"greenhn-dev"}},
			wantValue:   "greenhn-dev",
			wantFrom:    4,
			wantTo:      4,
			wantApplied: "use greenhn-dev",
		},
		{
			name:        "use partial prefix",
			input:       "use gre",
			ctx:         CompletionContext{Databases: []string{"greenhn-dev"}},
			wantValue:   "greenhn-dev",
			wantFrom:    4,
			wantTo:      7,
			wantApplied: "use greenhn-dev",
		},
		{
			name:        "connect trailing space",
			input:       "connect ",
			ctx:         CompletionContext{Connections: []CompletionConnection{{Name: "prod", Driver: "mysql", Mode: "ssh"}}},
			wantValue:   "prod",
			wantFrom:    8,
			wantTo:      8,
			wantApplied: "connect prod",
		},
		{
			name:        "connection show trailing space",
			input:       "connection show ",
			ctx:         CompletionContext{Connections: []CompletionConnection{{Name: "prod", Driver: "mysql", Mode: "ssh"}}},
			wantValue:   "prod",
			wantFrom:    16,
			wantTo:      16,
			wantApplied: "connection show prod",
		},
		{
			name:        "drop user trailing space",
			input:       "drop user ",
			ctx:         CompletionContext{Users: []string{"app_user"}},
			wantValue:   "app_user",
			wantFrom:    10,
			wantTo:      10,
			wantApplied: "drop user app_user",
		},
		{
			name:        "command prefix still replaces line token",
			input:       "connec",
			ctx:         CompletionContext{},
			wantValue:   "connect",
			wantFrom:    0,
			wantTo:      6,
			wantApplied: "connect",
		},
		{
			name:        "multi word subcommand replaces current token",
			input:       "connection ",
			ctx:         CompletionContext{},
			wantValue:   "create",
			wantFrom:    11,
			wantTo:      11,
			wantApplied: "connection create",
		},
		{
			name:        "count alias completion",
			input:       "count us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    6,
			wantTo:      8,
			wantApplied: "count users",
		},
		{
			name:        "count rows completion",
			input:       "count rows us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    11,
			wantTo:      13,
			wantApplied: "count rows users",
		},
		{
			name:        "columns alias completion",
			input:       "columns us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    8,
			wantTo:      10,
			wantApplied: "columns users",
		},
		{
			name:        "show columns completion",
			input:       "show columns us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    13,
			wantTo:      15,
			wantApplied: "show columns users",
		},
		{
			name:        "peek alias completion",
			input:       "peek us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    5,
			wantTo:      7,
			wantApplied: "peek users",
		},
		{
			name:        "peek rows completion",
			input:       "peek rows us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    10,
			wantTo:      12,
			wantApplied: "peek rows users",
		},
		{
			name:        "show indexes table completion",
			input:       "show indexes us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    13,
			wantTo:      15,
			wantApplied: "show indexes users",
		},
		{
			name:        "sample alias completion",
			input:       "sample us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    7,
			wantTo:      9,
			wantApplied: "sample users",
		},
		{
			name:        "sample rows completion",
			input:       "sample rows us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    12,
			wantTo:      14,
			wantApplied: "sample rows users",
		},
		{
			name:        "show foreign keys completion",
			input:       "show foreign keys us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    18,
			wantTo:      20,
			wantApplied: "show foreign keys users",
		},
		{
			name:        "show fks completion",
			input:       "show fks us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    9,
			wantTo:      11,
			wantApplied: "show fks users",
		},
		{
			name:        "show create table completion",
			input:       "show create table us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    18,
			wantTo:      20,
			wantApplied: "show create table users",
		},
		{
			name:        "show table status completion",
			input:       "show table status us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    18,
			wantTo:      20,
			wantApplied: "show table status users",
		},
		{
			name:        "truncate table completion",
			input:       "truncate table us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    15,
			wantTo:      17,
			wantApplied: "truncate table users",
		},
		{
			name:        "rename table source completion",
			input:       "rename table us",
			ctx:         CompletionContext{Tables: []string{"users"}},
			wantValue:   "users",
			wantFrom:    13,
			wantTo:      15,
			wantApplied: "rename table users",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			completion := calculateCompletion(tc.input, tc.ctx)
			if len(completion.Suggestions) == 0 {
				t.Fatalf("expected suggestions")
			}
			got := completion.Suggestions[0]
			if got.Value != tc.wantValue {
				t.Fatalf("value = %q, want %q", got.Value, tc.wantValue)
			}
			if got.ReplaceFrom != tc.wantFrom || got.ReplaceTo != tc.wantTo {
				t.Fatalf("replace range = [%d,%d], want [%d,%d]", got.ReplaceFrom, got.ReplaceTo, tc.wantFrom, tc.wantTo)
			}
			applied := tc.input[:got.ReplaceFrom] + got.Replacement + tc.input[got.ReplaceTo:]
			if applied != tc.wantApplied {
				t.Fatalf("applied = %q, want %q", applied, tc.wantApplied)
			}
		})
	}
}

func suggestionValues(completion ui.Completion) []string {
	values := make([]string, 0, len(completion.Suggestions))
	for _, suggestion := range completion.Suggestions {
		values = append(values, suggestion.Value)
	}
	return values
}
