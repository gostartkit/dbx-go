package app

import (
	"context"
	"database/sql"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/connect"
	"pkg.gostartkit.com/dbx/internal/driver"
)

type connectorClient interface {
	Open(ctx context.Context, cfg *config.ConnectionConfig) (*sql.DB, error)
	Diagnose(ctx context.Context, cfg *config.ConnectionConfig) (*driver.DiagnosticTrace, error)
	Ping(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) error
	ListDatabases(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) ([]string, error)
	ListTables(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string) ([]string, error)
	DescribeTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.TableColumn, error)
	ShowGrants(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, user string, host string) ([]string, error)
	QueryStrings(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, query string) ([]string, error)
}

func defaultConnector() connectorClient {
	return connect.NewConnector()
}
