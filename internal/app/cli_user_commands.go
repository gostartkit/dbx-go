package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

type createUserFlags struct {
	template         string
	host             string
	password         string
	passwordEnv      string
	generatePassword bool
	grant            string
	inputs           inputValues
}

type dropUserFlags struct {
	template string
	host     string
	inputs   inputValues
}

func (b *cliBuilder) showUsersCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "users",
		UsageLine: "dbx show users",
		Short:     "Show MySQL users.",
		Long:      commandLong("show users"),
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "show users", err)
			}
			if b.mode == ModeREPL {
				return b.application.handleShowUsers(ctx)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show users", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowUsers(ctx, application, meta)
			})
		},
	}
}

func (b *cliBuilder) createUserCommand() *cmd.Command {
	flags := &createUserFlags{
		host:   "%",
		inputs: inputValues{},
	}
	return &cmd.Command{
		Name:      "user",
		UsageLine: "dbx create user <name> [flags]",
		Short:     "Create a MySQL user.",
		Long:      commandLong("create user"),
		Positionals: b.positionalsForMode(
			[]cmd.PositionalArg{userPositional(true, b.completeUsers)},
			[]cmd.PositionalArg{userPositional(false, b.completeUsers)},
		),
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.template, "template", "", "Template name.", "")
			f.SetCompletion("template", b.completeTemplates)
			f.SetCompletionKey("template", "template")
			f.StringVar(&flags.host, "host", "%", "MySQL user host.", "")
			f.StringVar(&flags.password, "password", "", "MySQL user password.", "")
			f.StringVar(&flags.passwordEnv, "password-env", "", "Environment variable containing the MySQL user password.", "")
			f.BoolVar(&flags.generatePassword, "generate-password", false, "Generate a password automatically.", "")
			f.StringVar(&flags.grant, "grant", "", "Database grant mode.", "")
			f.SetEnum("grant", "all", "readonly")
			bindInputFlag(f, flags.inputs)
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				name := ""
				if len(args) == 1 {
					name = args[0]
				}
				return b.application.handleCreateUser(ctx, name)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "create user", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runCreateUser(ctx, application, args[0], flags, meta)
			})
		},
	}
}

func (b *cliBuilder) dropUserCommand() *cmd.Command {
	flags := &dropUserFlags{
		host:   "%",
		inputs: inputValues{},
	}
	return &cmd.Command{
		Name:      "user",
		UsageLine: "dbx drop user <name> [flags]",
		Short:     "Drop a MySQL user.",
		Long:      commandLong("drop user"),
		Positionals: b.positionalsForMode(
			[]cmd.PositionalArg{userPositional(true, b.completeUsers)},
			[]cmd.PositionalArg{userPositional(false, b.completeUsers)},
		),
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.template, "template", "", "Template name.", "")
			f.SetCompletion("template", b.completeTemplates)
			f.SetCompletionKey("template", "template")
			f.StringVar(&flags.host, "host", "%", "MySQL user host.", "")
			bindInputFlag(f, flags.inputs)
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				name := ""
				if len(args) == 1 {
					name = args[0]
				}
				return b.application.handleDropUser(ctx, name)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "drop user", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runDropUser(ctx, application, args[0], flags, meta)
			})
		},
	}
}

func (b *cliBuilder) runShowUsers(ctx context.Context, application *Application, meta *auditMetadata) error {
	cfg, err := application.commandContext().resolveCLIConnection(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	template, err := application.commandContext().selectCLITemplate("show users", cfg, "")
	if err != nil {
		return err
	}
	plan, previewPlan, err := buildPlans(template, cfg, map[string]string{})
	if err != nil {
		return err
	}

	if b.globals.DryRun {
		return b.writeDryRunPlanResult(ctx, application, cfg, "show users", plan, previewPlan)
	}

	db, err := application.openConnection(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	users, err := application.connector.QueryStrings(ctx, cfg, db, plan.Actions[0].SQL)
	if err != nil {
		return err
	}
	result := &UsersResult{
		OK:         true,
		Connection: cfg.Name,
		Users:      users,
	}
	return b.writeOutput(result, func() error {
		if len(users) == 0 {
			fmt.Fprintln(b.out, "No users found.")
			return nil
		}
		fmt.Fprintln(b.out, "Users:")
		for _, user := range users {
			fmt.Fprintf(b.out, "  - %s\n", user)
		}
		return nil
	})
}

func (b *cliBuilder) runCreateUser(ctx context.Context, application *Application, username string, flags *createUserFlags, meta *auditMetadata) error {
	if err := util.ValidateMySQLUsername(username); err != nil {
		return util.WrapLayer("validation", "validate MySQL username", err)
	}
	host := defaultUserHost(flags.host)
	if err := validateUserHost(host); err != nil {
		return util.WrapLayer("validation", "validate MySQL user host", err)
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

	grant := strings.TrimSpace(flags.grant)
	if grant != "" && strings.TrimSpace(application.session.Database) == "" {
		return util.WrapLayer("validation", "create user", fmt.Errorf("grant requires --database or an active database context"))
	}

	selectedTemplate, err := application.commandContext().selectCLITemplate("create user", cfg, flags.template)
	if err != nil {
		return err
	}

	password, generated, err := resolveCLIPassword(ctx, application, flags)
	if err != nil {
		return err
	}

	plan, previewPlan, err := application.buildCreateUserCLIPlansWithTemplate(cfg, selectedTemplate, userCreateOptions{
		Username:          username,
		Host:              host,
		Password:          password,
		PasswordGenerated: generated,
		Grant:             grant,
		Database:          application.session.Database,
	}, cloneInputValues(flags.inputs))
	if err != nil {
		return err
	}

	if !strings.EqualFold(b.globals.Format, "json") {
		application.printPlanPreview(previewPlan, b.globals.DryRun)
	}

	if err := b.requireCLIConfirmation("create user"); err != nil {
		return err
	}

	result, err := b.executeCLIPlan(ctx, application, cfg, "create user", plan, previewPlan)
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

	application.clearUserCompletion()
	if b.globals.DryRun {
		return b.writeOutput(result, func() error {
			fmt.Fprintln(b.out, "Dry-run completed.")
			return nil
		})
	}
	mutation := &UserMutationResult{
		OK:       true,
		User:     username,
		Host:     host,
		Grant:    grant,
		Database: application.session.Database,
	}
	return b.writeOutput(mutation, func() error {
		if generated && !b.globals.DryRun {
			printGeneratedPassword(application.prompt, password)
		}
		if b.globals.DryRun {
			fmt.Fprintln(b.out, "Dry-run completed.")
			return nil
		}
		fmt.Fprintf(b.out, "User %s@%s created.\n", username, host)
		if grant != "" && application.session.Database != "" {
			fmt.Fprintf(b.out, "Grant applied on %s.\n", application.session.Database)
		}
		return nil
	})
}

func (b *cliBuilder) runDropUser(ctx context.Context, application *Application, username string, flags *dropUserFlags, meta *auditMetadata) error {
	if err := util.ValidateMySQLUsername(username); err != nil {
		return util.WrapLayer("validation", "validate MySQL username", err)
	}
	host := defaultUserHost(flags.host)
	if err := validateUserHost(host); err != nil {
		return util.WrapLayer("validation", "validate MySQL user host", err)
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

	selectedTemplate, err := application.commandContext().selectCLITemplate("drop user", cfg, flags.template)
	if err != nil {
		return err
	}
	values := buildDropUserValues(userDropOptions{Username: username, Host: host})
	for key, value := range cloneInputValues(flags.inputs) {
		values[key] = value
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

	if err := b.requireCLIConfirmation("drop user"); err != nil {
		return err
	}

	result, err := b.executeCLIPlan(ctx, application, cfg, "drop user", plan, previewPlan)
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

	application.clearUserCompletion()
	if b.globals.DryRun {
		return b.writeOutput(result, func() error {
			fmt.Fprintln(b.out, "Dry-run completed.")
			return nil
		})
	}
	mutation := &UserMutationResult{
		OK:   true,
		User: username,
		Host: host,
	}
	return b.writeOutput(mutation, func() error {
		if b.globals.DryRun {
			fmt.Fprintln(b.out, "Dry-run completed.")
			return nil
		}
		fmt.Fprintf(b.out, "User %s@%s dropped.\n", username, host)
		return nil
	})
}

func resolveCLIPassword(ctx context.Context, application *Application, flags *createUserFlags) (string, bool, error) {
	sources := 0
	if strings.TrimSpace(flags.password) != "" {
		sources++
	}
	if strings.TrimSpace(flags.passwordEnv) != "" {
		sources++
	}
	if flags.generatePassword {
		sources++
	}
	if sources > 1 {
		return "", false, util.WrapLayer("validation", "create user", fmt.Errorf("choose only one of --password, --password-env, or --generate-password"))
	}

	switch {
	case strings.TrimSpace(flags.password) != "":
		return flags.password, false, nil
	case strings.TrimSpace(flags.passwordEnv) != "":
		password, err := resolvePasswordFromEnv(flags.passwordEnv)
		if err != nil {
			return "", false, err
		}
		return password, false, nil
	case flags.generatePassword:
		password, err := util.GeneratePassword(20)
		if err != nil {
			return "", false, util.WrapLayer("validation", "generate password", err)
		}
		return password, true, nil
	default:
		password, err := application.askPassword(ctx, "Password")
		if err != nil {
			return "", false, err
		}
		if strings.TrimSpace(password) == "" {
			return "", false, util.WrapLayer("validation", "validate password", fmt.Errorf("password is required"))
		}
		return password, false, nil
	}
}

func (a *Application) buildCreateUserCLIPlans(cfg *config.ConnectionConfig, templateName string, options userCreateOptions, extraInputs map[string]string) (*tpl.ExecutionPlan, *tpl.ExecutionPlan, error) {
	selectedTemplate, err := a.commandContext().selectCLITemplate("create user", cfg, templateName)
	if err != nil {
		return nil, nil, err
	}
	return a.buildCreateUserCLIPlansWithTemplate(cfg, selectedTemplate, options, extraInputs)
}

func (a *Application) buildCreateUserCLIPlansWithTemplate(cfg *config.ConnectionConfig, selectedTemplate *tpl.Template, options userCreateOptions, extraInputs map[string]string) (*tpl.ExecutionPlan, *tpl.ExecutionPlan, error) {
	values := buildCreateUserValues(options)
	for key, value := range extraInputs {
		values[key] = value
	}
	merged, err := mergeTemplateInputs(selectedTemplate, values, true)
	if err != nil {
		return nil, nil, util.WrapLayer("template", "collect template inputs", err)
	}

	plan, err := tpl.BuildPlan(selectedTemplate, cfg, merged)
	if err != nil {
		return nil, nil, util.WrapLayer("template", "build execution plan", err)
	}
	previewValues := redactTemplateValues(selectedTemplate, merged)
	if _, exists := previewValues["password"]; exists {
		previewValues["password"] = "***"
	}
	previewPlan, err := tpl.BuildPlan(selectedTemplate, cfg, previewValues)
	if err != nil {
		return nil, nil, util.WrapLayer("template", "build redacted execution preview", err)
	}
	return plan, previewPlan, nil
}
