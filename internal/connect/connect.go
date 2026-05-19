package connect

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
)

type Connector struct{}

func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) Open(ctx context.Context, cfg *config.ConnectionConfig) (*sql.DB, error) {
	switch cfg.Driver {
	case "mysql":
		return driver.OpenMySQL(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) Ping(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) error {
	switch cfg.Driver {
	case "mysql":
		pingCtx, cancel := context.WithTimeout(ctx, minDuration(cfg.QueryTimeout(), 5*time.Second))
		defer cancel()
		return driver.Ping(pingCtx, db)
	default:
		return fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ListDatabases(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) ([]string, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ListDatabases(queryCtx, db)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) QueryStrings(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, query string) ([]string, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.QueryStrings(queryCtx, db, query)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ExecStatement(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, statement string) error {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ExecStatement(queryCtx, db, statement)
	default:
		return fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func minDuration(a time.Duration, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
