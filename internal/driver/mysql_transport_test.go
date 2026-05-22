package driver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh/knownhosts"
	xproxy "golang.org/x/net/proxy"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestSSHAuthMethodsUsesPasswordEnv(t *testing.T) {
	t.Setenv("DBX_TEST_SSH_PASSWORD", "secret-value")
	methods, err := sshAuthMethods(&config.SSHConfig{PasswordEnv: "DBX_TEST_SSH_PASSWORD"})
	if err != nil {
		t.Fatalf("sshAuthMethods returned error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("auth method count = %d, want 1", len(methods))
	}
}

func TestSSHAuthMethodsRejectsMissingPasswordEnv(t *testing.T) {
	t.Parallel()

	_, err := sshAuthMethods(&config.SSHConfig{PasswordEnv: "DBX_TEST_SSH_PASSWORD_MISSING"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "environment variable DBX_TEST_SSH_PASSWORD_MISSING is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSHAuthMethodsUsesPrivateKey(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	keyPath := filepath.Join(root, "id_ed25519")
	if err := os.WriteFile(keyPath, testPrivateKeyPEM(t), 0o600); err != nil {
		t.Fatal(err)
	}

	methods, err := sshAuthMethods(&config.SSHConfig{PrivateKey: keyPath})
	if err != nil {
		t.Fatalf("sshAuthMethods returned error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("auth method count = %d, want 1", len(methods))
	}
}

func TestKnownHostsPathsUsesEnvOverride(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "known_hosts")
	second := filepath.Join(root, "known_hosts2")
	if err := os.WriteFile(first, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DBX_KNOWN_HOSTS", first+string(os.PathListSeparator)+filepath.Join(root, "missing")+string(os.PathListSeparator)+second)
	paths, err := knownHostsPaths()
	if err != nil {
		t.Fatalf("knownHostsPaths returned error: %v", err)
	}
	if len(paths) != 2 || paths[0] != first || paths[1] != second {
		t.Fatalf("unexpected known_hosts paths: %#v", paths)
	}
}

func TestKnownHostsErrorMissingHost(t *testing.T) {
	t.Parallel()

	err := knownHostsError([]string{"/tmp/known_hosts"}, "db.example.com:22", &knownhosts.KeyError{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "host db.example.com is not in known_hosts") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKnownHostsErrorMismatch(t *testing.T) {
	t.Parallel()

	err := knownHostsError([]string{"/tmp/known_hosts"}, "db.example.com:22", &knownhosts.KeyError{
		Want: []knownhosts.KnownKey{{Filename: "/tmp/known_hosts"}},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "host key mismatch for db.example.com") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterProxyDialerReusesNetwork(t *testing.T) {
	t.Parallel()

	registeredDialers = sync.Map{}
	cfg := &config.ConnectionConfig{
		Name:   "prod-proxy-test",
		Driver: "mysql",
		Mode:   "proxy",
		Host:   "127.0.0.1",
		Port:   3306,
		User:   "root",
		Proxy:  &config.ProxyConfig{URL: "socks5://127.0.0.1:1080"},
	}

	first, err := registerProxyDialer(cfg)
	if err != nil {
		t.Fatalf("registerProxyDialer returned error: %v", err)
	}
	second, err := registerProxyDialer(cfg)
	if err != nil {
		t.Fatalf("registerProxyDialer returned error: %v", err)
	}
	if first == "" || second == "" || first != second {
		t.Fatalf("unexpected networks: %q %q", first, second)
	}
}

func TestDialProxyWithContextReturnsContextError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	release := make(chan struct{})

	conn, err := dialProxyWithContext(ctx, blockingDialer{release: release}, "tcp", "127.0.0.1:1080")
	close(release)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if conn != nil {
		t.Fatalf("expected nil conn, got %#v", conn)
	}
}

type blockingDialer struct {
	release <-chan struct{}
}

func (d blockingDialer) Dial(string, string) (net.Conn, error) {
	<-d.release
	return nil, nil
}

func testPrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey returned error: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded})
}

var _ xproxy.Dialer = blockingDialer{}
