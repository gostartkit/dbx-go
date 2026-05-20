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
		"columns users":      "show columns users",
		"ctx":                "context",
		"ls db":              "show databases",
		"list databases":     "show databases",
		"show databases":     "show databases",
		"show dbs":           "show databases",
		"show fks orders":    "show foreign keys orders",
		"show index":         "show indexes",
		"show index users":   "show indexes users",
		"show processes":     "show processlist",
		"show trigger":       "show triggers",
		"show vars":          "show variables",
		"show vars innodb%":  "show variables innodb%",
		"show view":          "show views",
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
