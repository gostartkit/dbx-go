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
	"context":           {ReadOnly: true},
	"count rows":        {ReadOnly: true},
	"create connection": {Mutating: true, RequiresConfirmation: true},
	"create database":   {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"create user":       {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"describe table":    {ReadOnly: true},
	"doctor connection": {ReadOnly: true},
	"drop connection":   {Mutating: true, RequiresConfirmation: true},
	"drop database":     {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"drop user":         {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"edit connection":   {Mutating: true, RequiresConfirmation: true},
	"help":              {ReadOnly: true},
	"peek rows":         {ReadOnly: true},
	"rename table":      {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"run template":      {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"sample rows":       {ReadOnly: true},
	"show columns":      {ReadOnly: true},
	"show connection":   {ReadOnly: true},
	"show connections":  {ReadOnly: true},
	"show create table": {ReadOnly: true},
	"show databases":    {ReadOnly: true},
	"show foreign keys": {ReadOnly: true},
	"show grants":       {ReadOnly: true},
	"show indexes":      {ReadOnly: true},
	"show processlist":  {ReadOnly: true},
	"show table status": {ReadOnly: true},
	"show tables":       {ReadOnly: true},
	"show template":     {ReadOnly: true},
	"show templates":    {ReadOnly: true},
	"show triggers":     {ReadOnly: true},
	"show users":        {ReadOnly: true},
	"show variables":    {ReadOnly: true},
	"show views":        {ReadOnly: true},
	"test connection":   {ReadOnly: true},
	"truncate table":    {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"use database":      {ReadOnly: true},
	"validate template": {ReadOnly: true},
}

func behaviorForCommand(command string) CommandBehavior {
	if behavior, ok := commandBehaviors[command]; ok {
		return behavior
	}
	return CommandBehavior{}
}
