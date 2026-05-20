package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
)

type failingConnector struct {
	openErr error
}

func (f failingConnector) Open(context.Context, *config.ConnectionConfig) (*sql.DB, error) {
	return nil, f.openErr
}

func (f failingConnector) Diagnose(context.Context, *config.ConnectionConfig) (*driver.DiagnosticTrace, error) {
	return nil, f.openErr
}

func (f failingConnector) Ping(context.Context, *config.ConnectionConfig, *sql.DB) error {
	return nil
}

func (f failingConnector) ListDatabases(context.Context, *config.ConnectionConfig, *sql.DB) ([]string, error) {
	return nil, nil
}

func (f failingConnector) ListTables(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return nil, nil
}

func (f failingConnector) DescribeTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableColumn, error) {
	return nil, nil
}

func (f failingConnector) ShowIndexes(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableIndex, error) {
	return nil, nil
}

func (f failingConnector) ShowCreateTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) (string, error) {
	return "", nil
}

func (f failingConnector) ShowTableStatus(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]driver.TableStatus, error) {
	return nil, nil
}

func (f failingConnector) ShowGrants(context.Context, *config.ConnectionConfig, *sql.DB, string, string) ([]string, error) {
	return nil, nil
}

func (f failingConnector) ShowProcesslist(context.Context, *config.ConnectionConfig, *sql.DB) ([]driver.Process, error) {
	return nil, nil
}

func (f failingConnector) ShowVariables(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]driver.SystemVariable, error) {
	return nil, nil
}

func (f failingConnector) TruncateTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string) error {
	return nil
}

func (f failingConnector) RenameTable(context.Context, *config.ConnectionConfig, *sql.DB, string, string, string) error {
	return nil
}

func (f failingConnector) QueryStrings(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return nil, nil
}

func TestInteractiveConnectionCreateSavesWhenTestFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := strings.Join([]string{
		"prod",
		"direct",
		"127.0.0.1",
		"3306",
		"root",
		"prompt every time",
		"10",
		"30",
		"y",
		"bad-password",
		"y",
	}, "\n") + "\n"

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader(input), &out, &out, Options{
		ConfigDir: root,
		Connector: failingConnector{openErr: errors.New("mysql error: ping database: ssh error: complete SSH handshake with 39.108.126.24:22: ssh: handshake failed")},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	if err := app.handleConnectionCreate(context.Background()); err != nil {
		t.Fatalf("handleConnectionCreate returned error: %v", err)
	}

	if !app.store.ConnectionExists("prod") {
		t.Fatalf("expected saved connection after failed test")
	}

	output := out.String()
	if !strings.Contains(output, "Connection test failed:") {
		t.Fatalf("output missing failure warning: %q", output)
	}
	if !strings.Contains(output, "Saved connection:") {
		t.Fatalf("output missing saved message: %q", output)
	}
	if !strings.Contains(output, "connection edit prod") {
		t.Fatalf("output missing edit hint: %q", output)
	}
}
