package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

type userPasswordMode string

const (
	userPasswordPrompt    userPasswordMode = "prompt"
	userPasswordEnv       userPasswordMode = "env"
	userPasswordGenerated userPasswordMode = "generated"
)

type userCreateOptions struct {
	Username          string
	Host              string
	Password          string
	PasswordGenerated bool
	Grant             string
	Database          string
}

type userDropOptions struct {
	Username string
	Host     string
}

func defaultUserHost(host string) string {
	if strings.TrimSpace(host) == "" {
		return "%"
	}
	return strings.TrimSpace(host)
}

func validateUserHost(host string) error {
	return util.ValidateMySQLUserHost(host)
}

func buildCreateUserValues(options userCreateOptions) map[string]string {
	values := map[string]string{
		"username":          options.Username,
		"user_host":         defaultUserHost(options.Host),
		"password":          options.Password,
		"grant":             "",
		"grant_database":    "",
		"grant_description": "",
		"grant_sql":         "",
	}

	switch options.Grant {
	case "all":
		values["grant"] = "all"
		values["grant_database"] = options.Database
		values["grant_description"] = fmt.Sprintf("Grant ALL PRIVILEGES on `%s`.*", options.Database)
		values["grant_sql"] = fmt.Sprintf(
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%s'",
			options.Database,
			util.EscapeMySQLString(options.Username),
			util.EscapeMySQLString(defaultUserHost(options.Host)),
		)
	case "readonly":
		values["grant"] = "readonly"
		values["grant_database"] = options.Database
		values["grant_description"] = fmt.Sprintf("Grant SELECT on `%s`.*", options.Database)
		values["grant_sql"] = fmt.Sprintf(
			"GRANT SELECT ON `%s`.* TO '%s'@'%s'",
			options.Database,
			util.EscapeMySQLString(options.Username),
			util.EscapeMySQLString(defaultUserHost(options.Host)),
		)
	}

	return values
}

func buildDropUserValues(options userDropOptions) map[string]string {
	return map[string]string{
		"username":  options.Username,
		"user_host": defaultUserHost(options.Host),
	}
}

func resolvePasswordFromEnv(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", util.WrapLayer("validation", "resolve password env", fmt.Errorf("environment variable name is required"))
	}
	value, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(value) == "" {
		return "", util.WrapLayer("validation", "resolve password env", fmt.Errorf("environment variable %s is not set", name))
	}
	return value, nil
}

func (a *Application) promptUserGrant(ctx context.Context, currentDatabase string) (string, string, error) {
	if strings.TrimSpace(currentDatabase) == "" {
		return "", "", nil
	}

	grantCurrent, err := a.confirm(ctx, fmt.Sprintf("Grant access to current database %s?", currentDatabase), true)
	if err != nil {
		return "", "", err
	}
	if !grantCurrent {
		return "", "", nil
	}

	privilege, err := a.choose(ctx, "Privileges", []string{"all", "readonly"}, "all")
	if err != nil {
		return "", "", err
	}
	return privilege, currentDatabase, nil
}

func (a *Application) resolveCreateUserPlan(ctx context.Context, cfg *config.ConnectionConfig, templateName string, options userCreateOptions, extraInputs map[string]string, cli bool) (*tpl.ExecutionPlan, *tpl.ExecutionPlan, *tpl.Template, error) {
	var (
		selectedTemplate *tpl.Template
		err              error
	)
	if cli {
		selectedTemplate, err = a.selectTemplateForCLI("create user", cfg, templateName)
	} else {
		selectedTemplate, err = a.templates.Resolve("create user", cfg)
	}
	if err != nil {
		return nil, nil, nil, util.WrapLayer("template", "resolve create user template", err)
	}

	values := buildCreateUserValues(options)
	for key, value := range extraInputs {
		values[key] = value
	}

	merged, err := mergeTemplateInputs(selectedTemplate, values, cli)
	if err != nil {
		return nil, nil, nil, util.WrapLayer("template", "collect template inputs", err)
	}

	plan, previewPlan, err := buildPlans(selectedTemplate, cfg, merged)
	if err != nil {
		return nil, nil, nil, err
	}
	return plan, previewPlan, selectedTemplate, nil
}

func printGeneratedPassword(prompt printer, password string) {
	if strings.TrimSpace(password) == "" {
		return
	}
	prompt.Println("Generated password:")
	prompt.Println("  " + password)
}
