package config

import (
	"testing"
	"time"
)

func TestConnectionTimeoutDefaults(t *testing.T) {
	t.Parallel()

	cfg := &ConnectionConfig{
		Name:   "dev",
		Driver: "mysql",
		Mode:   "direct",
		Host:   "127.0.0.1",
		Port:   3306,
		User:   "root",
	}

	cfg.ApplyDefaults()

	if got := cfg.ConnectTimeout(); got != 10*time.Second {
		t.Fatalf("ConnectTimeout = %s", got)
	}
	if got := cfg.QueryTimeout(); got != 30*time.Second {
		t.Fatalf("QueryTimeout = %s", got)
	}
}
