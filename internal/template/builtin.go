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
			Name:    "builtin_show_columns",
			Match: Match{
				Command: "show columns",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show columns for table `{{table}}` in database `{{database}}`",
					SQL:         "SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, EXTRA FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '{{database}}' AND TABLE_NAME = '{{table}}' ORDER BY ORDINAL_POSITION",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show columns",
		},
		{
			Version: 1,
			Name:    "builtin_show_create_table",
			Match: Match{
				Command: "show create table",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show CREATE TABLE for `{{table}}`",
					SQL:         "SHOW CREATE TABLE `{{table}}`",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show create table",
		},
		{
			Version: 1,
			Name:    "builtin_show_table_status",
			Match: Match{
				Command: "show table status",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show table status{{table_status_scope}}",
					SQL:         "SHOW TABLE STATUS{{table_status_like_clause}}",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show table status",
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
			Name:    "builtin_show_foreign_keys",
			Match: Match{
				Command: "show foreign keys",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show foreign keys for table `{{table}}` in database `{{database}}`",
					SQL:         "SELECT CONSTRAINT_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA = '{{database}}' AND TABLE_NAME = '{{table}}' AND REFERENCED_TABLE_NAME IS NOT NULL ORDER BY CONSTRAINT_NAME, ORDINAL_POSITION",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show foreign keys",
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
			Name:    "builtin_show_triggers",
			Match: Match{
				Command: "show triggers",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show triggers in database `{{database}}`",
					SQL:         "SHOW TRIGGERS",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show triggers",
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
			Name:    "builtin_show_views",
			Match: Match{
				Command: "show views",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Show views in database `{{database}}`",
					SQL:         "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.VIEWS WHERE TABLE_SCHEMA = '{{database}}' ORDER BY TABLE_NAME",
				},
			},
			Layer:  "builtin",
			Source: "builtin:show views",
		},
		{
			Version: 1,
			Name:    "builtin_truncate_table",
			Match: Match{
				Command: "truncate table",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Truncate table {{table}}",
					SQL:         "TRUNCATE TABLE `{{table}}`",
				},
			},
			Layer:  "builtin",
			Source: "builtin:truncate table",
		},
		{
			Version: 1,
			Name:    "builtin_rename_table",
			Match: Match{
				Command: "rename table",
				Driver:  "mysql",
			},
			Actions: []Action{
				{
					Type:        "sql",
					Description: "Rename table {{from_table}} -> {{to_table}}",
					SQL:         "RENAME TABLE `{{from_table}}` TO `{{to_table}}`",
				},
			},
			Layer:  "builtin",
			Source: "builtin:rename table",
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
