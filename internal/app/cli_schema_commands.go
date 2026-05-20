package app

import (
	"context"
	"fmt"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (b *cliBuilder) showColumnsCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "columns",
		UsageLine:   "dbx show columns <table>",
		Short:       "Show columns for a table in the selected database",
		Long:        helpEntries["show columns"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "show columns", fmt.Errorf("usage: show columns <table>"))
				}
				return b.application.handleShowColumns(ctx, args[0])
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "show columns", fmt.Errorf("usage: dbx show columns <table>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show columns", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowColumns(ctx, application, args[0], meta)
			})
		},
	}
}

func (b *cliBuilder) columnsCommand() *cmd.Command {
	command := b.showColumnsCommand()
	command.Name = "columns"
	command.UsageLine = "dbx columns <table>"
	command.Short = "Alias for show columns"
	return command
}

func (b *cliBuilder) showForeignKeysCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "foreign",
		UsageLine: "dbx show foreign keys <table>",
		Short:     "Show foreign keys for a table in the selected database",
		Long:      helpEntries["show foreign keys"].body,
		SubCommands: []*cmd.Command{
			{
				Name:        "keys",
				UsageLine:   "dbx show foreign keys <table>",
				Short:       "Show foreign keys for a table in the selected database",
				Long:        helpEntries["show foreign keys"].body,
				Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
				Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
					if b.mode == ModeREPL {
						if len(args) != 1 {
							return util.WrapLayer("validation", "show foreign keys", fmt.Errorf("usage: show foreign keys <table>"))
						}
						return b.application.handleShowForeignKeys(ctx, args[0])
					}
					if len(args) != 1 {
						return util.WrapLayer("validation", "show foreign keys", fmt.Errorf("usage: dbx show foreign keys <table>"))
					}
					return b.withAuditedApplication(ctx, auditMetadata{Command: "show foreign keys", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
						return b.runShowForeignKeys(ctx, application, args[0], meta)
					})
				},
			},
		},
	}
}

func (b *cliBuilder) showFksCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "fks",
		UsageLine:   "dbx show fks <table>",
		Short:       "Alias for show foreign keys",
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "show foreign keys", fmt.Errorf("usage: show fks <table>"))
				}
				return b.application.handleShowForeignKeys(ctx, args[0])
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "show foreign keys", fmt.Errorf("usage: dbx show fks <table>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show foreign keys", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowForeignKeys(ctx, application, args[0], meta)
			})
		},
	}
}

func (b *cliBuilder) showTriggersCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "triggers",
		UsageLine: "dbx show triggers",
		Short:     "Show triggers in the selected database",
		Long:      helpEntries["show triggers"].body,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "show triggers", err)
				}
				return b.application.handleShowTriggers(ctx)
			}
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "show triggers", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show triggers", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowTriggers(ctx, application, meta)
			})
		},
	}
}

func (b *cliBuilder) showTriggerCommand() *cmd.Command {
	command := b.showTriggersCommand()
	command.Name = "trigger"
	command.UsageLine = "dbx show trigger"
	command.Short = "Alias for show triggers"
	return command
}

func (b *cliBuilder) showViewsCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "views",
		UsageLine: "dbx show views",
		Short:     "Show views in the selected database",
		Long:      helpEntries["show views"].body,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "show views", err)
				}
				return b.application.handleShowViews(ctx)
			}
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "show views", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show views", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowViews(ctx, application, meta)
			})
		},
	}
}

func (b *cliBuilder) showViewCommand() *cmd.Command {
	command := b.showViewsCommand()
	command.Name = "view"
	command.UsageLine = "dbx show view"
	command.Short = "Alias for show views"
	return command
}

func (b *cliBuilder) runShowColumns(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	if err := util.ValidateTableName(table); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "show columns")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show columns", cfg, "")
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
			result.Command = "show columns"
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

	columns, err := application.connector.ShowColumns(ctx, cfg, db, database, table)
	if err != nil {
		return err
	}
	result := &ColumnsResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Table:      table,
		Columns:    toSchemaColumnResults(columns),
	}
	return b.writeOutput(result, func() error {
		if len(columns) == 0 {
			fmt.Fprintln(b.out, "No columns found.")
			return nil
		}
		for _, column := range columns {
			fmt.Fprintln(b.out, formatSchemaColumnLine(column))
		}
		return nil
	})
}

func (b *cliBuilder) runShowForeignKeys(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	if err := util.ValidateTableName(table); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "show foreign keys")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show foreign keys", cfg, "")
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
			result.Command = "show foreign keys"
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

	keys, err := application.connector.ShowForeignKeys(ctx, cfg, db, database, table)
	if err != nil {
		return err
	}
	keys = sortedForeignKeys(keys)
	result := &ForeignKeysResult{
		OK:          true,
		Connection:  cfg.Name,
		Database:    database,
		Table:       table,
		ForeignKeys: toForeignKeyResults(keys),
	}
	return b.writeOutput(result, func() error {
		if len(keys) == 0 {
			fmt.Fprintln(b.out, "No foreign keys found.")
			return nil
		}
		for _, key := range keys {
			fmt.Fprintln(b.out, formatForeignKeyLine(key))
		}
		return nil
	})
}

func (b *cliBuilder) runShowTriggers(ctx context.Context, application *Application, meta *auditMetadata) error {
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "show triggers")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show triggers", cfg, "")
	if err != nil {
		return err
	}
	plan, previewPlan, err := buildPlans(template, cfg, map[string]string{"database": database})
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show triggers"
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

	triggers, err := application.connector.ShowTriggers(ctx, cfg, db, database)
	if err != nil {
		return err
	}
	triggers = sortedTriggers(triggers)
	result := &TriggersResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Triggers:   toTriggerResults(triggers),
	}
	return b.writeOutput(result, func() error {
		if len(triggers) == 0 {
			fmt.Fprintln(b.out, "No triggers found.")
			return nil
		}
		for _, trigger := range triggers {
			fmt.Fprintln(b.out, formatTriggerLine(trigger))
		}
		return nil
	})
}

func (b *cliBuilder) runShowViews(ctx context.Context, application *Application, meta *auditMetadata) error {
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "show views")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show views", cfg, "")
	if err != nil {
		return err
	}
	plan, previewPlan, err := buildPlans(template, cfg, map[string]string{"database": database})
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show views"
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

	views, err := application.connector.ShowViews(ctx, cfg, db, database)
	if err != nil {
		return err
	}
	result := &ViewsResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Views:      views,
	}
	return b.writeOutput(result, func() error {
		if len(views) == 0 {
			fmt.Fprintln(b.out, "No views found.")
			return nil
		}
		for _, view := range views {
			fmt.Fprintln(b.out, view)
		}
		return nil
	})
}
