package app

import (
	"context"
	"fmt"
	"strconv"

	"pkg.gostartkit.com/dbx/internal/driver"
	"pkg.gostartkit.com/dbx/internal/util"
)

const (
	defaultRowInspectionLimit = 10
	maxRowInspectionLimit     = 100
)

func (a *Application) handleCountRows(ctx context.Context, table string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "count rows", DryRun: a.dryRun}, func(meta *auditMetadata) error {
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

		template, err := a.templates.Resolve("count rows", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve count rows template", err)
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
				Command:    "count rows",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		count, err := a.connector.CountRows(ctx, cfg, db, database, table)
		if err != nil {
			return err
		}
		a.prompt.Printf("%s: %d rows\n", table, count)
		return nil
	})
}

func (a *Application) handlePeekRows(ctx context.Context, table string, limitArg string) error {
	return a.handleRowPreview(ctx, "peek rows", table, limitArg, false)
}

func (a *Application) handleSampleRows(ctx context.Context, table string, limitArg string) error {
	return a.handleRowPreview(ctx, "sample rows", table, limitArg, true)
}

func (a *Application) handleRowPreview(ctx context.Context, command string, table string, limitArg string, random bool) error {
	return a.auditCommand(ctx, auditMetadata{Command: command, DryRun: a.dryRun}, func(meta *auditMetadata) error {
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
		limit, err := parseRowInspectionLimit(limitArg)
		if err != nil {
			return err
		}

		template, err := a.templates.Resolve(command, cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve "+command+" template", err)
		}

		values := map[string]string{
			"database": database,
			"table":    table,
			"limit":    strconv.Itoa(limit),
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
				Command:    command,
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		var result *driver.RowSet
		if random {
			result, err = a.connector.SampleRows(ctx, cfg, db, database, table, limit)
		} else {
			result, err = a.connector.PeekRows(ctx, cfg, db, database, table, limit)
		}
		if err != nil {
			return err
		}
		if result == nil || len(result.Columns) == 0 {
			a.prompt.Println("No rows found.")
			return nil
		}
		lines := formatRowPreview(result.Columns, result.Rows)
		for _, line := range lines {
			a.prompt.Println(line)
		}
		if len(result.Rows) == 0 {
			a.prompt.Println("No rows found.")
		}
		return nil
	})
}

func parseRowInspectionLimit(value string) (int, error) {
	if value == "" {
		return defaultRowInspectionLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, util.WrapLayer("validation", "parse row limit", fmt.Errorf("invalid limit %q", value))
	}
	if limit <= 0 {
		return 0, util.WrapLayer("validation", "parse row limit", fmt.Errorf("limit must be greater than 0"))
	}
	if limit > maxRowInspectionLimit {
		return maxRowInspectionLimit, nil
	}
	return limit, nil
}
