package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

type commandContextResolver struct {
	app *Application
}

func (a *Application) commandContext() *commandContextResolver {
	return &commandContextResolver{app: a}
}

func (r *commandContextResolver) resolveCLIConnection(name string) (*config.ConnectionConfig, error) {
	if strings.TrimSpace(name) != "" {
		cfg, err := r.app.store.LoadConnection(name)
		if err != nil {
			return nil, util.WrapLayer("config", "load connection "+name, err)
		}
		return cfg, nil
	}

	sessionFile, err := r.app.store.LoadSession()
	if err != nil {
		return nil, util.WrapLayer("config", "load session", err)
	}
	if strings.TrimSpace(sessionFile.CurrentConnection) == "" {
		return nil, util.WrapLayer("config", "resolve connection", fmt.Errorf("no connection selected; use --connection or run connect"))
	}

	cfg, err := r.app.store.LoadConnection(sessionFile.CurrentConnection)
	if err != nil {
		return nil, util.WrapLayer("config", "load current session connection "+sessionFile.CurrentConnection, err)
	}
	return cfg, nil
}

func (r *commandContextResolver) resolveTemplateScope(connectionName string) (*config.ConnectionConfig, error) {
	if strings.TrimSpace(connectionName) != "" {
		cfg, err := r.app.store.LoadConnection(strings.TrimSpace(connectionName))
		if err != nil {
			return nil, util.WrapLayer("config", "load connection "+strings.TrimSpace(connectionName), err)
		}
		return cfg, nil
	}
	if r.app.session != nil && r.app.session.Connection != nil {
		return cloneConnectionConfig(r.app.session.Connection), nil
	}

	sessionFile, err := r.app.store.LoadSession()
	if err != nil {
		return nil, util.WrapLayer("config", "load session", err)
	}
	if strings.TrimSpace(sessionFile.CurrentConnection) == "" {
		return &config.ConnectionConfig{Driver: "mysql"}, nil
	}

	cfg, err := r.app.store.LoadConnection(sessionFile.CurrentConnection)
	if err != nil {
		return nil, util.WrapLayer("config", "load current session connection "+sessionFile.CurrentConnection, err)
	}
	return cfg, nil
}

func (r *commandContextResolver) applyCLIDatabaseSelection(ctx context.Context, cfg *config.ConnectionConfig, database string) error {
	if cfg == nil {
		return nil
	}
	r.app.session.Connection = cloneConnectionConfig(cfg)
	return r.app.setRuntimeDatabaseSelection(ctx, cfg, nil, database, false)
}

func (r *commandContextResolver) resolveCLIConnectionAndDatabase(ctx context.Context, connectionName string, databaseName string, command string) (*config.ConnectionConfig, string, error) {
	cfg, err := r.resolveCLIConnection(connectionName)
	if err != nil {
		return nil, "", err
	}
	if err := r.applyCLIDatabaseSelection(ctx, cfg, databaseName); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(r.app.session.Database) == "" {
		return nil, "", util.WrapLayer("validation", command, fmt.Errorf("no database selected; use --database <name>"))
	}
	return cfg, r.app.session.Database, nil
}

func (r *commandContextResolver) selectCLITemplate(command string, cfg *config.ConnectionConfig, templateName string) (*tpl.Template, error) {
	if strings.TrimSpace(templateName) != "" {
		selected, err := r.app.templates.ResolveNamed(command, cfg, templateName)
		if err != nil {
			return nil, util.WrapLayer("template", "resolve template "+templateName, err)
		}
		return selected, nil
	}

	match, err := r.app.templates.ResolveByLayer(command, cfg)
	if err != nil {
		return nil, util.WrapLayer("template", "resolve template for "+command, err)
	}
	if len(match.Templates) == 0 {
		return nil, util.WrapLayer("template", "resolve template for "+command, fmt.Errorf("no template found for command %q and driver %q", command, templateDriver(cfg)))
	}
	if len(match.Templates) > 1 {
		return nil, util.WrapLayer("template", "resolve template for "+command, buildCLITemplateAmbiguityError(command, match))
	}

	selected := match.Templates[0]
	return &selected, nil
}

func (r *commandContextResolver) requireDatabaseContext(ctx context.Context) (*config.ConnectionConfig, *sql.DB, string, error) {
	cfg, db, err := r.app.requireConnection(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	if strings.TrimSpace(r.app.session.Database) == "" {
		return nil, nil, "", util.WrapLayer("validation", "database context", fmt.Errorf("no database selected; use: use <database>"))
	}
	return cfg, db, r.app.session.Database, nil
}
