package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (b *cliBuilder) showTablesCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "tables",
		UsageLine: "dbx show tables",
		Short:     "List tables in the selected database",
		Long:      helpLong("show tables"),
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "show tables", err)
				}
				return b.application.handleShowTables(ctx)
			}
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "show tables", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show tables", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowTables(ctx, application, meta)
			})
		},
	}
}

func (b *cliBuilder) showContextCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "context",
		UsageLine: "dbx show context",
		Short:     "Show the current operational context",
		Long:      helpLong("show context"),
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "show context", err)
				}
				return b.application.handleContext(ctx)
			}
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "show context", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show context", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				result, err := application.contextForCLI(ctx, b.globals.Connection, b.globals.Database)
				if err != nil {
					return err
				}
				if result.Connection != "" {
					meta.Connection = result.Connection
					meta.Mode = result.Mode
				}
				return b.writeOutput(result, func() error {
					fmt.Fprintf(b.out, "Connection: %s\n", emptyValue(result.Connection, "<none>"))
					fmt.Fprintf(b.out, "Database: %s\n", emptyValue(result.Database, "<none>"))
					fmt.Fprintf(b.out, "Mode: %s\n", emptyValue(result.Mode, "<none>"))
					if result.DryRun {
						fmt.Fprintln(b.out, "Dry-run: on")
					} else {
						fmt.Fprintln(b.out, "Dry-run: off")
					}
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) runShowTables(ctx context.Context, application *Application, meta *auditMetadata) error {
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "show tables")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show tables", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{"database": database}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		return b.writeDryRunPlanResult(ctx, application, cfg, "show tables", plan, previewPlan)
	}

	db, err := application.openConnection(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	tables, err := application.connector.ListTables(ctx, cfg, db, database)
	if err != nil {
		return err
	}
	result := &TablesResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Tables:     tables,
	}
	return b.writeOutput(result, func() error {
		if len(tables) == 0 {
			fmt.Fprintln(b.out, "No tables found.")
			return nil
		}
		for _, table := range tables {
			fmt.Fprintln(b.out, table)
		}
		return nil
	})
}

func (b *cliBuilder) resolveConnectionAndDatabase(ctx context.Context, application *Application, command string) (*config.ConnectionConfig, string, error) {
	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return nil, "", err
	}
	if err := application.applyCLIDatabaseSelection(ctx, cfg, b.globals.Database); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(application.session.Database) == "" {
		return nil, "", util.WrapLayer("validation", command, fmt.Errorf("no database selected; use --database <name>"))
	}
	return cfg, application.session.Database, nil
}

func (a *Application) contextForCLI(ctx context.Context, connectionName string, database string) (*ContextResult, error) {
	if strings.TrimSpace(connectionName) != "" {
		cfg, err := a.store.LoadConnection(connectionName)
		if err != nil {
			return nil, util.WrapLayer("config", "load connection "+connectionName, err)
		}
		if err := a.applyCLIDatabaseSelection(ctx, cfg, database); err != nil {
			return nil, err
		}
		return &ContextResult{
			OK:         true,
			Connection: cfg.Name,
			Database:   a.session.Database,
			Mode:       cfg.Mode,
			DryRun:     a.dryRun,
		}, nil
	}
	return a.currentContextResult(), nil
}
