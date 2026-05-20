package app

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleShowTemplates(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "show templates"}, func(meta *auditMetadata) error {
		cfg, err := a.templateScopeConfig("")
		if err != nil {
			return err
		}
		if cfg != nil && strings.TrimSpace(cfg.Name) != "" {
			meta.Connection = cfg.Name
			meta.Mode = cfg.Mode
		}

		result, err := a.showTemplatesResult(cfg)
		if err != nil {
			return err
		}
		a.printTemplatesCatalog(result)
		return nil
	})
}

func (a *Application) handleDescribeTemplate(ctx context.Context, name string, verbose bool) error {
	return a.auditCommand(ctx, auditMetadata{Command: "describe template"}, func(meta *auditMetadata) error {
		cfg, err := a.templateScopeConfig("")
		if err != nil {
			return err
		}
		if cfg != nil && strings.TrimSpace(cfg.Name) != "" {
			meta.Connection = cfg.Name
			meta.Mode = cfg.Mode
		}

		result, err := a.describeTemplateResult(cfg, name, verbose)
		if err != nil {
			return err
		}
		a.printTemplateDescription(result, verbose)
		return nil
	})
}

func (a *Application) handleTemplateRun(ctx context.Context, name string, preview bool, verbose bool, dryRunFlag bool) error {
	effectiveDryRun := a.dryRun || dryRunFlag
	return a.auditCommand(ctx, auditMetadata{Command: "template run", DryRun: effectiveDryRun || preview}, func(meta *auditMetadata) error {
		cfg, err := a.requireTemplateRunConnection(ctx)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		selectedTemplate, err := a.templates.ResolveNamedAny(cfg, name)
		if err != nil {
			return util.WrapLayer("template", "resolve template "+name, err)
		}

		values, err := a.collectTemplateRunInputs(ctx, selectedTemplate, nil, true, a.currentDatabaseName())
		if err != nil {
			return util.WrapLayer("template", "collect template inputs", err)
		}

		plan, previewPlan, err := buildPlans(selectedTemplate, cfg, values)
		if err != nil {
			return err
		}

		redactedInputs := redactTemplateValues(selectedTemplate, values)
		a.printTemplateRunPreview(previewPlan, redactedInputs, verbose, preview, effectiveDryRun)

		if preview {
			return nil
		}

		if effectiveDryRun {
			result, runErr := a.runPlan(ctx, plan, noopTransactionStarter{}, true)
			a.printPlanResult(result)
			return runErr
		}

		confirmed, err := a.confirmExecutionIfNeeded(ctx, "template run")
		if err != nil {
			return err
		}
		if !confirmed {
			a.prompt.Println("Cancelled.")
			return nil
		}

		db, err := a.openConnection(ctx, cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		result, runErr := a.executePlan(ctx, plan, sqlRunner{db: db})
		a.printPlanResult(result)
		return runErr
	})
}

func (a *Application) showTemplatesResult(cfg *config.ConnectionConfig) (*TemplatesCatalogResult, error) {
	templates, err := a.templates.ListResolved(cfg)
	if err != nil {
		return nil, util.WrapLayer("template", "list templates", err)
	}

	result := &TemplatesCatalogResult{
		OK:        true,
		Templates: make([]TemplateSummaryResult, 0, len(templates)),
	}
	if cfg != nil && strings.TrimSpace(cfg.Name) != "" {
		result.Connection = cfg.Name
	}
	for _, candidate := range templates {
		result.Templates = append(result.Templates, TemplateSummaryResult{
			Name:        candidate.Name,
			Scope:       candidate.Layer,
			Command:     candidate.Match.Command,
			Description: candidate.Description,
		})
	}
	return result, nil
}

func (a *Application) describeTemplateResult(cfg *config.ConnectionConfig, name string, verbose bool) (*TemplateDescriptionResult, error) {
	selectedTemplate, err := a.templates.ResolveNamedAny(cfg, name)
	if err != nil {
		return nil, util.WrapLayer("template", "resolve template "+name, err)
	}

	result := &TemplateDescriptionResult{
		OK:          true,
		Name:        selectedTemplate.Name,
		Scope:       selectedTemplate.Layer,
		Command:     selectedTemplate.Match.Command,
		Description: selectedTemplate.Description,
		Transaction: selectedTemplate.Transaction,
		Inputs:      make([]TemplateInputResult, 0, len(selectedTemplate.Inputs)),
		Actions:     make([]TemplateActionSummaryResult, 0, len(selectedTemplate.Actions)),
	}
	if cfg != nil && strings.TrimSpace(cfg.Name) != "" {
		result.Connection = cfg.Name
	}

	for _, input := range selectedTemplate.Inputs {
		result.Inputs = append(result.Inputs, TemplateInputResult{
			Name:        input.Name,
			Type:        input.EffectiveType(),
			Description: input.Description,
			Required:    input.IsRequired(),
		})
	}
	for _, action := range selectedTemplate.Actions {
		entry := TemplateActionSummaryResult{
			Description: action.Description,
			Type:        action.Type,
		}
		if verbose {
			entry.SQL = redactTemplateActionSQL(selectedTemplate, action.SQL)
		}
		result.Actions = append(result.Actions, entry)
	}
	return result, nil
}

func (a *Application) templateRunResult(ctx context.Context, cfg *config.ConnectionConfig, name string, rawInputs map[string]string, preview bool, dryRun bool, verbose bool, database string) (*TemplateRunResult, error) {
	selectedTemplate, err := a.templates.ResolveNamedAny(cfg, name)
	if err != nil {
		return nil, util.WrapLayer("template", "resolve template "+name, err)
	}

	values, err := a.collectTemplateRunInputs(nil, selectedTemplate, rawInputs, true, database)
	if err != nil {
		return nil, util.WrapLayer("template", "collect template inputs", err)
	}

	plan, previewPlan, err := buildPlans(selectedTemplate, cfg, values)
	if err != nil {
		return nil, err
	}

	redactedInputs := redactTemplateValues(selectedTemplate, values)
	result := &TemplateRunResult{
		OK:          true,
		Connection:  cfg.Name,
		Command:     "template run",
		Template:    selectedTemplate.Name,
		Layer:       selectedTemplate.Layer,
		Source:      selectedTemplate.Source,
		Preview:     preview,
		DryRun:      dryRun,
		Transaction: selectedTemplate.Transaction,
		Inputs:      redactedInputs,
		Actions:     make([]ActionResult, 0, len(previewPlan.Actions)),
	}

	switch {
	case preview:
		for _, action := range previewPlan.Actions {
			result.Actions = append(result.Actions, ActionResult{
				Description: action.Description,
				SQL:         templateVerboseSQL(verbose, action.SQL),
				Status:      ActionStatusPreview,
			})
		}
		return result, nil
	case dryRun:
		planResult, runErr := a.runPlan(ctx, plan, noopTransactionStarter{}, true)
		if planResult != nil {
			result.Transaction = planResult.Transaction
			for index, action := range planResult.Actions {
				if verbose && index < len(previewPlan.Actions) {
					action.SQL = previewPlan.Actions[index].SQL
				} else {
					action.SQL = ""
				}
				result.Actions = append(result.Actions, action)
			}
		}
		return result, runErr
	default:
		db, err := a.openConnection(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		planResult, runErr := a.executePlan(ctx, plan, sqlRunner{db: db})
		if planResult != nil {
			result.OK = planResult.OK
			result.Transaction = planResult.Transaction
			result.Committed = planResult.Committed
			result.RolledBack = planResult.RolledBack
			for index, action := range planResult.Actions {
				if verbose && index < len(previewPlan.Actions) {
					action.SQL = previewPlan.Actions[index].SQL
				} else {
					action.SQL = ""
				}
				result.Actions = append(result.Actions, action)
			}
		}
		return result, runErr
	}
}

func (a *Application) collectTemplateRunInputs(ctx context.Context, template *tpl.Template, rawInputs map[string]string, requireAll bool, database string) (map[string]string, error) {
	values := make(map[string]string)
	if strings.TrimSpace(database) != "" {
		values["database"] = strings.TrimSpace(database)
	}
	for key, value := range rawInputs {
		if strings.HasSuffix(key, "-env") {
			name := strings.TrimSpace(strings.TrimSuffix(key, "-env"))
			if name == "" {
				return nil, util.WrapLayer("validation", "collect template inputs", fmt.Errorf("input key is required"))
			}
			resolved, err := resolveTemplateInputFromEnv(value)
			if err != nil {
				return nil, err
			}
			values[name] = resolved
			continue
		}
		values[key] = value
	}

	initialRequireAll := requireAll && ctx == nil
	merged, err := mergeTemplateInputs(template, values, initialRequireAll)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		if err := a.collectTemplateInputs(ctx, template, merged); err != nil {
			return nil, err
		}
	}
	return mergeTemplateInputs(template, merged, requireAll)
}

func (a *Application) printTemplatesCatalog(result *TemplatesCatalogResult) {
	if result == nil {
		return
	}
	a.prompt.Println("Templates:")
	lines := make([]string, 0, len(result.Templates))
	nameWidth := len("name")
	scopeWidth := len("scope")
	for _, candidate := range result.Templates {
		nameWidth = max(nameWidth, len(candidate.Name))
		scopeWidth = max(scopeWidth, len(candidate.Scope))
	}
	for _, candidate := range result.Templates {
		lines = append(lines, fmt.Sprintf("%-*s  %-*s  %s", nameWidth, candidate.Name, scopeWidth, candidate.Scope, candidate.Command))
	}
	slices.Sort(lines)
	for _, line := range lines {
		a.prompt.Println(line)
	}
}

func (a *Application) printTemplateDescription(result *TemplateDescriptionResult, verbose bool) {
	if result == nil {
		return
	}
	a.prompt.Printf("Template: %s\n", result.Name)
	a.prompt.Printf("Scope: %s\n", result.Scope)
	a.prompt.Printf("Command: %s\n", result.Command)
	a.prompt.Printf("Transaction: %t\n", result.Transaction)
	if strings.TrimSpace(result.Description) != "" {
		a.prompt.Printf("Description: %s\n", result.Description)
	}
	a.prompt.Println("")
	a.prompt.Println("Inputs:")
	if len(result.Inputs) == 0 {
		a.prompt.Println("  <none>")
	} else {
		for _, input := range result.Inputs {
			required := "optional"
			if input.Required {
				required = "required"
			}
			line := fmt.Sprintf("  %s  %s  %s", input.Name, input.Type, required)
			if strings.TrimSpace(input.Description) != "" {
				line += "  " + input.Description
			}
			a.prompt.Println(line)
		}
	}
	a.prompt.Println("")
	a.prompt.Println("Actions:")
	for index, action := range result.Actions {
		a.prompt.Printf("  %d. %s\n", index+1, action.Description)
		if verbose && strings.TrimSpace(action.SQL) != "" {
			a.prompt.Printf("     SQL: %s\n", action.SQL)
		}
	}
}

func (a *Application) printTemplateRunPreview(plan *tpl.ExecutionPlan, inputs map[string]string, verbose bool, preview bool, dryRun bool) {
	a.prompt.Printf("Template: %s\n", plan.TemplateName)
	a.prompt.Println("")
	a.prompt.Println("Inputs:")
	if len(inputs) == 0 {
		a.prompt.Println("  <none>")
	} else {
		keys := make([]string, 0, len(inputs))
		for key := range inputs {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			a.prompt.Printf("  %s: %s\n", key, displayTemplateInputValue(inputs[key]))
		}
	}
	a.prompt.Println("")
	a.prompt.Println("Execution Plan:")
	for index, action := range plan.Actions {
		a.prompt.Printf("  %d. %s\n", index+1, action.Description)
	}
	if verbose {
		a.prompt.Println("")
		a.prompt.Println("SQL Preview:")
		for index, action := range plan.Actions {
			a.prompt.Printf("  %d. %s\n", index+1, action.SQL)
		}
	}
	if preview {
		a.prompt.Println("")
		a.prompt.Println("Preview only. No actions executed.")
		return
	}
	if dryRun {
		a.prompt.Println("")
		a.prompt.Println("Dry-run mode is enabled. SQL will be rendered but not executed.")
	}
}

func (a *Application) printTemplateRunResult(result *TemplateRunResult) {
	if result == nil {
		return
	}
	a.prompt.Printf("Template: %s\n", result.Template)
	a.prompt.Println("")
	a.prompt.Println("Inputs:")
	if len(result.Inputs) == 0 {
		a.prompt.Println("  <none>")
	} else {
		keys := make([]string, 0, len(result.Inputs))
		for key := range result.Inputs {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			a.prompt.Printf("  %s: %s\n", key, displayTemplateInputValue(result.Inputs[key]))
		}
	}
	a.prompt.Println("")
	if result.Preview {
		a.prompt.Println("Execution Plan:")
		for index, action := range result.Actions {
			a.prompt.Printf("  %d. %s\n", index+1, action.Description)
			if strings.TrimSpace(action.SQL) != "" {
				a.prompt.Printf("     SQL: %s\n", action.SQL)
			}
		}
		a.prompt.Println("")
		a.prompt.Println("Preview only. No actions executed.")
		return
	}

	for _, action := range result.Actions {
		switch action.Status {
		case ActionStatusDryRun:
			a.prompt.Printf("[DRY-RUN] %s\n", action.Description)
		case ActionStatusFailed:
			a.prompt.Printf("[FAIL] %s%s\n", action.Description, formatActionDuration(action.DurationMS))
		default:
			a.prompt.Printf("[OK] %s%s\n", action.Description, formatActionDuration(action.DurationMS))
		}
		if strings.TrimSpace(action.SQL) != "" {
			a.prompt.Printf("  SQL: %s\n", action.SQL)
		}
	}
	if result.RolledBack {
		a.prompt.Println("Rolled back transaction.")
	}
	if result.Committed {
		a.prompt.Println("Committed transaction.")
	}
}

func (a *Application) templateScopeConfig(connectionName string) (*config.ConnectionConfig, error) {
	if strings.TrimSpace(connectionName) != "" {
		cfg, err := a.store.LoadConnection(strings.TrimSpace(connectionName))
		if err != nil {
			return nil, util.WrapLayer("config", "load connection "+strings.TrimSpace(connectionName), err)
		}
		return cfg, nil
	}
	if a.session != nil && a.session.Connection != nil {
		return cloneConnectionConfig(a.session.Connection), nil
	}
	sessionFile, err := a.store.LoadSession()
	if err != nil {
		return nil, util.WrapLayer("config", "load session", err)
	}
	if strings.TrimSpace(sessionFile.CurrentConnection) == "" {
		return &config.ConnectionConfig{Driver: "mysql"}, nil
	}
	cfg, err := a.store.LoadConnection(sessionFile.CurrentConnection)
	if err != nil {
		return nil, util.WrapLayer("config", "load current session connection "+sessionFile.CurrentConnection, err)
	}
	return cfg, nil
}

func (a *Application) requireTemplateRunConnection(ctx context.Context) (*config.ConnectionConfig, error) {
	cfg, db, err := a.requireConnection(ctx)
	if err != nil {
		return nil, util.WrapLayer("validation", "template run", fmt.Errorf("no active database connection; run connect first"))
	}
	if db == nil {
		return nil, util.WrapLayer("validation", "template run", fmt.Errorf("no active database connection; run connect first"))
	}
	return cfg, nil
}

func parseDescribeTemplateArgs(args []string) (string, bool, error) {
	if len(args) == 0 {
		return "", false, fmt.Errorf("usage: describe template <name> [--verbose]")
	}
	name := args[0]
	verbose := false
	for _, arg := range args[1:] {
		if arg == "--verbose" || arg == "verbose" {
			verbose = true
			continue
		}
		return "", false, fmt.Errorf("usage: describe template <name> [--verbose]")
	}
	return name, verbose, nil
}

func parseTemplateRunArgs(args []string) (string, bool, bool, bool, error) {
	if len(args) == 0 {
		return "", false, false, false, fmt.Errorf("usage: template run <name> [--preview] [--dry-run] [--verbose]")
	}
	name := args[0]
	preview := false
	verbose := false
	dryRunFlag := false
	for _, arg := range args[1:] {
		switch arg {
		case "--preview", "preview":
			preview = true
		case "--verbose", "verbose":
			verbose = true
		case "--dry-run", "dry-run":
			dryRunFlag = true
		default:
			return "", false, false, false, fmt.Errorf("usage: template run <name> [--preview] [--dry-run] [--verbose]")
		}
	}
	return name, preview, verbose, dryRunFlag, nil
}

func resolveTemplateInputFromEnv(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", util.WrapLayer("validation", "collect template inputs", fmt.Errorf("environment variable name is required"))
	}
	value, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(value) == "" {
		return "", util.WrapLayer("validation", "collect template inputs", fmt.Errorf("environment variable %s is not set", name))
	}
	return value, nil
}

func redactTemplateActionSQL(template *tpl.Template, sqlText string) string {
	redacted := sqlText
	for _, input := range template.Inputs {
		if input.EffectiveType() != "secret" {
			continue
		}
		pattern := regexp.MustCompile(`{{\s*` + regexp.QuoteMeta(input.Name) + `\s*}}`)
		redacted = pattern.ReplaceAllString(redacted, "***")
	}
	return redacted
}

func displayTemplateInputValue(value string) string {
	if strings.TrimSpace(value) == "***" {
		return "[REDACTED]"
	}
	return value
}

func templateVerboseSQL(verbose bool, sqlText string) string {
	if !verbose {
		return ""
	}
	return sqlText
}
