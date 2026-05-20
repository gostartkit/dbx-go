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

func (c *Connector) ListTables(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string) ([]string, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ListTables(queryCtx, db, database)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) DescribeTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.TableColumn, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.DescribeTable(queryCtx, db, database, table)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowColumns(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.SchemaColumn, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowColumns(queryCtx, db, database, table)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowIndexes(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.TableIndex, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowIndexes(queryCtx, db, database, table)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowForeignKeys(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.ForeignKey, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowForeignKeys(queryCtx, db, database, table)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowCreateTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) (string, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowCreateTable(queryCtx, db, database, table)
	default:
		return "", fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowTableStatus(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) ([]driver.TableStatus, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowTableStatus(queryCtx, db, database, table)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowTriggers(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string) ([]driver.Trigger, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowTriggers(queryCtx, db, database)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowViews(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string) ([]string, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowViews(queryCtx, db, database)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowGrants(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, user string, host string) ([]string, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowGrants(queryCtx, db, user, host)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowProcesslist(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB) ([]driver.Process, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowProcesslist(queryCtx, db)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) ShowVariables(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, pattern string) ([]driver.SystemVariable, error) {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.ShowVariables(queryCtx, db, pattern)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) TruncateTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, table string) error {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.TruncateTable(queryCtx, db, database, table)
	default:
		return fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func (c *Connector) RenameTable(ctx context.Context, cfg *config.ConnectionConfig, db *sql.DB, database string, from string, to string) error {
	switch cfg.Driver {
	case "mysql":
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout())
		defer cancel()
		return driver.RenameTable(queryCtx, db, database, from, to)
	default:
		return fmt.Errorf("unsupported driver %q", cfg.Driver)
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
