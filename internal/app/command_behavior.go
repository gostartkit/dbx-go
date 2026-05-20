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
	"columns":            {ReadOnly: true},
	"count":              {ReadOnly: true},
	"count rows":         {ReadOnly: true},
	"context":            {ReadOnly: true},
	"create database":    {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"create user":        {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"describe":           {ReadOnly: true},
	"describe template":  {ReadOnly: true},
	"describe table":     {ReadOnly: true},
	"drop database":      {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"drop user":          {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"help":               {ReadOnly: true},
	"list databases":     {ReadOnly: true},
	"list users":         {ReadOnly: true},
	"rename table":       {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"peek":               {ReadOnly: true},
	"peek rows":          {ReadOnly: true},
	"sample":             {ReadOnly: true},
	"sample rows":        {ReadOnly: true},
	"show databases":     {ReadOnly: true},
	"show columns":       {ReadOnly: true},
	"show create table":  {ReadOnly: true},
	"show foreign keys":  {ReadOnly: true},
	"show grants":        {ReadOnly: true},
	"show fks":           {ReadOnly: true},
	"show index":         {ReadOnly: true},
	"show indexes":       {ReadOnly: true},
	"show dbs":           {ReadOnly: true},
	"show processlist":   {ReadOnly: true},
	"show processes":     {ReadOnly: true},
	"show table status":  {ReadOnly: true},
	"show trigger":       {ReadOnly: true},
	"show triggers":      {ReadOnly: true},
	"show user accounts": {ReadOnly: true},
	"show tables":        {ReadOnly: true},
	"show users":         {ReadOnly: true},
	"show variables":     {ReadOnly: true},
	"show vars":          {ReadOnly: true},
	"show view":          {ReadOnly: true},
	"show views":         {ReadOnly: true},
	"status":             {ReadOnly: true},
	"show templates":     {ReadOnly: true},
	"template":           {ReadOnly: true},
	"template describe":  {ReadOnly: true},
	"template run":       {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
	"template show":      {ReadOnly: true},
	"templates":          {ReadOnly: true},
	"truncate table":     {Mutating: true, RequiresConfirmation: true, SkipConfirmOnDryRun: true},
}

func behaviorForCommand(command string) CommandBehavior {
	if behavior, ok := commandBehaviors[command]; ok {
		return behavior
	}
	return CommandBehavior{}
}
