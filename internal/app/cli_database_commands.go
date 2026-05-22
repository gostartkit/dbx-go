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
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "create",
		UsageFallback: "dbx create <subcommand>",
		ShortFallback: "Create database resources",
		SubCommands: []*cmd.Command{
			b.createConnectionCommand(),
			b.createDatabaseCommand(),
			b.createUserCommand(),
		},
	})
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
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "show",
		UsageFallback: "dbx show <subcommand>",
		ShortFallback: "Show resources",
		SubCommands:   subcommands,
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
	})
}

func (b *cliBuilder) dropGroupCommand() *cmd.Command {
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "drop",
		UsageFallback: "dbx drop <subcommand>",
		ShortFallback: "Drop resources",
		SubCommands: []*cmd.Command{
			b.dropConnectionCommand(),
			b.dropDatabaseCommand(),
			b.dropUserCommand(),
		},
	})
}

func (b *cliBuilder) showDatabasesCommand() *cmd.Command {
	flags := &planOnlyFlags{inputs: inputValues{}}
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "show databases",
		UsageFallback: "dbx show databases [flags]",
		ShortFallback: "Show databases on a connection",
		SetFlags: func(f *cmd.FlagSet) {
			b.bindManifestStringFlag(f, "show databases", "template", &flags.template, "", "template name")
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
	})
}

func (b *cliBuilder) createDatabaseCommand() *cmd.Command {
	flags := &createDatabaseFlags{
		charset:   "utf8mb4",
		collation: "utf8mb4_unicode_ci",
		inputs:    inputValues{},
	}
	flags.charset = manifestFlagDefaultString("create database", "charset", flags.charset)
	flags.collation = manifestFlagDefaultString("create database", "collation", flags.collation)
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "create database",
		Name:          "database",
		UsageFallback: "dbx create database <name> [flags]",
		ShortFallback: "Create a database from a template",
		Positionals: b.positionalsForMode(
			[]cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true}},
			nil,
		),
		SetFlags: func(f *cmd.FlagSet) {
			b.bindManifestStringFlag(f, "create database", "template", &flags.template, "", "template name")
			b.bindManifestStringFlag(f, "create database", "charset", &flags.charset, "utf8mb4", "database charset")
			b.bindManifestStringFlag(f, "create database", "collation", &flags.collation, "utf8mb4_unicode_ci", "database collation")
			b.bindManifestBoolFlag(f, "create database", "if-not-exists", &flags.ifNotExists, false, "use IF NOT EXISTS when supported by the template")
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
	})
}

func (b *cliBuilder) dropDatabaseCommand() *cmd.Command {
	flags := &planOnlyFlags{inputs: inputValues{}}
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "drop database",
		Name:          "database",
		UsageFallback: "dbx drop database <name> [flags]",
		ShortFallback: "Drop a database from a template",
		Positionals: b.positionalsForMode(
			[]cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true}},
			nil,
		),
		SetFlags: func(f *cmd.FlagSet) {
			b.bindManifestStringFlag(f, "drop database", "template", &flags.template, "", "template name")
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
	})
}

func (b *cliBuilder) useGroupCommand() *cmd.Command {
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "use",
		UsageFallback: "dbx use <name>",
		ShortFallback: "Select the current database",
		Positionals:   b.manifestPositionals("use", []cmd.PositionalArg{{Name: "name", Usage: "database name", Required: true, Completion: b.completeDatabases}}),
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
	})
}

func (b *cliBuilder) runCreateDatabase(ctx context.Context, application *Application, name string, flags *createDatabaseFlags, meta *auditMetadata) error {
	if err := util.ValidateDatabaseName(name); err != nil {
		return util.WrapLayer("validation", "validate database name", err)
	}

	cfg, err := application.commandContext().resolveCLIConnection(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if err := application.commandContext().applyCLIDatabaseSelection(ctx, cfg, b.globals.Database); err != nil {
		return err
	}

	selectedTemplate, err := application.commandContext().selectCLITemplate("create database", cfg, flags.template)
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
	cfg, err := application.commandContext().resolveCLIConnection(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if err := application.commandContext().applyCLIDatabaseSelection(ctx, cfg, b.globals.Database); err != nil {
		return err
	}

	selectedTemplate, err := application.commandContext().selectCLITemplate("show databases", cfg, flags.template)
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

	cfg, err := application.commandContext().resolveCLIConnection(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if err := application.commandContext().applyCLIDatabaseSelection(ctx, cfg, b.globals.Database); err != nil {
		return err
	}
	selectedTemplate, err := application.commandContext().selectCLITemplate("drop database", cfg, flags.template)
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
