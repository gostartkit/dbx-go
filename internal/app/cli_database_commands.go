package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/cmd"
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
	subcommands := []*cmd.Command{
		b.showConnectionCommand(),
		b.showConnectionsCommand(),
		b.showDatabasesCommand(),
		b.showUsersCommand(),
		b.showTablesCommand(),
		b.showTableCommand(),
		b.showColumnsCommand(),
		b.showRowsCommand(),
		b.showTemplatesCommand(),
		b.showContextCommand(),
	}
	return &cmd.Command{
		Name:        "show",
		UsageLine:   "dbx show <subcommand>",
		Short:       "Show resources",
		Long:        helpEntries["show"].body,
		SubCommands: subcommands,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) == 0 {
				usage := "dbx show <subcommand>"
				if b.mode == ModeREPL {
					usage = "show <subcommand>"
				}
				return util.WrapLayer("validation", "show", fmt.Errorf("usage: %s", usage))
			}
			return util.WrapLayer("validation", "show", cmd.UnknownSubcommandError("show", args[0], subcommands))
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
		Name:      "database",
		UsageLine: "dbx create database <name> [flags]",
		Short:     "Create a database from a template",
		Long:      helpEntries["create database"].body,
		Positionals: b.positionalsForMode(
			[]cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true}},
			nil,
		),
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.template, "template", "", "template name", "")
			f.StringVar(&flags.charset, "charset", "utf8mb4", "database charset", "")
			f.StringVar(&flags.collation, "collation", "utf8mb4_unicode_ci", "database collation", "")
			f.BoolVar(&flags.ifNotExists, "if-not-exists", false, "use IF NOT EXISTS when supported by the template", "")
			bindInputFlag(f, flags.inputs)
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				return b.application.handleCreateDatabase(ctx)
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
		Name:      "database",
		UsageLine: "dbx drop database <name> [flags]",
		Short:     "Drop a database from a template",
		Long:      helpEntries["drop database"].body,
		Positionals: b.positionalsForMode(
			[]cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true}},
			nil,
		),
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.template, "template", "", "template name", "")
			bindInputFlag(f, flags.inputs)
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				return b.application.handleDropDatabase(ctx)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "drop database", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runDropDatabase(ctx, application, args[0], flags, meta)
			})
		},
	}
}

func (b *cliBuilder) useGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "use",
		UsageLine:   "dbx use <name>",
		Short:       "Select the current database",
		Long:        helpEntries["use"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true, Completion: b.completeDatabases}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				return b.application.handleUseDatabase(ctx, args[0])
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "use"}, func(application *Application, meta *auditMetadata) error {
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

	result, err := b.executeCLIPlan(ctx, application, cfg, "create database", plan, previewPlan)
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
		return b.writeDryRunPlanResult(ctx, application, cfg, "show databases", plan, previewPlan)
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

	if err := b.requireCLIConfirmation("drop database"); err != nil {
		return err
	}

	result, err := b.executeCLIPlan(ctx, application, cfg, "drop database", plan, previewPlan)
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
