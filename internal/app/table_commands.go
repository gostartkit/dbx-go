package app

import (
	"context"
	"strings"

	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleShowCreateTable(ctx context.Context, table string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show create table", DryRun: a.dryRun}, func(meta *auditMetadata) error {
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

		template, err := a.templates.Resolve("show create table", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show create table template", err)
		}

		values := map[string]string{"database": database, "table": table}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		if a.dryRun {
			a.printPlanPreview(previewPlan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "show create table",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		ddl, err := a.connector.ShowCreateTable(ctx, cfg, db, database, table)
		if err != nil {
			return err
		}
		a.prompt.Println(ddl)
		return nil
	})
}

func (a *Application) handleShowTableStatus(ctx context.Context, table string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show table status", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, database, err := a.requireDatabaseContext(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		table, likeClause, err := normalizeOptionalTableName(table)
		if err != nil {
			return err
		}

		template, err := a.templates.Resolve("show table status", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show table status template", err)
		}

		values := map[string]string{
			"table_status_like_clause": likeClause,
			"table_status_scope":       tableStatusScopeLabel(table),
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
				Command:    "show table status",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		statuses, err := a.connector.ShowTableStatus(ctx, cfg, db, database, table)
		if err != nil {
			return err
		}
		statuses = sortedTableStatuses(statuses)
		if len(statuses) == 0 {
			a.prompt.Println("No table status found.")
			return nil
		}
		if table != "" {
			for _, line := range formatTableStatusDetail(statuses[0]) {
				a.prompt.Println(line)
			}
			return nil
		}
		for _, status := range statuses {
			a.prompt.Println(formatTableStatusSummary(status))
		}
		return nil
	})
}

func (a *Application) handleTruncateTable(ctx context.Context, table string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "truncate table", DryRun: a.dryRun}, func(meta *auditMetadata) error {
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

		template, err := a.templates.Resolve("truncate table", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve truncate table template", err)
		}

		values := map[string]string{"database": database, "table": table}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		a.printPlanPreview(previewPlan, a.dryRun)
		confirmed, err := a.confirmTableTruncate(ctx, table)
		if err != nil {
			return err
		}
		if !confirmed {
			a.prompt.Println("Cancelled.")
			return nil
		}

		if a.dryRun {
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "truncate table",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		if err := a.connector.TruncateTable(ctx, cfg, db, database, table); err != nil {
			return err
		}
		a.clearTableCompletion()
		a.printPlanResult(&PlanExecutionResult{
			OK:         true,
			Connection: cfg.Name,
			Command:    "truncate table",
			Actions: []ActionResult{{
				Description: plan.Actions[0].Description,
				Status:      ActionStatusOK,
			}},
		})
		return nil
	})
}

func (a *Application) handleRenameTable(ctx context.Context, from string, to string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "rename table", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, database, err := a.requireDatabaseContext(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		from, err = a.resolveTableName(ctx, from)
		if err != nil {
			return err
		}
		to, err = a.resolveTargetTableName(ctx, to)
		if err != nil {
			return err
		}

		template, err := a.templates.Resolve("rename table", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve rename table template", err)
		}

		values := map[string]string{
			"database":   database,
			"from_table": from,
			"to_table":   to,
		}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		confirmed, err := a.previewAndConfirm(ctx, "rename table", previewPlan)
		if err != nil {
			return err
		}
		if !confirmed {
			a.prompt.Println("Cancelled.")
			return nil
		}

		if a.dryRun {
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "rename table",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		if err := a.connector.RenameTable(ctx, cfg, db, database, from, to); err != nil {
			return err
		}
		a.clearTableCompletion()
		a.printPlanResult(&PlanExecutionResult{
			OK:         true,
			Connection: cfg.Name,
			Command:    "rename table",
			Actions: []ActionResult{{
				Description: plan.Actions[0].Description,
				Status:      ActionStatusOK,
			}},
		})
		return nil
	})
}

func normalizeOptionalTableName(table string) (string, string, error) {
	table = strings.TrimSpace(table)
	if table == "" {
		return "", "", nil
	}
	if err := util.ValidateTableName(table); err != nil {
		return "", "", util.WrapLayer("validation", "validate table name", err)
	}
	return table, " LIKE '" + util.EscapeMySQLString(table) + "'", nil
}

func (a *Application) resolveTargetTableName(ctx context.Context, table string) (string, error) {
	if strings.TrimSpace(table) == "" {
		value, err := a.ask(ctx, "New table name", "")
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

func (a *Application) confirmTableTruncate(ctx context.Context, table string) (bool, error) {
	behavior := behaviorForCommand("truncate table")
	if a.dryRun && behavior.SkipConfirmOnDryRun {
		return true, nil
	}
	a.prompt.Printf("This will delete all rows from table `%s`.\n", table)
	value, err := a.ask(ctx, "Type the table name to confirm", "")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(value) == table, nil
}
