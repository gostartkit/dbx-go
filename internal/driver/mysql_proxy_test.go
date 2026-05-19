package driver

import (
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestProxyDialerSettings(t *testing.T) {
	t.Parallel()

	cfg := &config.ConnectionConfig{
		Name: "prod_proxy",
		Mode: "proxy-ssh",
		Proxy: &config.ProxyConfig{
			URL: "socks5://proxy_user:proxy_password@127.0.0.1:1080",
		},
	}

	settings, err := proxyDialerSettings(cfg)
	if err != nil {
		t.Fatalf("proxyDialerSettings returned error: %v", err)
	}
	if settings.Address != "127.0.0.1:1080" {
		t.Fatalf("Address = %q", settings.Address)
	}
	if settings.Auth == nil || settings.Auth.User != "proxy_user" || settings.Auth.Password != "proxy_password" {
		t.Fatalf("unexpected auth: %+v", settings.Auth)
	}
	if settings.RedactedURL != "socks5://proxy_user:***@127.0.0.1:1080" {
		t.Fatalf("RedactedURL = %q", settings.RedactedURL)
	}
}

func TestProxyDialerSettingsRejectsUnsupportedScheme(t *testing.T) {
	t.Parallel()

	cfg := &config.ConnectionConfig{
		Name: "prod_proxy",
		Mode: "proxy-ssh",
		Proxy: &config.ProxyConfig{
			URL: "http://127.0.0.1:1080",
		},
	}

	_, err := proxyDialerSettings(cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported proxy scheme: http") {
		t.Fatalf("unexpected error: %v", err)
	}
}
