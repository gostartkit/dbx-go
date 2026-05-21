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
	Ping(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) error
	ListDatabases(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) ([]string, error)
	ListTables(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string) ([]string, error)
	ShowColumns(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.SchemaColumn, error)
	PeekRows(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string, limit int) (*driver.RowSet, error)
	SampleRows(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string, limit int) (*driver.RowSet, error)
	ShowCreateTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) (string, error)
	ShowTableStatus(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.TableStatus, error)
	QueryStrings(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, query string) ([]string, error)
}

func defaultConnector() connectorClient {
	return connect.NewConnector()
}
