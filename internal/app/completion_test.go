package app

import "testing"

func TestCalculateCompletionCommands(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		saved     []string
		wantFirst string
		wantCount int
	}{
		{name: "conn prefix", input: "conn", wantFirst: "connect", wantCount: 6},
		{name: "connection subcommands", input: "connection ", wantFirst: "create", wantCount: 4},
		{name: "create subcommand", input: "create ", wantFirst: "database", wantCount: 1},
		{name: "list subcommand", input: "list ", wantFirst: "databases", wantCount: 1},
		{name: "drop subcommand", input: "drop ", wantFirst: "database", wantCount: 1},
		{name: "connect saved", input: "connect ", saved: []string{"prod", "dev"}, wantFirst: "dev", wantCount: 2},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			completion := calculateCompletion(tc.input, tc.saved)
			if len(completion.Candidates) != tc.wantCount {
				t.Fatalf("candidate count = %d, want %d (%#v)", len(completion.Candidates), tc.wantCount, completion.Candidates)
			}
			if tc.wantCount > 0 && completion.Candidates[0] != tc.wantFirst {
				t.Fatalf("first candidate = %q, want %q", completion.Candidates[0], tc.wantFirst)
			}
		})
	}
}
