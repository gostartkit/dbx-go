package app

import (
	"context"

	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleShowColumns(ctx context.Context, table string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show columns", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, database, err := a.commandContext().requireDatabaseContext(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		table, err = a.resolveTableName(ctx, table)
		if err != nil {
			return err
		}

		template, err := a.resolveTemplateForAction(ctx, "show columns", cfg)
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
