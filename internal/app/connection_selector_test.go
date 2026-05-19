package app

import (
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestSortedConnections(t *testing.T) {
	t.Parallel()

	input := []config.ConnectionConfig{
		{Name: "dev", Driver: "mysql", Mode: "direct", Host: "127.0.0.1", Port: 3306},
		{Name: "prod", Driver: "mysql", Mode: "ssh", Host: "10.0.1.20", Port: 3306},
		{Name: "stage", Driver: "mysql", Mode: "direct", Host: "10.0.0.5", Port: 3306},
	}

	sorted := sortedConnections(input, "prod")
	if sorted[0].Name != "prod" || sorted[1].Name != "dev" || sorted[2].Name != "stage" {
		t.Fatalf("sorted connections = %#v", sorted)
	}
}

func TestFormatConnectionSummary(t *testing.T) {
	t.Parallel()

	direct := formatConnectionSummary(config.ConnectionConfig{
		Name:   "dev",
		Driver: "mysql",
		Mode:   "direct",
		Host:   "127.0.0.1",
		Port:   3306,
	})
	if direct == "" || direct == "dev" {
		t.Fatalf("direct summary = %q", direct)
	}

	ssh := formatConnectionSummary(config.ConnectionConfig{
		Name:   "prod",
		Driver: "mysql",
		Mode:   "ssh",
		Host:   "10.0.1.20",
		Port:   3306,
		SSH: &config.SSHConfig{
			Host: "bastion.example.com",
		},
	})
	if ssh == "" || ssh == direct {
		t.Fatalf("ssh summary = %q", ssh)
	}
}
