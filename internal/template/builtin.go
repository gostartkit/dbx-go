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
				Command: "list databases",
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
			Source: "builtin:list databases",
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
	}
}
