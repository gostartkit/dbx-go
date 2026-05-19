package app

type CommandBehavior struct {
	ReadOnly             bool
	Mutating             bool
	RequiresConfirmation bool
	SkipConfirmOnDryRun  bool
}

var commandBehaviors = map[string]CommandBehavior{
	"audit log":         {ReadOnly: true},
	"connection create": {Mutating: true, RequiresConfirmation: true},
	"connection delete": {Mutating: true, RequiresConfirmation: true},
	"connection doctor": {ReadOnly: true},
	"connection edit":   {Mutating: true, RequiresConfirmation: true},
	"connection show":   {ReadOnly: true},
	"connection test":   {ReadOnly: true},
	"connections":       {ReadOnly: true},
	"create database":   {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"drop database":     {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"help":              {ReadOnly: true},
	"list databases":    {ReadOnly: true},
	"show databases":    {ReadOnly: true},
	"show dbs":          {ReadOnly: true},
	"status":            {ReadOnly: true},
}

func behaviorForCommand(command string) CommandBehavior {
	if behavior, ok := commandBehaviors[command]; ok {
		return behavior
	}
	return CommandBehavior{}
}
