package app

import "testing"

func TestResolveAlias(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"q":                      "exit",
		"quit":                   "exit",
		"conn":                   "connect",
		"conn prod":              "connect prod",
		"cx dev":                 "connect dev",
		"conns":                  "connections",
		"count users":            "count rows users",
		"columns users":          "show columns users",
		"ctx":                    "context",
		"ls db":                  "show databases",
		"list databases":         "show databases",
		"show databases":         "show databases",
		"show dbs":               "show databases",
		"templates":              "show templates",
		"templates tag readonly": "show templates tag readonly",
		"templates database":     "show templates database",
		"show fks orders":        "show foreign keys orders",
		"show index":             "show indexes",
		"show index users":       "show indexes users",
		"show processes":         "show processlist",
		"show trigger":           "show triggers",
		"show vars":              "show variables",
		"show vars innodb%":      "show variables innodb%",
		"show view":              "show views",
		"template show create_database_with_user":     "describe template create_database_with_user",
		"template describe create_database_with_user": "describe template create_database_with_user",
		"run template create_database_with_user":      "template run create_database_with_user",
		"list users":                                  "show users",
		"show user accounts":                          "show users",
		"peek users":                                  "peek rows users",
		"sample users":                                "sample rows users",
		"desc table users":                            "describe table users",
		"create db":                                   "create database",
		"drop db":                                     "drop database",
		"test conn":                                   "connection test",
		"test conn prod":                              "connection test prod",
		"doctor conn":                                 "connection doctor",
		"doctor conn prod":                            "connection doctor prod",
		"dry on":                                      "dry-run on",
		"dry off":                                     "dry-run off",
		"connect":                                     "connect",
	}

	for input, want := range cases {
		if got := resolveAlias(input); got != want {
			t.Fatalf("resolveAlias(%q) = %q, want %q", input, got, want)
		}
	}
}
