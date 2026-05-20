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
			if len(args) != 1 {
				return util.WrapLayer("validation", "show rows", fmt.Errorf("usage: dbx show rows <table> [--limit n]"))
			}
			if b.mode == ModeREPL {
				return b.application.handleShowRows(ctx, args[0], strconv.Itoa(flags.limit))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show rows", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runRowPreview(ctx, application, "show rows", "peek rows", args[0], flags.limit, false, meta)
			})
		},
	}
}

func (b *cliBuilder) countCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "count",
		UsageLine: "dbx count rows <table>",
		Short:     "Count rows in a table",
		Long:      helpEntries["count rows"].body,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				table, err := parseCountArgs(args)
				if err != nil {
					return util.WrapLayer("validation", "count rows", err)
				}
				return b.application.handleCountRows(ctx, table)
			}
			table, err := parseCountArgs(args)
			if err != nil {
				return util.WrapLayer("validation", "count rows", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "count rows", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runCountRows(ctx, application, table, meta)
			})
		},
	}
}

func (b *cliBuilder) peekCommand() *cmd.Command {
	flags := &rowLimitFlags{limit: defaultRowInspectionLimit}
	return &cmd.Command{
		Name:      "peek",
		UsageLine: "dbx peek rows <table>",
		Short:     "Peek bounded rows from a table",
		Long:      helpEntries["peek rows"].body,
		SetFlags: func(f *cmd.FlagSet) {
			f.IntVar(&flags.limit, "limit", defaultRowInspectionLimit, "row limit", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				table, limit, err := parsePreviewArgs("peek", args, flags.limit)
				if err != nil {
					return util.WrapLayer("validation", "peek rows", err)
				}
				return b.application.handlePeekRows(ctx, table, strconv.Itoa(limit))
			}
			table, limit, err := parsePreviewArgs("peek", args, flags.limit)
			if err != nil {
				return util.WrapLayer("validation", "peek rows", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "peek rows", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runRowPreview(ctx, application, "peek rows", "peek rows", table, limit, false, meta)
			})
		},
	}
}

func (b *cliBuilder) sampleCommand() *cmd.Command {
	flags := &rowLimitFlags{limit: defaultRowInspectionLimit}
	return &cmd.Command{
		Name:      "sample",
		UsageLine: "dbx sample rows <table>",
		Short:     "Sample bounded rows from a table",
		Long:      helpEntries["sample rows"].body,
		SetFlags: func(f *cmd.FlagSet) {
			f.IntVar(&flags.limit, "limit", defaultRowInspectionLimit, "row limit", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				table, limit, err := parsePreviewArgs("sample", args, flags.limit)
				if err != nil {
					return util.WrapLayer("validation", "sample rows", err)
				}
				return b.application.handleSampleRows(ctx, table, strconv.Itoa(limit))
			}
			table, limit, err := parsePreviewArgs("sample", args, flags.limit)
			if err != nil {
				return util.WrapLayer("validation", "sample rows", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "sample rows", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runRowPreview(ctx, application, "sample rows", "sample rows", table, limit, true, meta)
			})
		},
	}
}

func (b *cliBuilder) runCountRows(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	if err := util.ValidateTableName(table); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "count rows")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("count rows", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{"database": database, "table": table}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "count rows"
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

	count, err := application.connector.CountRows(ctx, cfg, db, database, table)
	if err != nil {
		return err
	}
	result := &RowCountResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Table:      table,
		Rows:       count,
	}
	return b.writeOutput(result, func() error {
		fmt.Fprintf(b.out, "%s: %d rows\n", table, count)
		return nil
	})
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

func parseCountArgs(args []string) (string, error) {
	switch len(args) {
	case 1:
		return args[0], nil
	case 2:
		if args[0] != "rows" {
			return "", fmt.Errorf("usage: dbx count rows <table>")
		}
		return args[1], nil
	default:
		return "", fmt.Errorf("usage: dbx count rows <table>")
	}
}

func parsePreviewArgs(command string, args []string, limit int) (string, int, error) {
	switch len(args) {
	case 1:
		return args[0], limit, nil
	case 2:
		if args[0] == "rows" {
			return args[1], limit, nil
		}
		parsed, err := strconv.Atoi(args[1])
		if err != nil {
			return "", 0, fmt.Errorf("usage: dbx %s rows <table> [limit]", command)
		}
		return args[0], parsed, nil
	case 3:
		if args[0] != "rows" {
			return "", 0, fmt.Errorf("usage: dbx %s rows <table> [limit]", command)
		}
		parsed, err := strconv.Atoi(args[2])
		if err != nil {
			return "", 0, fmt.Errorf("usage: dbx %s rows <table> [limit]", command)
		}
		return args[1], parsed, nil
	default:
		return "", 0, fmt.Errorf("usage: dbx %s rows <table> [limit]", command)
	}
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
