package app

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleLine(ctx context.Context, line string) (bool, error) {
	line = strings.TrimSpace(line)
	if line != "" {
		if err := a.recordHistory(line); err != nil {
			a.prompt.Printf("Warning: %v\n", err)
		}
	}

	resolved := resolveAlias(line)
	fields := strings.Fields(resolved)

	switch {
	case len(fields) == 2 && fields[0] == "connect":
		return false, a.handleConnectByName(ctx, fields[1])
	case len(fields) == 2 && fields[0] == "use":
		return false, a.handleUseDatabase(ctx, fields[1])
	case line == "connection create":
		return false, a.handleConnectionCreate(ctx)
	case len(fields) == 2 && fields[0] == "connection" && fields[1] == "doctor":
		return false, a.handleConnectionDoctor(ctx, "")
	case len(fields) == 3 && fields[0] == "connection" && fields[1] == "doctor":
		return false, a.handleConnectionDoctor(ctx, fields[2])
	case len(fields) == 3 && fields[0] == "connection" && fields[1] == "edit":
		return false, a.handleConnectionEdit(ctx, fields[2])
	case len(fields) == 3 && fields[0] == "connection" && fields[1] == "delete":
		return false, a.handleConnectionDelete(ctx, fields[2])
	case len(fields) == 3 && fields[0] == "connection" && fields[1] == "show":
		return false, a.handleConnectionShow(ctx, fields[2])
	case len(fields) >= 2 && fields[0] == "connection" && fields[1] == "test":
		name, verbose, ok := parseConnectionTestArgs(fields[2:])
		if ok {
			return false, a.handleConnectionTest(ctx, name, verbose)
		}
	case len(fields) >= 1 && fields[0] == "help":
		return false, a.handleHelp(strings.TrimSpace(strings.TrimPrefix(line, "help")))
	}

	switch resolved {
	case "":
		return false, nil
	case "/":
		return false, a.handleHelp("")
	case "help":
		return false, a.handleHelp("")
	case "connect":
		return false, a.handleConnect(ctx)
	case "connections":
		return false, a.handleConnections(ctx)
	case "audit log":
		return false, a.handleAuditLog(ctx)
	case "status":
		return false, a.handleStatus(ctx)
	case "create database":
		return false, a.handleCreateDatabase(ctx)
	case "list databases":
		return false, a.handleListDatabases(ctx)
	case "drop database":
		return false, a.handleDropDatabase(ctx)
	case "dry-run on":
		a.dryRun = true
		a.prompt.Println("Dry-run mode is on.")
		return false, nil
	case "dry-run off":
		a.dryRun = false
		a.prompt.Println("Dry-run mode is off.")
		return false, nil
	case "exit":
		return true, nil
	default:
		return false, fmt.Errorf("unknown command %q; use / or help", line)
	}
}

func (a *Application) handleHelp(topic string) error {
	return printHelpTopic(a.prompt, topic)
}

func (a *Application) handleConnect(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "connect"}, func(meta *auditMetadata) error {
		connections, err := a.store.ListConnections()
		if err != nil {
			return util.WrapLayer("config", "list configured connections", err)
		}
		if len(connections) == 0 {
			a.prompt.Println("No saved connections. Run: connection create")
			return nil
		}

		name, err := a.promptForConnectionSelection(ctx, connections)
		if err != nil {
			return err
		}
		return a.connectByName(ctx, name, meta)
	})
}

func (a *Application) handleConnections(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "connections"}, func(meta *auditMetadata) error {
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
	})
}

func (a *Application) handleStatus(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "status", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		if a.session.Connection == nil {
			a.prompt.Println("No active connection.")
			return nil
		}
		meta.Connection = a.session.Connection.Name
		meta.Mode = a.session.Connection.Mode

		a.prompt.Printf("Connection: %s\n", a.session.Connection.Name)
		if strings.TrimSpace(a.session.Database) != "" {
			a.prompt.Printf("Database: %s\n", a.session.Database)
		}
		a.prompt.Printf("Driver: %s\n", a.session.Connection.Driver)
		a.prompt.Printf("Mode: %s\n", a.session.Connection.Mode)
		a.prompt.Printf("Address: %s\n", a.session.Connection.Address())
		a.prompt.Printf("Connect timeout: %s\n", a.session.Connection.ConnectTimeout())
		a.prompt.Printf("Query timeout: %s\n", a.session.Connection.QueryTimeout())
		a.prompt.Printf("Dry run: %t\n", a.dryRun)

		if a.session.DB == nil {
			a.prompt.Println("Status: selected but not connected")
			return nil
		}

		if err := a.connector.Ping(ctx, a.session.Connection, a.session.DB); err != nil {
			_ = a.session.Close()
			failed := false
			meta.Success = &failed
			a.prompt.Printf("Status: stale connection (%v)\n", err)
			return nil
		}

		a.prompt.Println("Status: connected")
		return nil
	})
}

func (a *Application) handleUseDatabase(ctx context.Context, database string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "use"}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		if err := a.setRuntimeDatabaseSelection(ctx, cfg, db, database, true); err != nil {
			if strings.Contains(err.Error(), "database not found:") {
				return fmt.Errorf("Database not found: %s", database)
			}
			return err
		}
		a.prompt.Printf("Using database: %s\n", a.session.Database)
		return nil
	})
}

func (a *Application) handleCreateDatabase(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "create database", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		databaseName, err := a.ask(ctx, "Database name", "")
		if err != nil {
			return err
		}
		if err := util.ValidateDatabaseName(databaseName); err != nil {
			return err
		}

		charset, err := a.choose(ctx, "Charset", []string{"utf8mb4", "utf8"}, "utf8mb4")
		if err != nil {
			return err
		}

		collationOptions := map[string][]string{
			"utf8mb4": {"utf8mb4_unicode_ci", "utf8mb4_general_ci"},
			"utf8":    {"utf8_unicode_ci", "utf8_general_ci"},
		}

		collationChoices := collationOptions[charset]
		collation, err := a.choose(ctx, "Collation", collationChoices, collationChoices[0])
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

		if err := a.collectTemplateInputs(ctx, template, values); err != nil {
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

		confirmed, err := a.previewAndConfirm(ctx, "create database", previewPlan)
		if err != nil {
			return err
		}
		if !confirmed {
			a.prompt.Println("Cancelled.")
			return nil
		}

		result, err := a.runPlan(ctx, plan, sqlRunner{db: db}, a.dryRun)
		a.printPlanResult(result)
		if err != nil {
			return err
		}

		a.prompt.Printf("Database %s created.\n", databaseName)
		return nil
	})
}

func (a *Application) handleListDatabases(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "list databases", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		template, err := a.templates.Resolve("list databases", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve list databases template", err)
		}

		plan, err := tpl.BuildPlan(template, cfg, map[string]string{})
		if err != nil {
			return util.WrapLayer("template", "build list databases execution plan", err)
		}

		if a.dryRun {
			a.printPlanPreview(plan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:      true,
				DryRun:  true,
				Actions: []ActionResult{{Description: plan.Actions[0].Description, SQL: plan.Actions[0].SQL, Status: ActionStatusDryRun}},
			})
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
	})
}

func (a *Application) handleDropDatabase(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "drop database", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		databases, err := a.connector.ListDatabases(ctx, cfg, db)
		if err != nil {
			return err
		}

		choices := filterDroppableDatabases(databases)
		if len(choices) == 0 {
			return fmt.Errorf("no droppable databases found")
		}

		databaseName, err := a.choose(ctx, "Database name", choices, "")
		if err != nil {
			return err
		}
		if err := util.ValidateDatabaseName(databaseName); err != nil {
			return err
		}

		template, err := a.templates.Resolve("drop database", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve drop database template", err)
		}

		values := map[string]string{
			"database": databaseName,
		}

		if err := a.collectTemplateInputs(ctx, template, values); err != nil {
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

		confirmed, err := a.previewAndConfirm(ctx, "drop database", previewPlan)
		if err != nil {
			return err
		}
		if !confirmed {
			a.prompt.Println("Cancelled.")
			return nil
		}

		result, err := a.runPlan(ctx, plan, sqlRunner{db: db}, a.dryRun)
		a.printPlanResult(result)
		if err != nil {
			return err
		}

		a.prompt.Printf("Database %s dropped.\n", databaseName)
		return nil
	})
}

func (a *Application) requireConnection(ctx context.Context) (*config.ConnectionConfig, *sql.DB, error) {
	if a.session.Connection == nil || a.session.DB == nil {
		return nil, nil, fmt.Errorf("no active database connection; run connect first")
	}
	if err := a.connector.Ping(ctx, a.session.Connection, a.session.DB); err != nil {
		_ = a.session.Close()
		return nil, nil, util.WrapLayer("mysql", "active connection is stale; reconnect with connect", err)
	}
	return a.session.Connection, a.session.DB, nil
}

func (a *Application) collectTemplateInputs(ctx context.Context, template *tpl.Template, values map[string]string) error {
	for _, input := range template.Inputs {
		if _, exists := values[input.Name]; exists {
			continue
		}

		var (
			value string
			err   error
		)

		switch input.EffectiveType() {
		case "select":
			value, err = a.choose(ctx, input.Prompt, input.SelectOptions(), input.DefaultString())
		case "secret":
			value, err = a.askPassword(ctx, input.Prompt)
		case "confirm":
			confirmed, confirmErr := a.confirm(ctx, input.Prompt, input.DefaultBool())
			if confirmErr != nil {
				return confirmErr
			}
			if confirmed {
				value = "true"
			} else {
				value = "false"
			}
		case "identifier":
			value, err = a.ask(ctx, input.Prompt, input.DefaultString())
			if err == nil {
				err = util.ValidateIdentifier(value)
			}
		case "int":
			var intValue int
			defaultInt, ok := input.DefaultInt()
			if ok {
				intValue, err = a.askInt(ctx, input.Prompt, defaultInt)
			} else {
				intValue, err = a.askInt(ctx, input.Prompt, 0)
			}
			if err == nil {
				value = fmt.Sprintf("%d", intValue)
			}
		default:
			value, err = a.ask(ctx, input.Prompt, input.DefaultString())
		}
		if err != nil {
			return err
		}

		normalized, err := normalizeTemplateInputValue(input, value)
		if err != nil {
			return err
		}

		if input.Identifier && input.EffectiveType() != "identifier" {
			if err := util.ValidateIdentifier(normalized); err != nil {
				return err
			}
		}

		values[input.Name] = normalized
	}
	return nil
}

func (a *Application) previewAndConfirm(ctx context.Context, command string, plan *tpl.ExecutionPlan) (bool, error) {
	a.printPlanPreview(plan, a.dryRun)
	return a.confirmExecutionIfNeeded(ctx, command)
}

func (a *Application) printPlanPreview(plan *tpl.ExecutionPlan, dryRun bool) {
	a.prompt.Printf("Template: %s (%s)\n", plan.TemplateName, plan.Layer)
	a.prompt.Printf("Source: %s\n", plan.Source)
	a.prompt.Println("Execution Plan")
	for index, action := range plan.Actions {
		a.prompt.Printf("  %d. %s\n", index+1, action.Description)
	}
	a.prompt.Println("Rendered SQL")
	for index, action := range plan.Actions {
		a.prompt.Printf("  %d. %s\n", index+1, action.SQL)
	}
	if dryRun {
		a.prompt.Println("Dry-run mode is enabled. SQL will be rendered but not executed.")
	}
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
		if input.EffectiveType() == "secret" {
			if _, exists := redacted[input.Name]; exists {
				redacted[input.Name] = "***"
			}
		}
	}

	return redacted
}
