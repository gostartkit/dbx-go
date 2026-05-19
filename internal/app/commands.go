package app

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleLine(ctx context.Context, line string) (bool, error) {
	switch strings.TrimSpace(line) {
	case "":
		return false, nil
	case "/", "/help":
		a.printHelp()
		return false, nil
	case "/connect":
		return false, a.handleConnect(ctx)
	case "/connections":
		return false, a.handleConnections()
	case "/status":
		return false, a.handleStatus(ctx)
	case "/create database":
		return false, a.handleCreateDatabase(ctx)
	case "/list databases":
		return false, a.handleListDatabases(ctx)
	case "/drop database":
		return false, a.handleDropDatabase(ctx)
	case "/exit":
		return true, nil
	default:
		return false, fmt.Errorf("unknown command %q; use /help", strings.TrimSpace(line))
	}
}

func (a *Application) printHelp() {
	a.prompt.Println("Available commands:")
	a.prompt.Println("  /                Show all commands")
	a.prompt.Println("  /help            Show all commands")
	a.prompt.Println("  /connect         Connect to a configured database")
	a.prompt.Println("  /connections     List configured connections")
	a.prompt.Println("  /status          Show the current session status")
	a.prompt.Println("  /create database Create a database from a template")
	a.prompt.Println("  /list databases  List databases on the active connection")
	a.prompt.Println("  /drop database   Drop a database from a template")
	a.prompt.Println("  /exit            Exit dbx")
}

func (a *Application) handleConnect(ctx context.Context) error {
	connections, err := a.store.ListConnections()
	if err != nil {
		return err
	}
	if len(connections) == 0 {
		return fmt.Errorf("no connections found in %s", a.store.RootDir)
	}

	names := make([]string, 0, len(connections))
	for _, connection := range connections {
		names = append(names, connection.Name)
	}

	defaultName := names[0]
	if a.session.Connection != nil && slices.Contains(names, a.session.Connection.Name) {
		defaultName = a.session.Connection.Name
	}

	name, err := a.prompt.Choose("Connection name", names, defaultName)
	if err != nil {
		return err
	}

	cfg, err := a.store.LoadConnection(name)
	if err != nil {
		return util.WrapLayer("config", "load connection "+name, err)
	}

	a.prompt.Println("Execution plan:")
	a.prompt.Printf("  1. Open %s MySQL connection %q to %s\n", cfg.Mode, cfg.Name, cfg.Address())
	if cfg.Mode == "ssh" {
		a.prompt.Printf("  2. Tunnel through SSH bastion %s:%d as %s\n", cfg.SSH.Host, cfg.SSH.Port, cfg.SSH.User)
	}

	confirmed, err := a.prompt.Confirm("Confirm execution?", true)
	if err != nil {
		return err
	}
	if !confirmed {
		a.prompt.Println("Cancelled.")
		return nil
	}

	db, err := a.connector.Open(ctx, cfg)
	if err != nil {
		return err
	}

	if err := a.session.Close(); err != nil {
		db.Close()
		return err
	}

	a.session.Connection = cfg
	a.session.DB = db

	if err := a.store.SaveSession(&config.SessionFile{CurrentConnection: cfg.Name}); err != nil {
		return util.WrapLayer("config", "save session", err)
	}

	a.prompt.Printf("Connected to %s.\n", cfg.Name)
	return nil
}

func (a *Application) handleConnections() error {
	a.prompt.Println("Execution plan:")
	a.prompt.Printf("  1. Read connection configs from %s\n", a.store.RootDir)

	confirmed, err := a.prompt.Confirm("Confirm execution?", true)
	if err != nil {
		return err
	}
	if !confirmed {
		a.prompt.Println("Cancelled.")
		return nil
	}

	connections, err := a.store.ListConnections()
	if err != nil {
		return util.WrapLayer("config", "list configured connections", err)
	}
	if len(connections) == 0 {
		a.prompt.Println("No configured connections found.")
		return nil
	}

	a.prompt.Println("Configured connections:")
	for _, connection := range connections {
		a.prompt.Printf("  - %s (%s %s %s)\n", connection.Name, connection.Driver, connection.Mode, connection.Address())
	}
	return nil
}

func (a *Application) handleStatus(ctx context.Context) error {
	a.prompt.Println("Execution plan:")
	a.prompt.Println("  1. Inspect the current session and ping the active database connection")

	confirmed, err := a.prompt.Confirm("Confirm execution?", true)
	if err != nil {
		return err
	}
	if !confirmed {
		a.prompt.Println("Cancelled.")
		return nil
	}

	if a.session.Connection == nil {
		a.prompt.Println("No active connection.")
		return nil
	}

	a.prompt.Printf("Connection: %s\n", a.session.Connection.Name)
	a.prompt.Printf("Driver: %s\n", a.session.Connection.Driver)
	a.prompt.Printf("Mode: %s\n", a.session.Connection.Mode)
	a.prompt.Printf("Address: %s\n", a.session.Connection.Address())

	if a.session.DB == nil {
		a.prompt.Println("Status: selected but not connected")
		return nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := a.session.DB.PingContext(pingCtx); err != nil {
		a.prompt.Printf("Status: connection error (%v)\n", err)
		return nil
	}

	a.prompt.Println("Status: connected")
	return nil
}

func (a *Application) handleCreateDatabase(ctx context.Context) error {
	cfg, db, err := a.requireConnection()
	if err != nil {
		return err
	}

	databaseName, err := a.prompt.Ask("Database name", "")
	if err != nil {
		return err
	}
	if err := util.ValidateIdentifier(databaseName); err != nil {
		return err
	}

	charset, err := a.prompt.Choose("Charset", []string{"utf8mb4", "utf8"}, "utf8mb4")
	if err != nil {
		return err
	}

	collationOptions := map[string][]string{
		"utf8mb4": {"utf8mb4_unicode_ci", "utf8mb4_general_ci"},
		"utf8":    {"utf8_unicode_ci", "utf8_general_ci"},
	}

	collationChoices := collationOptions[charset]
	collation, err := a.prompt.Choose("Collation", collationChoices, collationChoices[0])
	if err != nil {
		return err
	}

	template, err := a.templates.Resolve("create database", cfg)
	if err != nil {
		return util.WrapLayer("template", "resolve create database template", err)
	}

	values := map[string]string{
		"database":  databaseName,
		"charset":   charset,
		"collation": collation,
	}

	if err := a.collectTemplateInputs(template, values); err != nil {
		return util.WrapLayer("template", "collect template inputs", err)
	}

	plan, err := tpl.BuildPlan(template, cfg, values)
	if err != nil {
		return util.WrapLayer("template", "build create database execution plan", err)
	}

	previewPlan, err := tpl.BuildPlan(template, cfg, redactTemplateValues(template, values))
	if err != nil {
		return util.WrapLayer("template", "build redacted create database preview", err)
	}

	confirmed, err := a.previewAndConfirm(previewPlan)
	if err != nil {
		return err
	}
	if !confirmed {
		a.prompt.Println("Cancelled.")
		return nil
	}

	if err := a.connector.ExecStatements(ctx, cfg, db, statementsFromPlan(plan)); err != nil {
		return err
	}

	a.prompt.Printf("Database %s created.\n", databaseName)
	return nil
}

func (a *Application) handleListDatabases(ctx context.Context) error {
	cfg, db, err := a.requireConnection()
	if err != nil {
		return err
	}

	template, err := a.templates.Resolve("list databases", cfg)
	if err != nil {
		return util.WrapLayer("template", "resolve list databases template", err)
	}

	plan, err := tpl.BuildPlan(template, cfg, map[string]string{})
	if err != nil {
		return util.WrapLayer("template", "build list databases execution plan", err)
	}

	confirmed, err := a.previewAndConfirm(plan)
	if err != nil {
		return err
	}
	if !confirmed {
		a.prompt.Println("Cancelled.")
		return nil
	}

	results, err := a.connector.QueryStrings(ctx, cfg, db, plan.Actions[0].SQL)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		a.prompt.Println("No databases found.")
		return nil
	}

	a.prompt.Println("Databases:")
	for _, name := range results {
		a.prompt.Printf("  - %s\n", name)
	}
	return nil
}

func (a *Application) handleDropDatabase(ctx context.Context) error {
	cfg, db, err := a.requireConnection()
	if err != nil {
		return err
	}

	databases, err := a.connector.ListDatabases(ctx, cfg, db)
	if err != nil {
		return err
	}

	choices := filterDroppableDatabases(databases)
	if len(choices) == 0 {
		return fmt.Errorf("no droppable databases found")
	}

	databaseName, err := a.prompt.Choose("Database name", choices, "")
	if err != nil {
		return err
	}
	if err := util.ValidateIdentifier(databaseName); err != nil {
		return err
	}

	template, err := a.templates.Resolve("drop database", cfg)
	if err != nil {
		return util.WrapLayer("template", "resolve drop database template", err)
	}

	values := map[string]string{
		"database": databaseName,
	}

	if err := a.collectTemplateInputs(template, values); err != nil {
		return util.WrapLayer("template", "collect template inputs", err)
	}

	plan, err := tpl.BuildPlan(template, cfg, values)
	if err != nil {
		return util.WrapLayer("template", "build drop database execution plan", err)
	}

	previewPlan, err := tpl.BuildPlan(template, cfg, redactTemplateValues(template, values))
	if err != nil {
		return util.WrapLayer("template", "build redacted drop database preview", err)
	}

	confirmed, err := a.previewAndConfirm(previewPlan)
	if err != nil {
		return err
	}
	if !confirmed {
		a.prompt.Println("Cancelled.")
		return nil
	}

	if err := a.connector.ExecStatements(ctx, cfg, db, statementsFromPlan(plan)); err != nil {
		return err
	}

	a.prompt.Printf("Database %s dropped.\n", databaseName)
	return nil
}

func (a *Application) requireConnection() (*config.ConnectionConfig, *sql.DB, error) {
	if a.session.Connection == nil || a.session.DB == nil {
		return nil, nil, fmt.Errorf("no active database connection; run /connect first")
	}
	return a.session.Connection, a.session.DB, nil
}

func (a *Application) collectTemplateInputs(template *tpl.Template, values map[string]string) error {
	for _, input := range template.Inputs {
		if _, exists := values[input.Name]; exists {
			continue
		}

		var (
			value string
			err   error
		)

		switch {
		case len(input.Choices) > 0:
			value, err = a.prompt.Choose(input.Prompt, input.Choices, input.Default)
		case input.Secret:
			value, err = a.prompt.AskPassword(input.Prompt)
		default:
			value, err = a.prompt.Ask(input.Prompt, input.Default)
		}
		if err != nil {
			return err
		}

		if input.Identifier {
			if err := util.ValidateIdentifier(value); err != nil {
				return err
			}
		}

		values[input.Name] = value
	}
	return nil
}

func (a *Application) previewAndConfirm(plan *tpl.ExecutionPlan) (bool, error) {
	a.prompt.Printf("Template: %s (%s)\n", plan.TemplateName, plan.Layer)
	a.prompt.Printf("Source: %s\n", plan.Source)
	a.prompt.Println("Execution plan:")
	for index, action := range plan.Actions {
		a.prompt.Printf("  %d. %s\n", index+1, action.Description)
		a.prompt.Printf("     %s\n", action.SQL)
	}

	confirmed, err := a.prompt.Confirm("Confirm execution?", true)
	if err != nil {
		return false, err
	}
	return confirmed, nil
}

func statementsFromPlan(plan *tpl.ExecutionPlan) []string {
	statements := make([]string, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		statements = append(statements, action.SQL)
	}
	return statements
}

func filterDroppableDatabases(input []string) []string {
	systemDatabases := []string{"information_schema", "mysql", "performance_schema", "sys"}
	out := make([]string, 0, len(input))
	for _, name := range input {
		if slices.Contains(systemDatabases, name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func redactTemplateValues(template *tpl.Template, values map[string]string) map[string]string {
	redacted := make(map[string]string, len(values))
	for key, value := range values {
		redacted[key] = value
	}

	for _, input := range template.Inputs {
		if input.Secret {
			if _, exists := redacted[input.Name]; exists {
				redacted[input.Name] = "***"
			}
		}
	}

	return redacted
}
