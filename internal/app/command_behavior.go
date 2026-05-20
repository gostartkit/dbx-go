package app

type CommandBehavior struct {
	ReadOnly             bool
	Mutating             bool
	RequiresConfirmation bool
	SkipConfirmOnDryRun  bool
}

var commandBehaviors = map[string]CommandBehavior{
	"audit log":          {ReadOnly: true},
	"connection create":  {Mutating: true, RequiresConfirmation: true},
	"connection delete":  {Mutating: true, RequiresConfirmation: true},
	"connection doctor":  {ReadOnly: true},
	"connection edit":    {Mutating: true, RequiresConfirmation: true},
	"connection show":    {ReadOnly: true},
	"connection test":    {ReadOnly: true},
	"connections":        {ReadOnly: true},
	"context":            {ReadOnly: true},
	"create database":    {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"create user":        {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"describe":           {ReadOnly: true},
	"describe table":     {ReadOnly: true},
	"drop database":      {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"drop user":          {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"help":               {ReadOnly: true},
	"list databases":     {ReadOnly: true},
	"list users":         {ReadOnly: true},
	"rename table":       {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"show databases":     {ReadOnly: true},
	"show create table":  {ReadOnly: true},
	"show grants":        {ReadOnly: true},
	"show index":         {ReadOnly: true},
	"show indexes":       {ReadOnly: true},
	"show dbs":           {ReadOnly: true},
	"show processlist":   {ReadOnly: true},
	"show processes":     {ReadOnly: true},
	"show table status":  {ReadOnly: true},
	"show user accounts": {ReadOnly: true},
	"show tables":        {ReadOnly: true},
	"show users":         {ReadOnly: true},
	"show variables":     {ReadOnly: true},
	"show vars":          {ReadOnly: true},
	"status":             {ReadOnly: true},
	"truncate table":     {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
}

func behaviorForCommand(command string) CommandBehavior {
	if behavior, ok := commandBehaviors[command]; ok {
		return behavior
	}
	return CommandBehavior{}
}
