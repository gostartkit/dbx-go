package app

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) clearDatabaseSelection() {
	a.session.Database = ""
	a.completionDBs = nil
	a.completionDBsConn = ""
	a.clearTableCompletion()
}

func (a *Application) clearUserCompletion() {
	a.completionUsers = nil
	a.completionUsersConn = ""
}

func (a *Application) clearTableCompletion() {
	a.completionTables = nil
	a.completionTablesConn = ""
	a.completionTablesDB = ""
}

func (a *Application) saveCurrentSession() error {
	currentConnection := ""
	if a.session.Connection != nil {
		currentConnection = a.session.Connection.Name
	}
	return a.store.SaveSession(&config.SessionFile{
		CurrentConnection: currentConnection,
		CurrentDatabase:   a.session.Database,
	})
}

func (a *Application) listDatabasesForSelection(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) ([]string, error) {
	if db != nil {
		return a.connector.ListDatabases(ctx, cfg, db)
	}

	opened, err := a.openConnection(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if opened != nil {
		defer opened.Close()
	}

	return a.connector.ListDatabases(ctx, cfg, opened)
}

func (a *Application) setRuntimeDatabaseSelection(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, persist bool) error {
	database = strings.TrimSpace(database)
	if database == "" {
		if a.session.Connection == nil && cfg != nil {
			a.session.Connection = cloneConnectionConfig(cfg)
		}
		a.clearDatabaseSelection()
		if persist {
			if err := a.saveCurrentSession(); err != nil {
				return util.WrapLayer("config", "save session", err)
			}
		}
		return nil
	}

	if err := util.ValidateDatabaseName(database); err != nil {
		return util.WrapLayer("validation", "validate database name", err)
	}

	databases, err := a.listDatabasesForSelection(ctx, cfg, db)
	if err != nil {
		return err
	}
	if !slices.Contains(databases, database) {
		return util.WrapLayer("validation", "select database", fmt.Errorf("database not found: %s", database))
	}

	if a.session.Connection == nil && cfg != nil {
		a.session.Connection = cloneConnectionConfig(cfg)
	}
	a.session.Database = database
	a.completionDBsConn = cfg.Name
	a.completionDBs = append([]string(nil), databases...)
	a.clearTableCompletion()

	if persist {
		if err := a.saveCurrentSession(); err != nil {
			return util.WrapLayer("config", "save session", err)
		}
	}
	return nil
}
