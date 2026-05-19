package template

import (
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestBuildPlanRendersVariablesAndEscapesStrings(t *testing.T) {
	t.Parallel()

	cfg := &config.ConnectionConfig{
		Name:   "prod",
		Driver: "mysql",
		Mode:   "direct",
		Host:   "127.0.0.1",
		Port:   3306,
		User:   "root",
	}

	tpl := &Template{
		Name:   "create_database_with_user",
		Layer:  "global",
		Source: "test",
		Actions: []Action{
			{
				Type:        "sql",
				Description: "Create database {{database}} for {{connection.name}}",
				SQL:         "CREATE USER '{{database}}'@'%' IDENTIFIED BY '{{password}}'",
			},
		},
	}

	plan, err := BuildPlan(tpl, cfg, map[string]string{
		"database": "appdb",
		"password": "pa'ss",
	})
	if err != nil {
		t.Fatalf("BuildPlan returned error: %v", err)
	}

	if got := plan.Actions[0].Description; got != "Create database appdb for prod" {
		t.Fatalf("description = %q", got)
	}
	if got := plan.Actions[0].SQL; got != "CREATE USER 'appdb'@'%' IDENTIFIED BY 'pa''ss'" {
		t.Fatalf("sql = %q", got)
	}
}

func TestBuiltinMySQLSQLGeneration(t *testing.T) {
	t.Parallel()

	cfg := &config.ConnectionConfig{
		Name:   "dev",
		Driver: "mysql",
		Mode:   "direct",
		Host:   "127.0.0.1",
		Port:   3306,
		User:   "root",
	}

	cases := []struct {
		name    string
		command string
		values  map[string]string
		wantSQL string
	}{
		{
			name:    "create database",
			command: "create database",
			values: map[string]string{
				"database":  "greenhn-dev",
				"charset":   "utf8mb4",
				"collation": "utf8mb4_unicode_ci",
			},
			wantSQL: "CREATE DATABASE IF NOT EXISTS `greenhn-dev` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		},
		{
			name:    "list databases",
			command: "list databases",
			values:  map[string]string{},
			wantSQL: "SHOW DATABASES",
		},
		{
			name:    "drop database",
			command: "drop database",
			values: map[string]string{
				"database": "greenhn-dev",
			},
			wantSQL: "DROP DATABASE IF EXISTS `greenhn-dev`",
		},
		{
			name:    "show users",
			command: "show users",
			values:  map[string]string{},
			wantSQL: "SELECT CONCAT(User, '@', Host) FROM mysql.user ORDER BY User, Host",
		},
		{
			name:    "drop user",
			command: "drop user",
			values: map[string]string{
				"username":  "analytics-ro",
				"user_host": "%",
			},
			wantSQL: "DROP USER 'analytics-ro'@'%'",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var selected *Template
			for _, builtin := range Builtins() {
				if builtin.Match.Command == tc.command {
					builtinCopy := builtin
					selected = &builtinCopy
					break
				}
			}
			if selected == nil {
				t.Fatalf("builtin template for %q not found", tc.command)
			}

			plan, err := BuildPlan(selected, cfg, tc.values)
			if err != nil {
				t.Fatalf("BuildPlan returned error: %v", err)
			}

			if got := plan.Actions[0].SQL; got != tc.wantSQL {
				t.Fatalf("SQL = %q, want %q", got, tc.wantSQL)
			}
		})
	}
}

func TestBuiltinCreateUserSQLGeneration(t *testing.T) {
	t.Parallel()

	cfg := &config.ConnectionConfig{
		Name:   "dev",
		Driver: "mysql",
		Mode:   "direct",
		Host:   "127.0.0.1",
		Port:   3306,
		User:   "root",
	}

	var selected *Template
	for _, builtin := range Builtins() {
		if builtin.Match.Command == "create user" {
			builtinCopy := builtin
			selected = &builtinCopy
			break
		}
	}
	if selected == nil {
		t.Fatal("builtin create user template not found")
	}

	plan, err := BuildPlan(selected, cfg, map[string]string{
		"username":          "analytics-ro",
		"user_host":         "%",
		"password":          "S3cretPass",
		"grant_description": "Grant SELECT on `greenhn-dev`.*",
		"grant_sql":         "GRANT SELECT ON `greenhn-dev`.* TO 'analytics-ro'@'%'",
		"grant_database":    "greenhn-dev",
	})
	if err != nil {
		t.Fatalf("BuildPlan returned error: %v", err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("actions = %d, want 2", len(plan.Actions))
	}
	if got := plan.Actions[0].SQL; got != "CREATE USER 'analytics-ro'@'%' IDENTIFIED BY 'S3cretPass'" {
		t.Fatalf("create user SQL = %q", got)
	}
	if got := plan.Actions[1].SQL; got != "GRANT SELECT ON `greenhn-dev`.* TO 'analytics-ro'@'%'" {
		t.Fatalf("grant SQL = %q", got)
	}
}
