package app

import (
	"context"
	"database/sql"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/connect"
)

type connectorClient interface {
	Open(ctx context.Context, cfg *config.ConnectionConfig) (*sql.DB, error)
	Ping(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) error
	ListDatabases(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) ([]string, error)
	QueryStrings(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, query string) ([]string, error)
}

func defaultConnector() connectorClient {
	return connect.NewConnector()
}
