package app

import "testing"

func TestCalculateCompletionCommands(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		saved     []string
		databases []string
		wantFirst string
		wantCount int
	}{
		{name: "conn prefix", input: "conn", wantFirst: "connect", wantCount: 8},
		{name: "connection subcommands", input: "connection ", wantFirst: "create", wantCount: 6},
		{name: "create subcommand", input: "create ", wantFirst: "database", wantCount: 1},
		{name: "list subcommand", input: "list ", wantFirst: "databases", wantCount: 1},
		{name: "show subcommand", input: "show ", wantFirst: "databases", wantCount: 2},
		{name: "drop subcommand", input: "drop ", wantFirst: "database", wantCount: 1},
		{name: "connect saved", input: "connect ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 2},
		{name: "connection test saved", input: "connection test ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 3},
		{name: "connection test verbose", input: "connection test prod ", saved: []string{"prod", "dev"}, wantFirst: "verbose", wantCount: 1},
		{name: "connection test verbose prefix", input: "connection test prod v", saved: []string{"prod", "dev"}, wantFirst: "verbose", wantCount: 1},
		{name: "connection doctor saved", input: "connection doctor ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 2},
		{name: "use databases", input: "use ", databases: []string{"app_prod", "app_demo"}, wantFirst: "app_demo", wantCount: 2},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			completion := calculateCompletion(tc.input, tc.saved, tc.databases)
			if len(completion.Candidates) != tc.wantCount {
				t.Fatalf("candidate count = %d, want %d (%#v)", len(completion.Candidates), tc.wantCount, completion.Candidates)
			}
			if tc.wantCount > 0 && completion.Candidates[0] != tc.wantFirst {
				t.Fatalf("first candidate = %q, want %q", completion.Candidates[0], tc.wantFirst)
			}
		})
	}
}
