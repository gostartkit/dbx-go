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
				return b.application.handleShowColumns(ctx, args[0])
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show columns", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowColumns(ctx, application, args[0], meta)
			})
		},
	}
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
		return b.writeDryRunPlanResult(ctx, application, cfg, "show columns", plan, previewPlan)
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
