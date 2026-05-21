package app

type CommandBehavior struct {
	ReadOnly             bool
	Mutating             bool
	RequiresConfirmation bool
	SkipConfirmOnDryRun  bool
}

var commandBehaviors = map[string]CommandBehavior{
	"audit log":         {ReadOnly: true},
	"connect":           {ReadOnly: true},
	"create connection": {Mutating: true, RequiresConfirmation: true},
	"create database":   {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"create user":       {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"doctor":            {ReadOnly: true},
	"drop connection":   {Mutating: true, RequiresConfirmation: true},
	"drop database":     {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"drop user":         {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"help":              {ReadOnly: true},
	"run sql":           {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"run template":      {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"show columns":      {ReadOnly: true},
	"show connection":   {ReadOnly: true},
	"show connections":  {ReadOnly: true},
	"show context":      {ReadOnly: true},
	"show databases":    {ReadOnly: true},
	"show rows":         {ReadOnly: true},
	"show table":        {ReadOnly: true},
	"show tables":       {ReadOnly: true},
	"show users":        {ReadOnly: true},
	"show templates":    {ReadOnly: true},
	"use database":      {ReadOnly: true},
}

func behaviorForCommand(command string) CommandBehavior {
	if behavior, ok := commandBehaviors[command]; ok {
		return behavior
	}
	return CommandBehavior{}
}
