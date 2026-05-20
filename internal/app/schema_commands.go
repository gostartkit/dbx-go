package app

import (
	"context"

	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleShowColumns(ctx context.Context, table string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show columns", DryRun: a.dryRun}, func(meta *auditMetadata) error {
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

		template, err := a.templates.Resolve("show columns", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show columns template", err)
		}

		values := map[string]string{
			"database": database,
			"table":    table,
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
				Command:    "show columns",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		columns, err := a.connector.ShowColumns(ctx, cfg, db, database, table)
		if err != nil {
			return err
		}
		if len(columns) == 0 {
			a.prompt.Println("No columns found.")
			return nil
		}
		for _, column := range columns {
			a.prompt.Println(formatSchemaColumnLine(column))
		}
		return nil
	})
}

func (a *Application) handleShowForeignKeys(ctx context.Context, table string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show foreign keys", DryRun: a.dryRun}, func(meta *auditMetadata) error {
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

		template, err := a.templates.Resolve("show foreign keys", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show foreign keys template", err)
		}

		values := map[string]string{
			"database": database,
			"table":    table,
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
				Command:    "show foreign keys",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		keys, err := a.connector.ShowForeignKeys(ctx, cfg, db, database, table)
		if err != nil {
			return err
		}
		keys = sortedForeignKeys(keys)
		if len(keys) == 0 {
			a.prompt.Println("No foreign keys found.")
			return nil
		}
		for _, key := range keys {
			a.prompt.Println(formatForeignKeyLine(key))
		}
		return nil
	})
}

func (a *Application) handleShowTriggers(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show triggers", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, database, err := a.requireDatabaseContext(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		template, err := a.templates.Resolve("show triggers", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show triggers template", err)
		}

		values := map[string]string{"database": database}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		if a.dryRun {
			a.printPlanPreview(previewPlan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "show triggers",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		triggers, err := a.connector.ShowTriggers(ctx, cfg, db, database)
		if err != nil {
			return err
		}
		triggers = sortedTriggers(triggers)
		if len(triggers) == 0 {
			a.prompt.Println("No triggers found.")
			return nil
		}
		for _, trigger := range triggers {
			a.prompt.Println(formatTriggerLine(trigger))
		}
		return nil
	})
}

func (a *Application) handleShowViews(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show views", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, database, err := a.requireDatabaseContext(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		template, err := a.templates.Resolve("show views", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show views template", err)
		}

		values := map[string]string{"database": database}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		if a.dryRun {
			a.printPlanPreview(previewPlan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "show views",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		views, err := a.connector.ShowViews(ctx, cfg, db, database)
		if err != nil {
			return err
		}
		if len(views) == 0 {
			a.prompt.Println("No views found.")
			return nil
		}
		for _, view := range views {
			a.prompt.Println(view)
		}
		return nil
	})
}
