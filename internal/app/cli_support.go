package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) listConnectionSummaries() (*ConnectionsResult, error) {
	connections, err := a.store.ListConnections()
	if err != nil {
		return nil, util.WrapLayer("config", "list configured connections", err)
	}

	result := &ConnectionsResult{
		OK:          true,
		Connections: make([]ConnectionSummary, 0, len(connections)),
	}
	for _, connection := range connections {
		result.Connections = append(result.Connections, summarizeConnection(connection))
	}
	return result, nil
}

func (a *Application) showConnection(name string) (*RedactedConnection, error) {
	cfg, err := a.store.LoadConnection(name)
	if err != nil {
		return nil, util.WrapLayer("config", "load connection "+name, err)
	}
	return redactConnection(cfg), nil
}

func (a *Application) resolveConnectionConfig(name string) (*config.ConnectionConfig, error) {
	if strings.TrimSpace(name) != "" {
		cfg, err := a.store.LoadConnection(name)
		if err != nil {
			return nil, util.WrapLayer("config", "load connection "+name, err)
		}
		return cfg, nil
	}

	sessionFile, err := a.store.LoadSession()
	if err != nil {
		return nil, util.WrapLayer("config", "load session", err)
	}
	if strings.TrimSpace(sessionFile.CurrentConnection) == "" {
		return nil, util.WrapLayer("config", "resolve connection", fmt.Errorf("no connection selected; use --connection or run connect"))
	}

	cfg, err := a.store.LoadConnection(sessionFile.CurrentConnection)
	if err != nil {
		return nil, util.WrapLayer("config", "load current session connection "+sessionFile.CurrentConnection, err)
	}
	return cfg, nil
}

func (a *Application) connectNonInteractive(ctx context.Context, name string) (*ConnectResult, error) {
	cfg, err := a.store.LoadConnection(name)
	if err != nil {
		return nil, util.WrapLayer("config", "load connection "+name, err)
	}

	if err := a.activateConnection(ctx, cfg, true); err != nil {
		return nil, err
	}
	defer a.session.Close()

	return &ConnectResult{
		OK:         true,
		Connection: cfg.Name,
		Message:    fmt.Sprintf("Connected to %s.", cfg.Name),
	}, nil
}

func (a *Application) applyCLIDatabaseSelection(ctx context.Context, cfg *config.ConnectionConfig, database string) error {
	if cfg == nil {
		return nil
	}
	a.session.Connection = cloneConnectionConfig(cfg)
	return a.setRuntimeDatabaseSelection(ctx, cfg, nil, database, false)
}

func (a *Application) useDatabaseForCLI(ctx context.Context, connectionName string, database string) (*UseDatabaseResult, error) {
	cfg, err := a.resolveConnectionConfig(connectionName)
	if err != nil {
		return nil, err
	}
	if err := a.setRuntimeDatabaseSelection(ctx, cfg, nil, database, true); err != nil {
		return nil, err
	}
	return &UseDatabaseResult{
		OK:         true,
		Connection: cfg.Name,
		Database:   a.session.Database,
	}, nil
}

func (a *Application) selectTemplateForCLI(command string, cfg *config.ConnectionConfig, templateName string) (*tpl.Template, error) {
	if strings.TrimSpace(templateName) != "" {
		selected, err := a.templates.ResolveNamed(command, cfg, templateName)
		if err != nil {
			return nil, util.WrapLayer("template", "resolve template "+templateName, err)
		}
		return selected, nil
	}

	match, err := a.templates.ResolveByLayer(command, cfg)
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
