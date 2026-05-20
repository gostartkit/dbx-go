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
	ShowColumns(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.SchemaColumn, error)
	CountRows(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) (int64, error)
	PeekRows(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string, limit int) (*driver.RowSet, error)
	SampleRows(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string, limit int) (*driver.RowSet, error)
	ShowIndexes(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.TableIndex, error)
	ShowForeignKeys(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.ForeignKey, error)
	ShowCreateTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) (string, error)
	ShowTableStatus(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.TableStatus, error)
	ShowTriggers(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string) ([]driver.Trigger, error)
	ShowViews(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string) ([]string, error)
	ShowGrants(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, user string, host string) ([]string, error)
	ShowProcesslist(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) ([]driver.Process, error)
	ShowVariables(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, pattern string) ([]driver.SystemVariable, error)
	TruncateTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) error
	RenameTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, from string, to string) error
	QueryStrings(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, query string) ([]string, error)
}

func defaultConnector() connectorClient {
	return connect.NewConnector()
}
