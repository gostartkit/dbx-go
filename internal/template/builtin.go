package template

func Builtins() []Template {
	return []Template{
		{
			Version: 1,
			Name:    "builtin_create_database",
			Match: Match{
				Command: "create database",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Create database `{{database}}` with charset {{charset}} and collation {{collation}}",
					SQL:         "CREATE DATABASE IF NOT EXISTS `{{database}}` CHARACTER SET {{charset}} COLLATE {{collation}}",
				},
			},
			Layer:  "builtin",
			Source: "builtin:create database",
		},
		{
			Version: 1,
			Name:    "builtin_list_databases",
			Match: Match{
				Command: "show databases",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "List databases on connection {{connection.name}}",
					SQL:         "SHOW DATABASES",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show databases",
		},
		{
			Version: 1,
			Name:    "builtin_drop_database",
			Match: Match{
				Command: "drop database",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Drop database `{{database}}` if it exists",
					SQL:         "DROP DATABASE IF EXISTS `{{database}}`",
				},
			},
			Layer:  "builtin",
			Source: "builtin:drop database",
		},
		{
			Version: 1,
			Name:    "builtin_create_user",
			Match: Match{
				Command: "create user",
				Driver:  "mysql",
			},
			Inputs: []Input{
				{
					Name:   "password",
					Type:   "secret",
					Prompt: "Password",
				},
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Create MySQL user '{{username}}'@'{{user_host}}'",
					SQL:         "CREATE USER '{{username}}'@'{{user_host}}' IDENTIFIED BY '{{password}}'",
				},
				{
					Type:        "sql",
					Description: "{{grant_description}}",
					SQL:         "{{grant_sql}}",
				},
			},
			Layer:  "builtin",
			Source: "builtin:create user",
		},
		{
			Version: 1,
			Name:    "builtin_show_users",
			Match: Match{
				Command: "show users",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "List MySQL user accounts on connection {{connection.name}}",
					SQL:         "SELECT CONCAT(User, '@', Host) FROM mysql.user ORDER BY User, Host",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show users",
		},
		{
			Version: 1,
			Name:    "builtin_show_tables",
			Match: Match{
				Command: "show tables",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "List tables in database `{{database}}`",
					SQL:         "SHOW TABLES FROM `{{database}}`",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show tables",
		},
		{
			Version: 1,
			Name:    "builtin_describe_table",
			Match: Match{
				Command: "describe table",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Describe table `{{table}}` in database `{{database}}`",
					SQL:         "DESCRIBE `{{database}}`.`{{table}}`",
				},
			},
			Layer:  "builtin",
			Source: "builtin:describe table",
		},
		{
			Version: 1,
			Name:    "builtin_show_indexes",
			Match: Match{
				Command: "show indexes",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show indexes for table `{{table}}` in database `{{database}}`",
					SQL:         "SHOW INDEXES FROM `{{table}}` FROM `{{database}}`",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show indexes",
		},
		{
			Version: 1,
			Name:    "builtin_show_grants",
			Match: Match{
				Command: "show grants",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show grants for '{{username}}'@'{{user_host}}'",
					SQL:         "SHOW GRANTS FOR '{{username}}'@'{{user_host}}'",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show grants",
		},
		{
			Version: 1,
			Name:    "builtin_show_processlist",
			Match: Match{
				Command: "show processlist",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show the active MySQL processlist on connection {{connection.name}}",
					SQL:         "SHOW PROCESSLIST",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show processlist",
		},
		{
			Version: 1,
			Name:    "builtin_show_variables",
			Match: Match{
				Command: "show variables",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show MySQL system variables{{variable_scope}}",
					SQL:         "SHOW VARIABLES{{variable_like_clause}}",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show variables",
		},
		{
			Version: 1,
			Name:    "builtin_drop_user",
			Match: Match{
				Command: "drop user",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Drop MySQL user '{{username}}'@'{{user_host}}'",
					SQL:         "DROP USER '{{username}}'@'{{user_host}}'",
				},
			},
			Layer:  "builtin",
			Source: "builtin:drop user",
		},
	}
}
