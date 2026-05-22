package app

import (
	"context"
	"fmt"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (b *cliBuilder) showTableCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "table",
		UsageLine:   "dbx show table <table>",
		Short:       "Show table details",
		Long:        helpLong("show table"),
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				return b.application.handleShowCreateTable(ctx, args[0])
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show table", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowTable(ctx, application, args[0], meta)
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
		return b.writeDryRunPlanResult(ctx, application, cfg, "show create table", plan, previewPlan)
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

func (b *cliBuilder) runShowTable(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	return b.runShowCreateTable(ctx, application, table, meta)
}
