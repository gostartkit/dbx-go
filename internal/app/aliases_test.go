package app

import "testing"

func TestResolveAlias(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"q":                  "exit",
		"quit":               "exit",
		"conn":               "connect",
		"conn prod":          "connect prod",
		"cx dev":             "connect dev",
		"conns":              "connections",
		"ctx":                "context",
		"ls db":              "list databases",
		"show databases":     "list databases",
		"show dbs":           "list databases",
		"list users":         "show users",
		"show user accounts": "show users",
		"desc table users":   "describe table users",
		"create db":          "create database",
		"drop db":            "drop database",
		"test conn":          "connection test",
		"test conn prod":     "connection test prod",
		"doctor conn":        "connection doctor",
		"doctor conn prod":   "connection doctor prod",
		"dry on":             "dry-run on",
		"dry off":            "dry-run off",
		"connect":            "connect",
	}

	for input, want := range cases {
		if got := resolveAlias(input); got != want {
			t.Fatalf("resolveAlias(%q) = %q, want %q", input, got, want)
		}
	}
}
