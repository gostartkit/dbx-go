package app

import (
	"context"
	"fmt"
	"strconv"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/driver"
	"pkg.gostartkit.com/dbx/internal/util"
)

type rowLimitFlags struct {
	limit int
}

func (b *cliBuilder) showRowsCommand() *cmd.Command {
	flags := &rowLimitFlags{limit: defaultRowInspectionLimit}
	return &cmd.Command{
		Name:        "rows",
		UsageLine:   "dbx show rows <table> [--limit n]",
		Short:       "Show rows from a table",
		Long:        helpEntries["show rows"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
		SetFlags: func(f *cmd.FlagSet) {
			f.IntVar(&flags.limit, "limit", defaultRowInspectionLimit, "row limit", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				return b.application.handleShowRows(ctx, args[0], strconv.Itoa(flags.limit))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show rows", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runRowPreview(ctx, application, "show rows", "peek rows", args[0], flags.limit, false, meta)
			})
		},
	}
}

func (b *cliBuilder) runRowPreview(ctx context.Context, application *Application, command string, templateCommand string, table string, limit int, random bool, meta *auditMetadata) error {
	if err := util.ValidateTableName(table); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	normalizedLimit, err := normalizeRowInspectionLimit(limit)
	if err != nil {
		return err
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, command)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI(templateCommand, cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{
		"database": database,
		"table":    table,
		"limit":    strconv.Itoa(normalizedLimit),
	}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = command
			applyPreviewSQL(result, previewPlan)
		}
		return b.writeOutput(result, func() error {
			application.printPlanPreview(previewPlan, true)
			application.printPlanResult(result)
			return runErr
		})
	}

	db, err := application.openConnection(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	var rows *driver.RowSet
	if random {
		rows, err = application.connector.SampleRows(ctx, cfg, db, database, table, normalizedLimit)
	} else {
		rows, err = application.connector.PeekRows(ctx, cfg, db, database, table, normalizedLimit)
	}
	if err != nil {
		return err
	}
	if rows == nil {
		rows = &driver.RowSet{Columns: []string{}, Rows: [][]any{}}
	}
	result := &RowPreviewResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Table:      table,
		Columns:    append([]string(nil), rows.Columns...),
		Rows:       append([][]any(nil), rows.Rows...),
		Limit:      normalizedLimit,
	}
	return b.writeOutput(result, func() error {
		lines := formatRowPreview(rows.Columns, rows.Rows)
		for _, line := range lines {
			fmt.Fprintln(b.out, line)
		}
		if len(rows.Rows) == 0 {
			fmt.Fprintln(b.out, "No rows found.")
		}
		return nil
	})
}

func normalizeRowInspectionLimit(limit int) (int, error) {
	if limit <= 0 {
		return 0, util.WrapLayer("validation", "parse row limit", fmt.Errorf("limit must be greater than 0"))
	}
	if limit > maxRowInspectionLimit {
		return maxRowInspectionLimit, nil
	}
	return limit, nil
}
