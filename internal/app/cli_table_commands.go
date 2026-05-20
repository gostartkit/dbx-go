package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (b *cliBuilder) showCreateGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "create",
		UsageLine: "dbx show create <subcommand>",
		Short:     "Show CREATE statements",
		SubCommands: []*cmd.Command{
			b.showCreateTableCommand(),
		},
	}
}

func (b *cliBuilder) showTableGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "table",
		UsageLine: "dbx show table <subcommand>",
		Short:     "Show table details",
		SubCommands: []*cmd.Command{
			b.showTableStatusCommand(),
		},
	}
}

func (b *cliBuilder) truncateGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "truncate",
		UsageLine: "dbx truncate <subcommand>",
		Short:     "Truncate database resources",
		SubCommands: []*cmd.Command{
			b.truncateTableCommand(),
		},
	}
}

func (b *cliBuilder) renameGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "rename",
		UsageLine: "dbx rename <subcommand>",
		Short:     "Rename database resources",
		SubCommands: []*cmd.Command{
			b.renameTableCommand(),
		},
	}
}

func (b *cliBuilder) showCreateTableCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "table",
		UsageLine:   "dbx show create table <table>",
		Short:       "Show CREATE TABLE for a table",
		Long:        helpEntries["show create table"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "show create table", fmt.Errorf("usage: show create table <table>"))
				}
				return b.application.handleShowCreateTable(ctx, args[0])
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "show create table", fmt.Errorf("usage: dbx show create table <table>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show create table", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowCreateTable(ctx, application, args[0], meta)
			})
		},
	}
}

func (b *cliBuilder) showTableStatusCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "status",
		UsageLine:   "dbx show table status [table]",
		Short:       "Show table status for one or more tables",
		Long:        helpEntries["show table status"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Completion: b.completeTables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) > 1 {
					return util.WrapLayer("validation", "show table status", fmt.Errorf("usage: show table status [table]"))
				}
				table := ""
				if len(args) == 1 {
					table = args[0]
				}
				return b.application.handleShowTableStatus(ctx, table)
			}
			if len(args) > 1 {
				return util.WrapLayer("validation", "show table status", fmt.Errorf("usage: dbx show table status [table]"))
			}
			table := ""
			if len(args) == 1 {
				table = args[0]
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show table status", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowTableStatus(ctx, application, table, meta)
			})
		},
	}
}

func (b *cliBuilder) truncateTableCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "table",
		UsageLine:   "dbx truncate table <table>",
		Short:       "Delete all rows from a table",
		Long:        helpEntries["truncate table"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "truncate table", fmt.Errorf("usage: truncate table <table>"))
				}
				return b.application.handleTruncateTable(ctx, args[0])
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "truncate table", fmt.Errorf("usage: dbx truncate table <table>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "truncate table", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runTruncateTable(ctx, application, args[0], meta)
			})
		},
	}
}

func (b *cliBuilder) renameTableCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "table",
		UsageLine: "dbx rename table <from> <to>",
		Short:     "Rename a table",
		Long:      helpEntries["rename table"].body,
		Positionals: []cmd.PositionalArg{
			{Name: "from", Usage: "existing table name", Required: true, Completion: b.completeTables},
			{Name: "to", Usage: "new table name", Required: true},
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 2 {
					return util.WrapLayer("validation", "rename table", fmt.Errorf("usage: rename table <from> <to>"))
				}
				return b.application.handleRenameTable(ctx, args[0], args[1])
			}
			if len(args) != 2 {
				return util.WrapLayer("validation", "rename table", fmt.Errorf("usage: dbx rename table <from> <to>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "rename table", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runRenameTable(ctx, application, args[0], args[1], meta)
			})
		},
	}
}

func (b *cliBuilder) runShowCreateTable(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	if err := util.ValidateTableName(table); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "show create table")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show create table", cfg, "")
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
			result.Command = "show create table"
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

	ddl, err := application.connector.ShowCreateTable(ctx, cfg, db, database, table)
	if err != nil {
		return err
	}
	result := &CreateTableResult{
		OK:          true,
		Connection:  cfg.Name,
		Database:    database,
		Table:       table,
		CreateTable: ddl,
	}
	return b.writeOutput(result, func() error {
		fmt.Fprintln(b.out, ddl)
		return nil
	})
}

func (b *cliBuilder) runShowTableStatus(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	table, likeClause, err := normalizeOptionalTableName(table)
	if err != nil {
		return err
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "show table status")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show table status", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{
		"table_status_like_clause": likeClause,
		"table_status_scope":       tableStatusScopeLabel(table),
	}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show table status"
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

	statuses, err := application.connector.ShowTableStatus(ctx, cfg, db, database, table)
	if err != nil {
		return err
	}
	statuses = sortedTableStatuses(statuses)
	result := &TableStatusResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Table:      table,
		Tables:     toTableStatusResults(statuses),
	}
	return b.writeOutput(result, func() error {
		if len(statuses) == 0 {
			fmt.Fprintln(b.out, "No table status found.")
			return nil
		}
		if table != "" {
			for _, line := range formatTableStatusDetail(statuses[0]) {
				fmt.Fprintln(b.out, line)
			}
			return nil
		}
		for _, status := range statuses {
			fmt.Fprintln(b.out, formatTableStatusSummary(status))
		}
		return nil
	})
}

func (b *cliBuilder) runTruncateTable(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	if err := util.ValidateTableName(table); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	if err := b.requireCLIConfirmation("truncate table"); err != nil {
		return err
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "truncate table")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("truncate table", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{"database": database, "table": table}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}
	if !b.globals.DryRun && !strings.EqualFold(b.globals.Format, "json") {
		application.printPlanPreview(previewPlan, false)
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "truncate table"
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

	if err := application.connector.TruncateTable(ctx, cfg, db, database, table); err != nil {
		return err
	}
	application.clearTableCompletion()
	result := &TableMutationResult{OK: true, Table: table, Action: "truncate"}
	return b.writeOutput(result, func() error {
		application.printPlanResult(&PlanExecutionResult{
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

func (b *cliBuilder) runRenameTable(ctx context.Context, application *Application, from string, to string, meta *auditMetadata) error {
	if err := util.ValidateTableName(from); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	if err := util.ValidateTableName(to); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	if err := b.requireCLIConfirmation("rename table"); err != nil {
		return err
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "rename table")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("rename table", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{"database": database, "from_table": from, "to_table": to}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}
	if !b.globals.DryRun && !strings.EqualFold(b.globals.Format, "json") {
		application.printPlanPreview(previewPlan, false)
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "rename table"
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

	if err := application.connector.RenameTable(ctx, cfg, db, database, from, to); err != nil {
		return err
	}
	application.clearTableCompletion()
	result := &TableMutationResult{OK: true, From: from, To: to, Action: "rename"}
	return b.writeOutput(result, func() error {
		application.printPlanResult(&PlanExecutionResult{
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
