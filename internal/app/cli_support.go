package app

import (
	"context"
	"fmt"

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

func (a *Application) useDatabaseForCLI(ctx context.Context, connectionName string, database string) (*UseDatabaseResult, error) {
	cfg, err := a.commandContext().resolveCLIConnection(connectionName)
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
