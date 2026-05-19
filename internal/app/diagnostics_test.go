package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

type diagnosticConnector struct {
	openErr   error
	openCalls int
	lastName  string
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

func (d *diagnosticConnector) Ping(context.Context, *config.ConnectionConfig, *sql.DB) error {
	return nil
}

func (d *diagnosticConnector) ListDatabases(context.Context, *config.ConnectionConfig, *sql.DB) ([]string, error) {
	return nil, nil
}

func (d *diagnosticConnector) QueryStrings(context.Context, *config.ConnectionConfig, *sql.DB, string) ([]string, error) {
	return nil, nil
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

func TestHelpConnectionIncludesTest(t *testing.T) {
	t.Parallel()

	entry := helpEntries["connection"].body
	if !strings.Contains(entry, "connection test") {
		t.Fatalf("connection help missing test command: %q", entry)
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

			result, diagErr := app.diagnoseConnection(context.Background(), tc.cfg)
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

	result, diagErr := app.diagnoseConnection(context.Background(), cfg)
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
