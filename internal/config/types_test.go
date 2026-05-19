package config

import (
	"strings"
	"testing"
)

func TestConnectionConfigValidateProxySSH(t *testing.T) {
	t.Parallel()

	cfg := &ConnectionConfig{
		Name:        "prod_proxy",
		Driver:      "mysql",
		Mode:        "proxy-ssh",
		Host:        "10.0.1.20",
		Port:        3306,
		User:        "root",
		PasswordEnv: "MYSQL_PROD_PASSWORD",
		Proxy: &ProxyConfig{
			URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080",
		},
		SSH: &SSHConfig{
			Host:       "bastion.example.com",
			Port:       22,
			User:       "ubuntu",
			PrivateKey: "~/.ssh/id_rsa",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestConnectionConfigValidateProxy(t *testing.T) {
	t.Parallel()

	cfg := &ConnectionConfig{
		Name:        "prod_proxy",
		Driver:      "mysql",
		Mode:        "proxy",
		Host:        "10.0.1.20",
		Port:        3306,
		User:        "root",
		PasswordEnv: "MYSQL_PROD_PASSWORD",
		Proxy: &ProxyConfig{
			URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestConnectionConfigValidateProxyRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *ConnectionConfig
		want string
	}{
		{
			name: "proxy requires proxy url",
			cfg: &ConnectionConfig{
				Name: "prod_proxy", Driver: "mysql", Mode: "proxy", Host: "10.0.1.20", Port: 3306, User: "root", PasswordEnv: "MYSQL_PROD_PASSWORD",
			},
			want: "proxy.url is required",
		},
		{
			name: "proxy rejects ssh",
			cfg: &ConnectionConfig{
				Name: "prod_proxy", Driver: "mysql", Mode: "proxy", Host: "10.0.1.20", Port: 3306, User: "root", PasswordEnv: "MYSQL_PROD_PASSWORD",
				Proxy: &ProxyConfig{URL: "socks5://127.0.0.1:1080"},
				SSH:   &SSHConfig{Host: "bastion.example.com", Port: 22, User: "ubuntu", PrivateKey: "~/.ssh/id_rsa"},
			},
			want: "ssh settings are not supported for proxy mode",
		},
		{
			name: "proxy ssh requires proxy url",
			cfg: &ConnectionConfig{
				Name: "prod_proxy", Driver: "mysql", Mode: "proxy-ssh", Host: "10.0.1.20", Port: 3306, User: "root", PasswordEnv: "MYSQL_PROD_PASSWORD",
				SSH: &SSHConfig{Host: "bastion.example.com", Port: 22, User: "ubuntu", PrivateKey: "~/.ssh/id_rsa"},
			},
			want: "proxy.url is required",
		},
		{
			name: "proxy ssh requires ssh",
			cfg: &ConnectionConfig{
				Name: "prod_proxy", Driver: "mysql", Mode: "proxy-ssh", Host: "10.0.1.20", Port: 3306, User: "root", PasswordEnv: "MYSQL_PROD_PASSWORD",
				Proxy: &ProxyConfig{URL: "socks5://127.0.0.1:1080"},
			},
			want: "ssh settings are required",
		},
		{
			name: "ssh rejects proxy",
			cfg: &ConnectionConfig{
				Name: "prod_proxy", Driver: "mysql", Mode: "ssh", Host: "10.0.1.20", Port: 3306, User: "root", PasswordEnv: "MYSQL_PROD_PASSWORD",
				Proxy: &ProxyConfig{URL: "socks5://127.0.0.1:1080"},
				SSH:   &SSHConfig{Host: "bastion.example.com", Port: 22, User: "ubuntu", PrivateKey: "~/.ssh/id_rsa"},
			},
			want: "proxy settings are not supported for ssh mode",
		},
		{
			name: "unsupported proxy scheme",
			cfg: &ConnectionConfig{
				Name: "prod_proxy", Driver: "mysql", Mode: "proxy-ssh", Host: "10.0.1.20", Port: 3306, User: "root", PasswordEnv: "MYSQL_PROD_PASSWORD",
				Proxy: &ProxyConfig{URL: "http://127.0.0.1:1080"},
				SSH:   &SSHConfig{Host: "bastion.example.com", Port: 22, User: "ubuntu", PrivateKey: "~/.ssh/id_rsa"},
			},
			want: "unsupported proxy scheme: http",
		},
		{
			name: "direct rejects proxy",
			cfg: &ConnectionConfig{
				Name: "prod_proxy", Driver: "mysql", Mode: "direct", Host: "10.0.1.20", Port: 3306, User: "root", PasswordEnv: "MYSQL_PROD_PASSWORD",
				Proxy: &ProxyConfig{URL: "socks5://127.0.0.1:1080"},
			},
			want: "proxy settings are not supported for direct mode",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if err == nil {
				t.Fatalf("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParseAndRedactProxyURL(t *testing.T) {
	t.Parallel()

	parsed, err := ParseProxyURL("socks5://proxy_user:proxy_password@127.0.0.1:1080")
	if err != nil {
		t.Fatalf("ParseProxyURL returned error: %v", err)
	}
	if parsed.Scheme != "socks5" || parsed.Address != "127.0.0.1:1080" || parsed.Username != "proxy_user" || parsed.Password != "proxy_password" {
		t.Fatalf("unexpected parsed proxy: %+v", parsed)
	}

	redacted := RedactProxyURL("socks5://proxy_user:proxy_password@127.0.0.1:1080")
	if redacted != "socks5://proxy_user:***@127.0.0.1:1080" {
		t.Fatalf("RedactProxyURL = %q", redacted)
	}
	if strings.Contains(redacted, "proxy_password") {
		t.Fatalf("redacted proxy URL leaked password: %q", redacted)
	}
}
