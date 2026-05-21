package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleShowUsers(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show users", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		template, err := a.resolveTemplateForAction(ctx, "show users", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show users template", err)
		}

		plan, previewPlan, err := buildPlans(template, cfg, map[string]string{})
		if err != nil {
			return err
		}

		if a.dryRun {
			a.printPlanPreview(previewPlan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "show users",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		users, err := a.connector.QueryStrings(ctx, cfg, db, plan.Actions[0].SQL)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			a.prompt.Println("No users found.")
			return nil
		}
		a.prompt.Println("Users:")
		for _, user := range users {
			a.prompt.Printf("  - %s\n", user)
		}
		return nil
	})
}

func (a *Application) handleShowUser(ctx context.Context, username string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show user", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		username, err = a.resolveUsernameInput(ctx, username)
		if err != nil {
			return err
		}

		template, err := a.resolveTemplateForAction(ctx, "show user", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve show user template", err)
		}

		values := map[string]string{"username": username}
		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		if a.dryRun {
			a.printPlanPreview(previewPlan, true)
			a.printPlanResult(&PlanExecutionResult{
				OK:         true,
				Connection: cfg.Name,
				Command:    "show user",
				DryRun:     true,
				Actions: []ActionResult{{
					Description: plan.Actions[0].Description,
					SQL:         previewPlan.Actions[0].SQL,
					Status:      ActionStatusDryRun,
				}},
			})
			return nil
		}

		users, err := a.connector.QueryStrings(ctx, cfg, db, plan.Actions[0].SQL)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			a.prompt.Printf("User %s not found.\n", username)
			return nil
		}
		a.prompt.Printf("User %s:\n", username)
		for _, user := range users {
			a.prompt.Printf("  - %s\n", user)
		}
		return nil
	})
}

func (a *Application) handleCreateUser(ctx context.Context, username string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "create user", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		username, err = a.resolveUsernameInput(ctx, username)
		if err != nil {
			return err
		}

		host, err := a.ask(ctx, "Host", "%")
		if err != nil {
			return err
		}
		host = defaultUserHost(host)
		if err := validateUserHost(host); err != nil {
			return util.WrapLayer("validation", "validate MySQL user host", err)
		}

		password, generated, err := a.promptCreateUserPassword(ctx)
		if err != nil {
			return err
		}

		grant, grantDatabase, err := a.promptUserGrant(ctx, a.session.Database)
		if err != nil {
			return err
		}

		plan, previewPlan, err := a.buildCreateUserPlans(ctx, cfg, userCreateOptions{
			Username:          username,
			Host:              host,
			Password:          password,
			PasswordGenerated: generated,
			Grant:             grant,
			Database:          grantDatabase,
		})
		if err != nil {
			return err
		}

		confirmed, err := a.previewAndConfirm(ctx, "create user", previewPlan)
		if err != nil {
			return err
		}
		if !confirmed {
			a.prompt.Println("Cancelled.")
			return nil
		}

		result, err := a.runPlan(ctx, plan, sqlRunner{db: db}, a.dryRun)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "create user"
			applyPreviewSQL(result, previewPlan)
		}
		a.printPlanResult(result)
		if err != nil {
			return err
		}

		a.clearUserCompletion()
		if generated && !a.dryRun {
			printGeneratedPassword(a.prompt, password)
		}
		if a.dryRun {
			a.prompt.Println("Dry-run completed.")
			return nil
		}
		a.prompt.Printf("User %s@%s created.\n", username, host)
		if grant != "" && grantDatabase != "" {
			a.prompt.Printf("Grant applied on %s.\n", grantDatabase)
		}
		return nil
	})
}

func (a *Application) handleDropUser(ctx context.Context, username string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "drop user", DryRun: a.dryRun}, func(meta *auditMetadata) error {
		cfg, db, err := a.requireConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		username, err = a.resolveUsernameInput(ctx, username)
		if err != nil {
			return err
		}

		host, err := a.ask(ctx, "Host", "%")
		if err != nil {
			return err
		}
		host = defaultUserHost(host)
		if err := validateUserHost(host); err != nil {
			return util.WrapLayer("validation", "validate MySQL user host", err)
		}

		template, err := a.resolveTemplateForAction(ctx, "drop user", cfg)
		if err != nil {
			return util.WrapLayer("template", "resolve drop user template", err)
		}

		values := buildDropUserValues(userDropOptions{Username: username, Host: host})
		if err := a.collectTemplateInputs(ctx, template, values); err != nil {
			return util.WrapLayer("template", "collect template inputs", err)
		}

		plan, previewPlan, err := buildPlans(template, cfg, values)
		if err != nil {
			return err
		}

		confirmed, err := a.previewAndConfirm(ctx, "drop user", previewPlan)
		if err != nil {
			return err
		}
		if !confirmed {
			a.prompt.Println("Cancelled.")
			return nil
		}

		result, err := a.runPlan(ctx, plan, sqlRunner{db: db}, a.dryRun)
		if result != nil {
			result.Connection = cfg.Name
			result.Command = "drop user"
			applyPreviewSQL(result, previewPlan)
		}
		a.printPlanResult(result)
		if err != nil {
			return err
		}

		a.clearUserCompletion()
		if a.dryRun {
			a.prompt.Println("Dry-run completed.")
			return nil
		}
		a.prompt.Printf("User %s@%s dropped.\n", username, host)
		return nil
	})
}

func (a *Application) resolveUsernameInput(ctx context.Context, username string) (string, error) {
	if strings.TrimSpace(username) == "" {
		value, err := a.ask(ctx, "Username", "")
		if err != nil {
			return "", err
		}
		username = value
	}
	if err := util.ValidateMySQLUsername(username); err != nil {
		return "", util.WrapLayer("validation", "validate MySQL username", err)
	}
	return strings.TrimSpace(username), nil
}

func (a *Application) promptCreateUserPassword(ctx context.Context) (string, bool, error) {
	mode, err := a.choose(ctx, "Password mode", []string{"prompt", "env variable", "generated password"}, "prompt")
	if err != nil {
		return "", false, err
	}

	switch mode {
	case "env variable":
		envName, askErr := a.ask(ctx, "Environment variable name", "")
		if askErr != nil {
			return "", false, askErr
		}
		password, resolveErr := resolvePasswordFromEnv(envName)
		if resolveErr != nil {
			return "", false, resolveErr
		}
		return password, false, nil
	case "generated password":
		password, generateErr := util.GeneratePassword(20)
		if generateErr != nil {
			return "", false, util.WrapLayer("validation", "generate password", generateErr)
		}
		return password, true, nil
	default:
		password, askErr := a.askPassword(ctx, "Password")
		if askErr != nil {
			return "", false, askErr
		}
		if strings.TrimSpace(password) == "" {
			return "", false, util.WrapLayer("validation", "validate password", fmt.Errorf("password is required"))
		}
		return password, false, nil
	}
}

func (a *Application) buildCreateUserPlans(ctx context.Context, cfg *config.ConnectionConfig, options userCreateOptions) (*tpl.ExecutionPlan, *tpl.ExecutionPlan, error) {
	template, err := a.resolveTemplateForAction(ctx, "create user", cfg)
	if err != nil {
		return nil, nil, util.WrapLayer("template", "resolve create user template", err)
	}

	values := buildCreateUserValues(options)
	if err := a.collectTemplateInputs(ctx, template, values); err != nil {
		return nil, nil, util.WrapLayer("template", "collect template inputs", err)
	}

	plan, err := tpl.BuildPlan(template, cfg, values)
	if err != nil {
		return nil, nil, util.WrapLayer("template", "build execution plan", err)
	}

	previewValues := redactTemplateValues(template, values)
	if _, exists := previewValues["password"]; exists {
		previewValues["password"] = "***"
	}
	previewPlan, err := tpl.BuildPlan(template, cfg, previewValues)
	if err != nil {
		return nil, nil, util.WrapLayer("template", "build redacted execution preview", err)
	}
	return plan, previewPlan, nil
}
