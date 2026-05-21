package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

type createDatabaseFlags struct {
	template    string
	charset     string
	collation   string
	ifNotExists bool
	inputs      inputValues
}

type planOnlyFlags struct {
	template string
	inputs   inputValues
}

func (b *cliBuilder) createGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "create",
		UsageLine: "dbx create <subcommand>",
		Short:     "Create database resources",
		Long:      helpEntries["create"].body,
		SubCommands: []*cmd.Command{
			b.createConnectionCommand(),
			b.createDatabaseCommand(),
			b.createUserCommand(),
		},
	}
}

func (b *cliBuilder) showGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "show",
		UsageLine: "dbx show <subcommand>",
		Short:     "Show resources",
		Long:      helpEntries["show"].body,
		SubCommands: []*cmd.Command{
			b.showConnectionCommand(),
			b.showConnectionsCommand(),
			b.showDatabasesCommand(),
			b.showTablesCommand(),
			b.showTableCommand(),
			b.showColumnsCommand(),
			b.showRowsCommand(),
			b.showTemplatesCommand(),
			b.showContextCommand(),
		},
	}
}

func (b *cliBuilder) dropGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "drop",
		UsageLine: "dbx drop <subcommand>",
		Short:     "Drop resources",
		Long:      helpEntries["drop"].body,
		SubCommands: []*cmd.Command{
			b.dropConnectionCommand(),
			b.dropDatabaseCommand(),
			b.dropUserCommand(),
		},
	}
}

func (b *cliBuilder) showDatabasesCommand() *cmd.Command {
	flags := &planOnlyFlags{inputs: inputValues{}}
	return &cmd.Command{
		Name:      "databases",
		UsageLine: "dbx show databases [flags]",
		Short:     "Show databases on a connection",
		Long:      helpEntries["show databases"].body,
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.template, "template", "", "template name", "")
			bindInputFlag(f, flags.inputs)
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "show databases", err)
				}
				return b.application.handleShowDatabases(ctx)
			}
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "show databases", err)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show databases", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowDatabases(ctx, application, flags, meta)
			})
		},
	}
}

func (b *cliBuilder) createDatabaseCommand() *cmd.Command {
	flags := &createDatabaseFlags{
		charset:   "utf8mb4",
		collation: "utf8mb4_unicode_ci",
		inputs:    inputValues{},
	}
	return &cmd.Command{
		Name:        "database",
		UsageLine:   "dbx create database <name> [flags]",
		Short:       "Create a database from a template",
		Long:        helpEntries["create database"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true}},
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.template, "template", "", "template name", "")
			f.StringVar(&flags.charset, "charset", "utf8mb4", "database charset", "")
			f.StringVar(&flags.collation, "collation", "utf8mb4_unicode_ci", "database collation", "")
			f.BoolVar(&flags.ifNotExists, "if-not-exists", false, "use IF NOT EXISTS when supported by the template", "")
			bindInputFlag(f, flags.inputs)
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 0 {
					return util.WrapLayer("validation", "create database", fmt.Errorf("usage: create database"))
				}
				return b.application.handleCreateDatabase(ctx)
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "create database", fmt.Errorf("usage: dbx create database <name> [flags]"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "create database", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runCreateDatabase(ctx, application, args[0], flags, meta)
			})
		},
	}
}

func (b *cliBuilder) dropDatabaseCommand() *cmd.Command {
	flags := &planOnlyFlags{inputs: inputValues{}}
	return &cmd.Command{
		Name:        "database",
		UsageLine:   "dbx drop database <name> [flags]",
		Short:       "Drop a database from a template",
		Long:        helpEntries["drop database"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true}},
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.template, "template", "", "template name", "")
			bindInputFlag(f, flags.inputs)
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 0 {
					return util.WrapLayer("validation", "drop database", fmt.Errorf("usage: drop database"))
				}
				return b.application.handleDropDatabase(ctx)
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "drop database", fmt.Errorf("usage: dbx drop database <name> [flags]"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "drop database", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runDropDatabase(ctx, application, args[0], flags, meta)
			})
		},
	}
}

func (b *cliBuilder) useGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "use",
		UsageLine: "dbx use <subcommand>",
		Short:     "Select session-scoped resources",
		Long:      helpEntries["use"].body,
		SubCommands: []*cmd.Command{
			b.useDatabaseCommand(),
		},
	}
}

func (b *cliBuilder) useDatabaseCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "database",
		UsageLine:   "dbx use database <name>",
		Short:       "Select the current database",
		Long:        helpEntries["use"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true, Completion: b.completeDatabases}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "use database", fmt.Errorf("usage: dbx use database <name>"))
			}
			if b.mode == ModeREPL {
				return b.application.handleUseDatabase(ctx, args[0])
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "use database"}, func(application *Application, meta *auditMetadata) error {
				result, err := application.useDatabaseForCLI(ctx, b.globals.Connection, args[0])
				if err != nil {
					return err
				}
				meta.Connection = result.Connection
				return b.writeOutput(result, func() error {
					fmt.Fprintf(b.out, "Using database: %s\n", result.Database)
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) runCreateDatabase(ctx context.Context, application *Application, name string, flags *createDatabaseFlags, meta *auditMetadata) error {
	if err := util.ValidateDatabaseName(name); err != nil {
		return util.WrapLayer("validation", "validate database name", err)
	}

	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if err := application.applyCLIDatabaseSelection(ctx, cfg, b.globals.Database); err != nil {
		return err
	}

	selectedTemplate, err := application.selectTemplateForCLI("create database", cfg, flags.template)
	if err != nil {
		return err
	}

	values := cloneInputValues(flags.inputs)
	values["database"] = name
	values["charset"] = flags.charset
	values["collation"] = flags.collation
	if flags.ifNotExists {
		values["if_not_exists"] = "true"
		values["if_not_exists_clause"] = "IF NOT EXISTS"
	}

	mergedValues, err := mergeTemplateInputs(selectedTemplate, values, true)
	if err != nil {
		return util.WrapLayer("template", "collect template inputs", err)
	}

	plan, previewPlan, err := buildPlans(selectedTemplate, cfg, mergedValues)
	if err != nil {
		return err
	}
	if !strings.EqualFold(b.globals.Format, "json") {
		application.printPlanPreview(previewPlan, b.globals.DryRun)
	}

	if err := b.requireCLIConfirmation("create database"); err != nil {
		return err
	}

	var result *PlanExecutionResult
	if b.globals.DryRun {
		result, err = application.runPlan(ctx, plan, noopTransactionStarter{}, true)
	} else {
		db, openErr := application.openConnection(ctx, cfg)
		if openErr != nil {
			return openErr
		}
		defer db.Close()

		result, err = application.runPlan(ctx, plan, sqlRunner{db: db}, false)
	}
	if result != nil {
		result.Connection = cfg.Name
		result.Command = "create database"
		applyPreviewSQL(result, previewPlan)
	}
	if !strings.EqualFold(b.globals.Format, "json") {
		application.printPlanResult(result)
	}
	if err != nil {
		if strings.EqualFold(b.globals.Format, "json") && result != nil {
			result.Error = errorResult(err)
			if writeErr := b.writeOutput(result, func() error { return nil }); writeErr != nil {
				return writeErr
			}
			return util.MarkOutputHandled(err)
		}
		return err
	}

	return b.writeOutput(result, func() error {
		fmt.Fprintf(b.out, "Database %s created.\n", name)
		return nil
	})
}

func (b *cliBuilder) runShowDatabases(ctx context.Context, application *Application, flags *planOnlyFlags, meta *auditMetadata) error {
	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if err := application.applyCLIDatabaseSelection(ctx, cfg, b.globals.Database); err != nil {
		return err
	}

	selectedTemplate, err := application.selectTemplateForCLI("show databases", cfg, flags.template)
	if err != nil {
		return err
	}
	values, err := mergeTemplateInputs(selectedTemplate, cloneInputValues(flags.inputs), true)
	if err != nil {
		return util.WrapLayer("template", "collect template inputs", err)
	}
	plan, previewPlan, err := buildPlans(selectedTemplate, cfg, values)
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		result, runErr := application.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "show databases"
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

	databases, err := application.connector.QueryStrings(ctx, cfg, db, plan.Actions[0].SQL)
	if err != nil {
		return err
	}
	result := &DatabasesResult{
		OK:         true,
		Connection: cfg.Name,
		Databases:  databases,
	}
	return b.writeOutput(result, func() error {
		if len(databases) == 0 {
			fmt.Fprintln(b.out, "No databases found.")
			return nil
		}
		fmt.Fprintln(b.out, "Databases:")
		for _, name := range databases {
			fmt.Fprintf(b.out, "  - %s\n", name)
		}
		return nil
	})
}

func (b *cliBuilder) runDropDatabase(ctx context.Context, application *Application, name string, flags *planOnlyFlags, meta *auditMetadata) error {
	if err := util.ValidateDatabaseName(name); err != nil {
		return util.WrapLayer("validation", "validate database name", err)
	}
	if err := b.requireCLIConfirmation("drop database"); err != nil {
		return err
	}

	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if err := application.applyCLIDatabaseSelection(ctx, cfg, b.globals.Database); err != nil {
		return err
	}
	selectedTemplate, err := application.selectTemplateForCLI("drop database", cfg, flags.template)
	if err != nil {
		return err
	}

	values := cloneInputValues(flags.inputs)
	values["database"] = name
	mergedValues, err := mergeTemplateInputs(selectedTemplate, values, true)
	if err != nil {
		return util.WrapLayer("template", "collect template inputs", err)
	}
	plan, previewPlan, err := buildPlans(selectedTemplate, cfg, mergedValues)
	if err != nil {
		return err
	}

	if !strings.EqualFold(b.globals.Format, "json") {
		application.printPlanPreview(previewPlan, b.globals.DryRun)
	}

	var result *PlanExecutionResult
	if b.globals.DryRun {
		result, err = application.runPlan(ctx, plan, noopTransactionStarter{}, true)
	} else {
		db, openErr := application.openConnection(ctx, cfg)
		if openErr != nil {
			return openErr
		}
		defer db.Close()
		result, err = application.runPlan(ctx, plan, sqlRunner{db: db}, false)
	}
	if result != nil {
		result.Connection = cfg.Name
		result.Command = "drop database"
		applyPreviewSQL(result, previewPlan)
	}
	if !strings.EqualFold(b.globals.Format, "json") {
		application.printPlanResult(result)
	}
	if err != nil {
		if strings.EqualFold(b.globals.Format, "json") && result != nil {
			result.Error = errorResult(err)
			if writeErr := b.writeOutput(result, func() error { return nil }); writeErr != nil {
				return writeErr
			}
			return util.MarkOutputHandled(err)
		}
		return err
	}
	return b.writeOutput(result, func() error {
		if !b.globals.DryRun {
			fmt.Fprintf(b.out, "Database %s dropped.\n", name)
		}
		return nil
	})
}

func buildPlans(template *tpl.Template, cfg *config.ConnectionConfig, values map[string]string) (*tpl.ExecutionPlan, *tpl.ExecutionPlan, error) {
	plan, err := tpl.BuildPlan(template, cfg, values)
	if err != nil {
		return nil, nil, util.WrapLayer("template", "build execution plan", err)
	}
	previewPlan, err := tpl.BuildPlan(template, cfg, redactTemplateValues(template, values))
	if err != nil {
		return nil, nil, util.WrapLayer("template", "build redacted execution preview", err)
	}
	return plan, previewPlan, nil
}

func cloneInputValues(values inputValues) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func applyPreviewSQL(result *PlanExecutionResult, previewPlan *tpl.ExecutionPlan) {
	if result == nil || previewPlan == nil {
		return
	}

	for index := range result.Actions {
		if index >= len(previewPlan.Actions) {
			return
		}
		result.Actions[index].SQL = previewPlan.Actions[index].SQL
	}
}

type noopTransactionStarter struct{}

func (noopTransactionStarter) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, fmt.Errorf("dry-run execution does not use a SQL runner")
}

func (noopTransactionStarter) BeginTx(context.Context, *sql.TxOptions) (transaction, error) {
	return nil, fmt.Errorf("dry-run execution does not start a transaction")
}
