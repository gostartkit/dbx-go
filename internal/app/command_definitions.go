package app

import "pkg.gostartkit.com/cmd"

func commandPositional(name string, usage string, kind string, completionKey string, required bool, completion cmd.CompletionFunc) cmd.PositionalArg {
	return cmd.PositionalArg{
		Name:          name,
		Usage:         usage,
		Required:      required,
		Kind:          kind,
		CompletionKey: completionKey,
		Completion:    completion,
	}
}

func connectionPositional(required bool, completion cmd.CompletionFunc) cmd.PositionalArg {
	return commandPositional("name", "saved connection name", "connection", "connection", required, completion)
}

func databasePositional(required bool, completion cmd.CompletionFunc) cmd.PositionalArg {
	return commandPositional("name", "database name", "database", "database", required, completion)
}

func tablePositional(completion cmd.CompletionFunc) cmd.PositionalArg {
	return commandPositional("table", "table name", "table", "table", true, completion)
}

func userPositional(required bool, completion cmd.CompletionFunc) cmd.PositionalArg {
	return commandPositional("name", "MySQL username", "user", "user", required, completion)
}

func operationPositional(completion cmd.CompletionFunc) cmd.PositionalArg {
	return commandPositional("operation", "operation name", "operation", "operation", true, completion)
}

func templateQueryPositional() cmd.PositionalArg {
	return commandPositional("query", "optional substring filter", "string", "", false, nil)
}

func helpTopicPositional(completion cmd.CompletionFunc) cmd.PositionalArg {
	return commandPositional("topic", "command or topic", "string", "topic", false, completion)
}
