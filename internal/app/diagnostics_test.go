package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
	"pkg.gostartkit.com/dbx/internal/util"
)

type diagnosticConnector struct {
	openErr   error
	openCalls int
	lastName  string
	trace     *driver.DiagnosticTrace
}

func (d *diagnosticConnector) Open(_ context.Context, cfg *config.ConnectionConfig) (*sql.DB, error) {
	d.openCalls++
	if cfg != nil {
		d.lastName = cfg.Name
	}
	if d.openErr != nil {
		return nil, d.openErr
	}
	return nil, nil
}

func (d *diagnosticConnector) Diagnose(_ context.Context, cfg *config.ConnectionConfig) (*driver.DiagnosticTrace, error) {
	d.openCalls++
	if cfg != nil {
		d.lastName = cfg.Name
	}
	if d.trace != nil {
		return d.trace, d.openErr
	}

	trace := &driver.DiagnosticTrace{
		Steps: make([]driver.DiagnosticStep, 0, len(expectedDiagnosticLayers(cfg))),
	}
	failedLayer := ""
	if d.openErr != nil {
		failedLayer = diagnosticFailureLayerForTest(cfg, d.openErr)
	}
	for _, layer := range expectedDiagnosticLayers(cfg) {
		if layer == failedLayer {
			trace.Steps = append(trace.Steps, driver.DiagnosticStep{
				Name:   layer,
				Status: "fail",
				Error:  diagnosticRootError(d.openErr),
			})
			return trace, d.openErr
		}
		trace.Steps = append(trace.Steps, driver.DiagnosticStep{Name: layer, Status: "ok"})
	}
	return trace, nil
}

func (d *diagnosticConnector) Ping(context.Context, *config.ConnectionConfig, *sql.DB) error {
	return nil
}

func (d *diagnosticConnector) ListDatabases(context.Context, *config.ConnectionConfig, *sql.DB) ([]string, error) {
	return nil, nil
}

func (d *diagnosticConnector) QueryStrings(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return nil, nil
}

func diagnosticFailureLayerForTest(cfg *config.ConnectionConfig, err error) string {
	found := ""
	for current := err; current != nil; current = errors.Unwrap(current) {
		layerErr, ok := current.(*util.LayerError)
		if !ok {
			continue
		}
		switch layerErr.Layer {
		case "validation", "config":
			found = "config"
		case "proxy", "ssh", "mysql":
			found = layerErr.Layer
		}
	}
	if found != "" {
		return found
	}
	expected := expectedDiagnosticLayers(cfg)
	return expected[len(expected)-1]
}

func TestHandleLineConnectionTestParsesName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	connector := &diagnosticConnector{}
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{
		ConfigDir: root,
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	exit, err := app.handleLine(context.Background(), "connection test prod")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if connector.openCalls != 1 || connector.lastName != "prod" {
		t.Fatalf("unexpected connector calls: calls=%d name=%q", connector.openCalls, connector.lastName)
	}
}

func TestHandleLineConnectionTestParsesVerboseName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	connector := &diagnosticConnector{}
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{
		ConfigDir: root,
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	exit, err := app.handleLine(context.Background(), "connection test prod verbose")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if connector.openCalls != 1 || connector.lastName != "prod" {
		t.Fatalf("unexpected connector calls: calls=%d name=%q", connector.openCalls, connector.lastName)
	}
}

func TestHandleLineConnectionTestParsesVerboseWithoutName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	connector := &diagnosticConnector{}
	app, err := NewWithOptions(strings.NewReader("prod\n"), &out, &out, Options{
		ConfigDir: root,
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	exit, err := app.handleLine(context.Background(), "connection test verbose")
	if err != nil {
		t.Fatalf("handleLine returned error: %v", err)
	}
	if exit {
		t.Fatalf("expected REPL to continue")
	}
	if connector.openCalls != 1 || connector.lastName != "prod" {
		t.Fatalf("unexpected connector calls: calls=%d name=%q", connector.openCalls, connector.lastName)
	}
}

func TestHelpConnectionIncludesTest(t *testing.T) {
	t.Parallel()

	entry := helpEntries["connection"].body
	if !strings.Contains(entry, "connection test") {
		t.Fatalf("connection help missing test command: %q", entry)
	}
}

func TestHelpConnectionTestMentionsVerbose(t *testing.T) {
	t.Parallel()

	entry := helpEntries["connection test"].body
	if !strings.Contains(entry, "--verbose") || !strings.Contains(entry, "connection test prod verbose") {
		t.Fatalf("connection test help missing verbose usage: %q", entry)
	}
}

func TestDiagnoseConnectionStepOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  *config.ConnectionConfig
		want []string
	}{
		{
			name: "direct",
			cfg:  sampleConnection("dev"),
			want: []string{"config", "mysql"},
		},
		{
			name: "ssh",
			cfg: &config.ConnectionConfig{
				Name:        "prod",
				Driver:      "mysql",
				Mode:        "ssh",
				Host:        "10.0.1.20",
				Port:        3306,
				User:        "root",
				PasswordEnv: "MYSQL_PROD_PASSWORD",
				SSH: &config.SSHConfig{
					Host:       "bastion.example.com",
					Port:       22,
					User:       "ubuntu",
					PrivateKey: "~/.ssh/id_rsa",
				},
				Timeout: &config.TimeoutConfig{ConnectSeconds: 10, QuerySeconds: 30},
			},
			want: []string{"config", "ssh", "mysql"},
		},
		{
			name: "proxy",
			cfg: &config.ConnectionConfig{
				Name:        "prod_proxy",
				Driver:      "mysql",
				Mode:        "proxy",
				Host:        "10.0.1.20",
				Port:        3306,
				User:        "root",
				PasswordEnv: "MYSQL_PROD_PASSWORD",
				Proxy:       &config.ProxyConfig{URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080"},
				Timeout:     &config.TimeoutConfig{ConnectSeconds: 10, QuerySeconds: 30},
			},
			want: []string{"config", "proxy", "mysql"},
		},
		{
			name: "proxy ssh",
			cfg: &config.ConnectionConfig{
				Name:        "prod_proxy",
				Driver:      "mysql",
				Mode:        "proxy-ssh",
				Host:        "10.0.1.20",
				Port:        3306,
				User:        "root",
				PasswordEnv: "MYSQL_PROD_PASSWORD",
				Proxy:       &config.ProxyConfig{URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080"},
				SSH: &config.SSHConfig{
					Host:       "bastion.example.com",
					Port:       22,
					User:       "ubuntu",
					PrivateKey: "~/.ssh/id_rsa",
				},
				Timeout: &config.TimeoutConfig{ConnectSeconds: 10, QuerySeconds: 30},
			},
			want: []string{"config", "proxy", "ssh", "mysql"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{
				ConfigDir: t.TempDir(),
				Connector: &diagnosticConnector{},
			})
			if err != nil {
				t.Fatalf("NewWithOptions returned error: %v", err)
			}

			result, diagErr := app.diagnoseConnection(context.Background(), tc.cfg, diagnosticOptions{})
			if diagErr != nil {
				t.Fatalf("diagnoseConnection returned error: %v", diagErr)
			}

			got := make([]string, 0, len(result.Steps))
			for _, step := range result.Steps {
				got = append(got, step.Name)
				if step.Status != "ok" {
					t.Fatalf("unexpected non-ok step: %+v", step)
				}
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("step order = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDiagnoseConnectionStopsAtFailedLayer(t *testing.T) {
	t.Parallel()

	cfg := &config.ConnectionConfig{
		Name:        "prod_proxy",
		Driver:      "mysql",
		Mode:        "proxy-ssh",
		Host:        "10.0.1.20",
		Port:        3306,
		User:        "root",
		PasswordEnv: "MYSQL_PROD_PASSWORD",
		Proxy:       &config.ProxyConfig{URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080"},
		SSH: &config.SSHConfig{
			Host:       "bastion.example.com",
			Port:       22,
			User:       "ubuntu",
			PrivateKey: "~/.ssh/id_rsa",
		},
		Timeout: &config.TimeoutConfig{ConnectSeconds: 10, QuerySeconds: 30},
	}

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{
		ConfigDir: t.TempDir(),
		Connector: &diagnosticConnector{
			openErr: util.WrapLayer("mysql", "ping database", util.WrapLayer("ssh", "complete SSH handshake with bastion.example.com:22", errors.New("ssh: unable to authenticate"))),
		},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, diagErr := app.diagnoseConnection(context.Background(), cfg, diagnosticOptions{})
	if diagErr == nil {
		t.Fatalf("expected diagnostic failure")
	}
	if len(result.Steps) != 3 {
		t.Fatalf("steps = %+v", result.Steps)
	}
	if result.Steps[1].Name != "proxy" || result.Steps[1].Status != "ok" {
		t.Fatalf("unexpected proxy step: %+v", result.Steps[1])
	}
	if result.Steps[2].Name != "ssh" || result.Steps[2].Status != "fail" {
		t.Fatalf("unexpected failed step: %+v", result.Steps[2])
	}
	if result.Steps[2].Error != "ssh: unable to authenticate" {
		t.Fatalf("unexpected step error: %+v", result.Steps[2])
	}
}

func TestDiagnoseConnectionVerboseIncludesDetails(t *testing.T) {
	t.Parallel()

	cfg := sampleConnection("prod")
	connector := &diagnosticConnector{
		trace: &driver.DiagnosticTrace{
			Steps: []driver.DiagnosticStep{
				{
					Name:   "mysql",
					Status: "ok",
					Details: map[string]any{
						"target":      "127.0.0.1:3306",
						"duration_ms": int64(42),
					},
				},
			},
		},
	}

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{
		ConfigDir: t.TempDir(),
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, diagErr := app.diagnoseConnection(context.Background(), cfg, diagnosticOptions{
		Verbose:    true,
		ConfigPath: "/tmp/prod/config.json",
	})
	if diagErr != nil {
		t.Fatalf("diagnoseConnection returned error: %v", diagErr)
	}
	if got := result.Steps[0].Details["config_path"]; got != "/tmp/prod/config.json" {
		t.Fatalf("config details missing path: %+v", result.Steps[0].Details)
	}
	if got := result.Steps[1].Details["duration_ms"]; got != int64(42) {
		t.Fatalf("mysql details missing duration: %+v", result.Steps[1].Details)
	}
}

func TestDiagnoseConnectionNonVerboseOmitsDetails(t *testing.T) {
	t.Parallel()

	cfg := sampleConnection("prod")
	connector := &diagnosticConnector{
		trace: &driver.DiagnosticTrace{
			Steps: []driver.DiagnosticStep{
				{
					Name:   "mysql",
					Status: "ok",
					Details: map[string]any{
						"target":      "127.0.0.1:3306",
						"duration_ms": int64(42),
					},
				},
			},
		},
	}

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{
		ConfigDir: t.TempDir(),
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, diagErr := app.diagnoseConnection(context.Background(), cfg, diagnosticOptions{})
	if diagErr != nil {
		t.Fatalf("diagnoseConnection returned error: %v", diagErr)
	}
	for _, step := range result.Steps {
		if len(step.Details) != 0 {
			t.Fatalf("expected non-verbose diagnostic to omit details, got %+v", result.Steps)
		}
	}
}

func TestDiagnoseConnectionVerboseFailureIncludesDetails(t *testing.T) {
	t.Parallel()

	cfg := sampleConnection("prod_proxy")
	cfg.Mode = "proxy"
	cfg.Proxy = &config.ProxyConfig{URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080"}

	connector := &diagnosticConnector{
		openErr: util.WrapLayer("proxy", "dial socks5://proxy_user:***@127.0.0.1:1080", errors.New("connection refused")),
		trace: &driver.DiagnosticTrace{
			Steps: []driver.DiagnosticStep{
				{
					Name:   "proxy",
					Status: "fail",
					Error:  "connection refused",
					Details: map[string]any{
						"url":         "socks5://proxy_user:***@127.0.0.1:1080",
						"target":      "10.0.1.20:3306",
						"duration_ms": int64(12),
					},
				},
			},
		},
	}

	app, err := NewWithOptions(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, Options{
		ConfigDir: t.TempDir(),
		Connector: connector,
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	result, diagErr := app.diagnoseConnection(context.Background(), cfg, diagnosticOptions{Verbose: true})
	if diagErr == nil {
		t.Fatalf("expected diagnostic failure")
	}
	if len(result.Steps) != 2 {
		t.Fatalf("unexpected steps: %+v", result.Steps)
	}
	if got := result.Steps[1].Details["duration_ms"]; got != int64(12) {
		t.Fatalf("missing failure details: %+v", result.Steps[1].Details)
	}
	if strings.Contains(fmt.Sprint(result.Steps[1].Details["url"]), "proxy_password") {
		t.Fatalf("proxy password leaked in details: %+v", result.Steps[1].Details)
	}
}

func TestPrintDiagnosticResultVerboseIncludesDetails(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader(""), &out, &out, Options{ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	app.printDiagnosticResult(&DiagnosticResult{
		Steps: []DiagnosticStep{
			{
				Name:   "proxy",
				Status: "fail",
				Error:  "connection refused",
				Details: map[string]any{
					"url":         "socks5://127.0.0.1:1080",
					"target":      "10.0.1.20:3306",
					"duration_ms": int64(12),
				},
			},
		},
	}, true)

	output := out.String()
	if !strings.Contains(output, "url: socks5://127.0.0.1:1080") {
		t.Fatalf("verbose output missing proxy url: %q", output)
	}
	if !strings.Contains(output, "duration: 12ms") {
		t.Fatalf("verbose output missing duration: %q", output)
	}
	if !strings.Contains(output, "error: connection refused") {
		t.Fatalf("verbose output missing error detail: %q", output)
	}
}

func TestInteractiveConnectionCreateProxyModeSkipsSSHPrompts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := strings.Join([]string{
		"prod_proxy",
		"proxy",
		"10.0.1.20",
		"3306",
		"root",
		"prompt every time",
		"10",
		"30",
		"socks5://127.0.0.1:1080",
		"n",
		"y",
		"n",
	}, "\n") + "\n"

	var out bytes.Buffer
	app, err := NewWithOptions(strings.NewReader(input), &out, &out, Options{ConfigDir: root})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}

	if err := app.handleConnectionCreate(context.Background()); err != nil {
		t.Fatalf("handleConnectionCreate returned error: %v", err)
	}

	cfg, err := app.store.LoadConnection("prod_proxy")
	if err != nil {
		t.Fatalf("LoadConnection returned error: %v", err)
	}
	if cfg.Mode != "proxy" || cfg.Proxy == nil || cfg.Proxy.URL != "socks5://127.0.0.1:1080" {
		t.Fatalf("unexpected saved config: %+v", cfg)
	}
	if cfg.SSH != nil {
		t.Fatalf("proxy mode should not save ssh config: %+v", cfg.SSH)
	}
	if strings.Contains(out.String(), "SSH host") {
		t.Fatalf("proxy mode should not prompt for SSH host: %q", out.String())
	}
}
