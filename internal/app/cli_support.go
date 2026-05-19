package app

import (
	"context"
	"fmt"
	"sort"
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

func (a *Application) statusForCLI(connectionName string) (*StatusResult, error) {
	result := &StatusResult{
		OK:     true,
		DryRun: a.dryRun,
	}

	if strings.TrimSpace(connectionName) != "" {
		cfg, err := a.store.LoadConnection(connectionName)
		if err != nil {
			return nil, util.WrapLayer("config", "load connection "+connectionName, err)
		}
		result.Connection = redactConnection(cfg)
		result.ConnectionName = cfg.Name
		result.ConnectionExists = true
		result.SelectedByFlag = true
		result.Message = "loaded connection config"
		return result, nil
	}

	sessionFile, err := a.store.LoadSession()
	if err != nil {
		return nil, util.WrapLayer("config", "load session", err)
	}
	if strings.TrimSpace(sessionFile.CurrentConnection) == "" {
		result.Message = "no saved session"
		return result, nil
	}

	result.HasStoredSession = true
	result.CurrentSession = sessionFile.CurrentConnection
	result.ConnectionName = sessionFile.CurrentConnection
	result.ConnectionExists = a.store.ConnectionExists(sessionFile.CurrentConnection)
	if !result.ConnectionExists {
		result.Message = "saved session points to a missing connection"
		return result, nil
	}

	cfg, err := a.store.LoadConnection(sessionFile.CurrentConnection)
	if err != nil {
		return nil, util.WrapLayer("config", "load current session connection "+sessionFile.CurrentConnection, err)
	}
	result.Connection = redactConnection(cfg)
	result.Message = "loaded saved session config"
	return result, nil
}

func (a *Application) selectTemplateForCLI(command string, cfg *config.ConnectionConfig, templateName string) (*tpl.Template, error) {
	if strings.TrimSpace(templateName) != "" {
		selected, err := a.templates.ResolveNamed(command, cfg, templateName)
		if err != nil {
			return nil, util.WrapLayer("template", "resolve template "+templateName, err)
		}
		return selected, nil
	}

	templates, err := a.templates.List(command, cfg)
	if err != nil {
		return nil, util.WrapLayer("template", "list templates for "+command, err)
	}
	if len(templates) == 0 {
		return nil, util.WrapLayer("template", "resolve template for "+command, fmt.Errorf("no template found"))
	}
	if len(templates) > 1 {
		names := make([]string, 0, len(templates))
		for _, candidate := range templates {
			names = append(names, candidate.Name)
		}
		sort.Strings(names)
		return nil, util.WrapLayer("template", "resolve template for "+command, fmt.Errorf("multiple templates match; specify --template (%s)", strings.Join(names, ", ")))
	}

	selected := templates[0]
	return &selected, nil
}
