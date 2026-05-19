package connect

import (
	"context"
	"fmt"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
)

func (c *Connector) Diagnose(ctx context.Context, cfg *config.ConnectionConfig) (*driver.DiagnosticTrace, error) {
	switch cfg.Driver {
	case "mysql":
		return driver.DiagnoseMySQL(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}
