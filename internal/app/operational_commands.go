package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleContext(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "context", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		result := a.currentContextResult()
		if result.Connection != "" {
			meta.Connection = result.Connection
			meta.Mode = result.Mode
		}
		a.prompt.Printf("Connection: %s\n", emptyValue(result.Connection, "<none>"))
		a.prompt.Printf("Database: %s\n", emptyValue(result.Database, "<none>"))
		a.prompt.Printf("Mode: %s\n", emptyValue(result.Mode, "<none>"))
		if result.DryRun {
			a.prompt.Println("Dry-run: on")
		} else {
			a.prompt.Println("Dry-run: off")
		}
		return nil
	})
}

func (a *Application) handleShowTables(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show tables", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, database, err := a.requireDatabaseContext(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		template, err := a.templates.Resolve("show tables", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show tables template", err)
		}

		values := map[string]string{"database": database}
		if err := a.collectTemplateInputs(ctx, template, values); err != nil {
			return util.WrapLayer("template", "collect template inputs", err)
		}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		if a.dryRun {
			a.printPlanPreview(previewPlan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "show tables",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		tables, err := a.connector.ListTables(ctx, cfg, db, database)
		if err != nil {
			return err
		}
		a.completionTablesConn = cfg.Name
		a.completionTablesDB = database
		a.completionTables = append([]string(nil), tables...)

		if len(tables) == 0 {
			a.prompt.Println("No tables found.")
			return nil
		}
		a.prompt.Println("Tables:")
		for _, table := range tables {
			a.prompt.Printf("  - %s\n", table)
		}
		return nil
	})
}

func (a *Application) handleDescribeTable(ctx context.Context, table string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "describe table", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, database, err := a.requireDatabaseContext(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		table, err = a.resolveTableName(ctx, table)
		if err != nil {
			return err
		}

		template, err := a.templates.Resolve("describe table", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve describe table template", err)
		}

		values := map[string]string{
			"database": database,
			"table":    table,
		}
		if err := a.collectTemplateInputs(ctx, template, values); err != nil {
			return util.WrapLayer("template", "collect template inputs", err)
		}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		if a.dryRun {
			a.printPlanPreview(previewPlan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "describe table",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		columns, err := a.connector.DescribeTable(ctx, cfg, db, database, table)
		if err != nil {
			return err
		}

		if len(columns) == 0 {
			a.prompt.Println("No columns found.")
			return nil
		}
		for _, column := range columns {
			a.prompt.Printf("%-16s %s\n", column.Name, column.Type)
		}
		return nil
	})
}

func (a *Application) handleShowGrants(ctx context.Context, username string, host string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show grants", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		username, err = a.resolveUsernameInput(ctx, username)
		if err != nil {
			return err
		}
		if strings.TrimSpace(host) == "" {
			host = "%"
		}
		if err := validateUserHost(host); err != nil {
			return util.WrapLayer("validation", "validate MySQL user host", err)
		}

		template, err := a.templates.Resolve("show grants", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show grants template", err)
		}

		values := map[string]string{
			"username":  username,
			"user_host": host,
		}
		if err := a.collectTemplateInputs(ctx, template, values); err != nil {
			return util.WrapLayer("template", "collect template inputs", err)
		}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		if a.dryRun {
			a.printPlanPreview(previewPlan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "show grants",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		grants, err := a.connector.ShowGrants(ctx, cfg, db, username, host)
		if err != nil {
			return err
		}
		if len(grants) == 0 {
			a.prompt.Println("No grants found.")
			return nil
		}
		a.prompt.Printf("Grants for %s@%s:\n", username, host)
		for _, grant := range grants {
			a.prompt.Printf("  - %s\n", grant)
		}
		return nil
	})
}

func (a *Application) requireDatabaseContext(ctx context.Context) (*config.ConnectionConfig, *sql.DB, string, error) {
	cfg, db, err := a.requireConnection(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	if strings.TrimSpace(a.session.Database) == "" {
		return nil, nil, "", util.WrapLayer("validation", "database context", fmt.Errorf("no database selected; use: use <database>"))
	}
	return cfg, db, a.session.Database, nil
}

func (a *Application) resolveTableName(ctx context.Context, table string) (string, error) {
	if strings.TrimSpace(table) == "" {
		value, err := a.ask(ctx, "Table name", "")
		if err != nil {
			return "", err
		}
		table = value
	}
	if err := util.ValidateTableName(table); err != nil {
		return "", util.WrapLayer("validation", "validate table name", err)
	}
	return strings.TrimSpace(table), nil
}

func (a *Application) currentContextResult() *ContextResult {
	result := &ContextResult{
		OK:     true,
		DryRun: a.dryRun,
	}
	if a.session == nil || a.session.Connection == nil {
		return result
	}
	result.Connection = a.session.Connection.Name
	result.Database = a.session.Database
	result.Mode = a.session.Connection.Mode
	return result
}

func emptyValue(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
