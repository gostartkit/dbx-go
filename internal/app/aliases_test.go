package app

import "testing"

func TestResolveAlias(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"q":         "exit",
		"quit":      "exit",
		"conn":      "connect",
		"conn prod": "connect prod",
		"cx dev":    "connect dev",
		"conns":     "connections",
		"ls db":     "list databases",
		"show dbs":  "list databases",
		"create db": "create database",
		"drop db":   "drop database",
		"dry on":    "dry-run on",
		"dry off":   "dry-run off",
		"connect":   "connect",
	}

	for input, want := range cases {
		if got := resolveAlias(input); got != want {
			t.Fatalf("resolveAlias(%q) = %q, want %q", input, got, want)
		}
	}
}
