package connect

import (
	"context"
	"database/sql"
	"fmt"

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

func (c *Connector) ListDatabases(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) ([]string, error) {
	switch cfg.Driver {
	case "mysql":
		return driver.ListDatabases(ctx, db)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) QueryStrings(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, query string) ([]string, error) {
	switch cfg.Driver {
	case "mysql":
		return driver.QueryStrings(ctx, db, query)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ExecStatements(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, statements []string) error {
	switch cfg.Driver {
	case "mysql":
		return driver.ExecStatements(ctx, db, statements)
	default:
		return fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}
