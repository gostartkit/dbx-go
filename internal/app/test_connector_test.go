package app

import (
	"context"
	"database/sql"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
)

type spyConnector struct {
	openErr   error
	openCalls int
	lastName  string
}

func (s *spyConnector) Open(_ context.Context, cfg *config.ConnectionConfig) (*sql.DB, error) {
	s.openCalls++
	if cfg != nil {
		s.lastName = cfg.Name
	}
	if s.openErr != nil {
		return nil, s.openErr
	}
	return nil, nil
}

func (s *spyConnector) Ping(context.Context, *config.ConnectionConfig, *sql.DB) error {
	return nil
}

func (s *spyConnector) ListDatabases(context.Context, *config.ConnectionConfig, *sql.DB) ([]string, error) {
	return nil, nil
}

func (s *spyConnector) ListTables(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return nil, nil
}

func (s *spyConnector) ShowColumns(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.SchemaColumn, error) {
	return nil, nil
}

func (s *spyConnector) PeekRows(context.Context, *config.ConnectionConfig, *sql.DB, string, string, int) (*driver.RowSet, error) {
	return nil, nil
}

func (s *spyConnector) SampleRows(context.Context, *config.ConnectionConfig, *sql.DB, string, string, int) (*driver.RowSet, error) {
	return nil, nil
}

func (s *spyConnector) ShowCreateTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) (string, error) {
	return "", nil
}

func (s *spyConnector) ShowTableStatus(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableStatus, error) {
	return nil, nil
}

func (s *spyConnector) QueryStrings(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return nil, nil
}
