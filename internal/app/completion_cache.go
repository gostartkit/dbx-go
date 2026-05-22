package app

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"pkg.gostartkit.com/dbx/internal/config"
)

const completionRefreshTimeout = 2 * time.Second

func (a *Application) completionSnapshot() (*config.ConnectionConfig, *sql.DB, string) {
	if a == nil || a.session == nil || a.session.Connection == nil || a.session.DB == nil {
		return nil, nil, ""
	}
	return cloneConnectionConfig(a.session.Connection), a.session.DB, a.session.Database
}

func (a *Application) currentDatabaseCompletionValues(cfg *config.ConnectionConfig, db *sql.DB) []string {
	connName := strings.TrimSpace(cfg.Name)
	if connName == "" {
		return nil
	}

	a.completionMu.Lock()
	if a.completionDBsReady && a.completionDBsConn == connName {
		values := append([]string(nil), a.completionDBs...)
		a.completionMu.Unlock()
		return values
	}
	if a.completionDBsLoadingConn == connName {
		a.completionMu.Unlock()
		return nil
	}
	a.completionDBsLoadingConn = connName
	a.completionMu.Unlock()

	go a.refreshDatabaseCompletions(connName, cfg, db)
	return nil
}

func (a *Application) currentUserCompletionValues(cfg *config.ConnectionConfig, db *sql.DB) []string {
	connName := strings.TrimSpace(cfg.Name)
	if connName == "" {
		return nil
	}

	a.completionMu.Lock()
	if a.completionUsersReady && a.completionUsersConn == connName {
		values := append([]string(nil), a.completionUsers...)
		a.completionMu.Unlock()
		return values
	}
	if a.completionUsersLoadingConn == connName {
		a.completionMu.Unlock()
		return nil
	}
	a.completionUsersLoadingConn = connName
	a.completionMu.Unlock()

	go a.refreshUserCompletions(connName, cfg, db)
	return nil
}

func (a *Application) currentTableCompletionValues(cfg *config.ConnectionConfig, db *sql.DB, database string) []string {
	connName := strings.TrimSpace(cfg.Name)
	database = strings.TrimSpace(database)
	if connName == "" || database == "" {
		return nil
	}

	a.completionMu.Lock()
	if a.completionTablesReady && a.completionTablesConn == connName && a.completionTablesDB == database {
		values := append([]string(nil), a.completionTables...)
		a.completionMu.Unlock()
		return values
	}
	if a.completionTablesLoadingConn == connName && a.completionTablesLoadingDB == database {
		a.completionMu.Unlock()
		return nil
	}
	a.completionTablesLoadingConn = connName
	a.completionTablesLoadingDB = database
	a.completionMu.Unlock()

	go a.refreshTableCompletions(connName, database, cfg, db)
	return nil
}

func (a *Application) refreshDatabaseCompletions(connName string, cfg *config.ConnectionConfig, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), completionRefreshTimeout)
	defer cancel()

	values, err := a.connector.ListDatabases(ctx, cfg, db)

	a.completionMu.Lock()
	defer a.completionMu.Unlock()
	if a.completionDBsLoadingConn == connName {
		a.completionDBsLoadingConn = ""
	}
	if err != nil {
		return
	}
	a.completionDBsConn = connName
	a.completionDBs = append([]string(nil), values...)
	a.completionDBsReady = true
}

func (a *Application) refreshUserCompletions(connName string, cfg *config.ConnectionConfig, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), completionRefreshTimeout)
	defer cancel()

	values, err := a.connector.QueryStrings(ctx, cfg, db, "SELECT DISTINCT User FROM mysql.user ORDER BY User")

	a.completionMu.Lock()
	defer a.completionMu.Unlock()
	if a.completionUsersLoadingConn == connName {
		a.completionUsersLoadingConn = ""
	}
	if err != nil {
		return
	}
	a.completionUsersConn = connName
	a.completionUsers = append([]string(nil), values...)
	a.completionUsersReady = true
}

func (a *Application) refreshTableCompletions(connName string, database string, cfg *config.ConnectionConfig, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), completionRefreshTimeout)
	defer cancel()

	values, err := a.connector.ListTables(ctx, cfg, db, database)

	a.completionMu.Lock()
	defer a.completionMu.Unlock()
	if a.completionTablesLoadingConn == connName && a.completionTablesLoadingDB == database {
		a.completionTablesLoadingConn = ""
		a.completionTablesLoadingDB = ""
	}
	if err != nil {
		return
	}
	a.completionTablesConn = connName
	a.completionTablesDB = database
	a.completionTables = append([]string(nil), values...)
	a.completionTablesReady = true
}
