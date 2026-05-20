package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (b *cliBuilder) showTablesCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "tables",
		UsageLine: "dbx show tables",
		Short:     "List tables in the selected database",
		Long:      helpEntries["show tables"].body,
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
		Long:      helpEntries["show context"].body,
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

func (b *cliBuilder) showIndexesCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "indexes",
		UsageLine:   "dbx show indexes <table>",
		Short:       "Show indexes for a table in the selected database",
		Long:        helpEntries["show indexes"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "show indexes", fmt.Errorf("usage: show indexes <table>"))
				}
				return b.application.handleShowIndexes(ctx, args[0])
			}
			table := ""
			switch len(args) {
			case 1:
				table = args[0]
			case 2:
				if args[0] != "on" {
					return util.WrapLayer("validation", "show indexes", fmt.Errorf("usage: dbx show indexes <table>"))
				}
				table = args[1]
			default:
				return util.WrapLayer("validation", "show indexes", fmt.Errorf("usage: dbx show indexes <table>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show indexes", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowIndexes(ctx, application, table, meta)
			})
		},
	}
}

func (b *cliBuilder) showGrantsCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "grants",
		UsageLine:   "dbx show grants <user> [host]",
		Short:       "Show MySQL grants for a user",
		Long:        helpEntries["show grants"].body,
		Positionals: []cmd.PositionalArg{{Name: "user", Usage: "MySQL username", Required: true, Completion: b.completeUsers}, {Name: "host", Usage: "MySQL user host"}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) == 0 || len(args) > 2 {
					return util.WrapLayer("validation", "show grants", fmt.Errorf("usage: show grants <user> [host]"))
				}
				host := ""
				if len(args) == 2 {
					host = args[1]
				}
				return b.application.handleShowGrants(ctx, args[0], host)
			}
			if len(args) == 0 || len(args) > 2 {
				return util.WrapLayer("validation", "show grants", fmt.Errorf("usage: dbx show grants <user> [host]"))
			}
			host := "%"
			if len(args) == 2 {
				host = args[1]
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show grants", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowGrants(ctx, application, args[0], host, meta)
			})
		},
	}
}

func (b *cliBuilder) showProcesslistCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "processlist",
		UsageLine: "dbx show processlist",
		Short:     "Show the active MySQL processlist",
		Long:      helpEntries["show processlist"].body,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "show processlist", err)
				}
				return b.application.handleShowProcesslist(ctx)
			}
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "show processlist", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show processlist", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowProcesslist(ctx, application, meta)
			})
		},
	}
}

func (b *cliBuilder) showVariablesCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "variables",
		UsageLine:   "dbx show variables [pattern]",
		Short:       "Show MySQL system variables",
		Long:        helpEntries["show variables"].body,
		Positionals: []cmd.PositionalArg{{Name: "pattern", Usage: "exact name or LIKE pattern", Completion: b.completeVariables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) > 1 {
					return util.WrapLayer("validation", "show variables", fmt.Errorf("usage: show variables [pattern]"))
				}
				pattern := ""
				if len(args) == 1 {
					pattern = args[0]
				}
				return b.application.handleShowVariables(ctx, pattern)
			}
			if len(args) > 1 {
				return util.WrapLayer("validation", "show variables", fmt.Errorf("usage: dbx show variables [pattern]"))
			}
			pattern := ""
			if len(args) == 1 {
				pattern = args[0]
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show variables", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowVariables(ctx, application, pattern, meta)
			})
		},
	}
}

func (b *cliBuilder) contextCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "context",
		UsageLine: "dbx context",
		Short:     "Show the current operational context",
		Long:      helpEntries["context"].body,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "context", err)
				}
				return b.application.handleContext(ctx)
			}
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "context", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "context", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
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

func (b *cliBuilder) describeCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "describe",
		UsageLine:   "dbx describe <table>",
		Short:       "Describe a table in the selected database",
		Long:        helpEntries["describe"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true, Completion: b.completeTables}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "describe", fmt.Errorf("usage: describe <table>"))
				}
				return b.application.handleDescribeTable(ctx, args[0])
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "describe", fmt.Errorf("usage: dbx describe <table>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "describe table", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runDescribeTable(ctx, application, args[0], meta)
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
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show tables"
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

func (b *cliBuilder) runShowIndexes(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	if err := util.ValidateTableName(table); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "show indexes")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show indexes", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{
		"database": database,
		"table":    table,
	}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show indexes"
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

	indexes, err := application.connector.ShowIndexes(ctx, cfg, db, database, table)
	if err != nil {
		return err
	}
	indexes = sortedIndexes(indexes)
	result := &TableIndexesResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Table:      table,
		Indexes:    toTableIndexResults(indexes),
	}
	return b.writeOutput(result, func() error {
		if len(indexes) == 0 {
			fmt.Fprintln(b.out, "No indexes found.")
			return nil
		}
		for _, index := range indexes {
			fmt.Fprintln(b.out, formatIndexLine(index))
		}
		return nil
	})
}

func (b *cliBuilder) runDescribeTable(ctx context.Context, application *Application, table string, meta *auditMetadata) error {
	if err := util.ValidateTableName(table); err != nil {
		return util.WrapLayer("validation", "validate table name", err)
	}
	cfg, database, err := b.resolveConnectionAndDatabase(ctx, application, "describe")
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("describe table", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{
		"database": database,
		"table":    table,
	}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "describe table"
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

	columns, err := application.connector.DescribeTable(ctx, cfg, db, database, table)
	if err != nil {
		return err
	}
	result := &TableDescriptionResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   database,
		Table:      table,
		Columns:    toTableColumnResults(columns),
	}
	return b.writeOutput(result, func() error {
		if len(columns) == 0 {
			fmt.Fprintln(b.out, "No columns found.")
			return nil
		}
		for _, column := range columns {
			fmt.Fprintf(b.out, "%-16s %s\n", column.Name, column.Type)
		}
		return nil
	})
}

func (b *cliBuilder) runShowGrants(ctx context.Context, application *Application, user string, host string, meta *auditMetadata) error {
	if err := util.ValidateMySQLUsername(user); err != nil {
		return util.WrapLayer("validation", "validate MySQL username", err)
	}
	if err := validateUserHost(host); err != nil {
		return util.WrapLayer("validation", "validate MySQL user host", err)
	}

	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show grants", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{
		"username":  user,
		"user_host": host,
	}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show grants"
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

	grants, err := application.connector.ShowGrants(ctx, cfg, db, user, host)
	if err != nil {
		return err
	}
	result := &GrantsResult{
		OK:         true,
		Connection: cfg.Name,
		User:       user,
		Host:       host,
		Grants:     grants,
	}
	return b.writeOutput(result, func() error {
		if len(grants) == 0 {
			fmt.Fprintln(b.out, "No grants found.")
			return nil
		}
		for _, grant := range grants {
			fmt.Fprintln(b.out, grant)
		}
		return nil
	})
}

func (b *cliBuilder) runShowProcesslist(ctx context.Context, application *Application, meta *auditMetadata) error {
	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.selectTemplateForCLI("show processlist", cfg, "")
	if err != nil {
		return err
	}
	plan, previewPlan, err := buildPlans(template, cfg, map[string]string{})
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show processlist"
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

	processes, err := application.connector.ShowProcesslist(ctx, cfg, db)
	if err != nil {
		return err
	}
	processes = sortedProcesses(processes)
	result := &ProcesslistResult{
		OK:         true,
		Connection: cfg.Name,
		Processes:  toProcessResults(processes),
	}
	return b.writeOutput(result, func() error {
		if len(processes) == 0 {
			fmt.Fprintln(b.out, "No processes found.")
			return nil
		}
		for _, process := range processes {
			fmt.Fprintln(b.out, formatProcessLine(process))
		}
		return nil
	})
}

func (b *cliBuilder) runShowVariables(ctx context.Context, application *Application, pattern string, meta *auditMetadata) error {
	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	pattern, likeClause, err := normalizeVariablePattern(pattern)
	if err != nil {
		return err
	}

	template, err := application.selectTemplateForCLI("show variables", cfg, "")
	if err != nil {
		return err
	}
	values := map[string]string{
		"variable_like_clause": likeClause,
		"variable_scope":       variableScopeLabel(pattern),
	}
	plan, previewPlan, err := buildPlans(template, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show variables"
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

	variables, err := application.connector.ShowVariables(ctx, cfg, db, pattern)
	if err != nil {
		return util.WrapLayer("mysql", "variable query failed", err)
	}
	variables = sortedVariables(variables)
	result := &VariablesResult{
		OK:         true,
		Connection: cfg.Name,
		Pattern:    pattern,
		Variables:  toVariableResults(variables),
	}
	return b.writeOutput(result, func() error {
		if len(variables) == 0 {
			fmt.Fprintln(b.out, "No variables found.")
			return nil
		}
		for _, variable := range variables {
			fmt.Fprintln(b.out, formatVariableLine(variable))
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

func toTableColumnResults(columns []driver.TableColumn) []TableColumnResult {
	results := make([]TableColumnResult, 0, len(columns))
	for _, column := range columns {
		results = append(results, TableColumnResult{
			Name:    column.Name,
			Type:    column.Type,
			Null:    column.Null,
			Key:     column.Key,
			Default: column.Default,
			Extra:   column.Extra,
		})
	}
	return results
}
