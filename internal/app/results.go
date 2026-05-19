package app

import (
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
)

type ActionStatus string

const (
	ActionStatusOK     ActionStatus = "ok"
	ActionStatusFailed ActionStatus = "failed"
	ActionStatusDryRun ActionStatus = "dry-run"
)

type ActionResult struct {
	Description string       `json:"description"`
	SQL         string       `json:"sql,omitempty"`
	Status      ActionStatus `json:"status"`
}

type PlanExecutionResult struct {
	OK          bool           `json:"ok"`
	Connection  string         `json:"connection,omitempty"`
	Command     string         `json:"command,omitempty"`
	Template    string         `json:"template,omitempty"`
	Layer       string         `json:"layer,omitempty"`
	Source      string         `json:"source,omitempty"`
	DryRun      bool           `json:"dry_run,omitempty"`
	Transaction bool           `json:"transaction,omitempty"`
	Committed   bool           `json:"committed,omitempty"`
	RolledBack  bool           `json:"rolled_back,omitempty"`
	Actions     []ActionResult `json:"actions,omitempty"`
}

type ConnectionSummary struct {
	Name    string `json:"name"`
	Driver  string `json:"driver"`
	Mode    string `json:"mode"`
	Address string `json:"address"`
	ViaSSH  string `json:"via_ssh,omitempty"`
}

type ConnectionsResult struct {
	OK          bool                `json:"ok"`
	Connections []ConnectionSummary `json:"connections"`
}

type DatabasesResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	Databases  []string `json:"databases,omitempty"`
}

type ConnectResult struct {
	OK         bool   `json:"ok"`
	Connection string `json:"connection,omitempty"`
	Message    string `json:"message,omitempty"`
}

type RedactedConnection struct {
	Name           string               `json:"name"`
	Driver         string               `json:"driver"`
	Mode           string               `json:"mode"`
	Host           string               `json:"host"`
	Port           int                  `json:"port"`
	User           string               `json:"user"`
	ConnectTimeout int                  `json:"connect_timeout_seconds"`
	QueryTimeout   int                  `json:"query_timeout_seconds"`
	Password       RedactedPassword     `json:"password"`
	SSH            *RedactedSSHSettings `json:"ssh,omitempty"`
}

type RedactedPassword struct {
	Mode  string `json:"mode"`
	Env   string `json:"env,omitempty"`
	Value string `json:"value,omitempty"`
}

type RedactedSSHSettings struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	PrivateKey   string `json:"private_key,omitempty"`
	PasswordEnv  string `json:"password_env,omitempty"`
	PasswordMode string `json:"password_mode,omitempty"`
}

type StatusResult struct {
	OK                 bool                `json:"ok"`
	Connection         *RedactedConnection `json:"connection,omitempty"`
	ConnectionName     string              `json:"connection_name,omitempty"`
	CurrentSession     string              `json:"current_session,omitempty"`
	ConnectionExists   bool                `json:"connection_exists,omitempty"`
	SelectedByFlag     bool                `json:"selected_by_flag,omitempty"`
	HasStoredSession   bool                `json:"has_stored_session,omitempty"`
	ConnectedInProcess bool                `json:"connected_in_process,omitempty"`
	DryRun             bool                `json:"dry_run,omitempty"`
	Message            string              `json:"message,omitempty"`
}

func summarizeConnection(cfg config.ConnectionConfig) ConnectionSummary {
	summary := ConnectionSummary{
		Name:    cfg.Name,
		Driver:  cfg.Driver,
		Mode:    cfg.Mode,
		Address: cfg.Address(),
	}
	if cfg.Mode == "ssh" && cfg.SSH != nil {
		summary.ViaSSH = cfg.SSH.Host
	}
	return summary
}

func redactConnection(cfg *config.ConnectionConfig) *RedactedConnection {
	if cfg == nil {
		return nil
	}

	cfg.ApplyDefaults()

	result := &RedactedConnection{
		Name:           cfg.Name,
		Driver:         cfg.Driver,
		Mode:           cfg.Mode,
		Host:           cfg.Host,
		Port:           cfg.Port,
		User:           cfg.User,
		ConnectTimeout: cfg.Timeout.ConnectSeconds,
		QueryTimeout:   cfg.Timeout.QuerySeconds,
		Password:       redactPassword(cfg),
	}

	if cfg.SSH != nil {
		sshSettings := &RedactedSSHSettings{
			Host:       cfg.SSH.Host,
			Port:       cfg.SSH.Port,
			User:       cfg.SSH.User,
			PrivateKey: cfg.SSH.PrivateKey,
		}
		if strings.TrimSpace(cfg.SSH.PasswordEnv) != "" {
			sshSettings.PasswordEnv = cfg.SSH.PasswordEnv
			sshSettings.PasswordMode = "env"
		} else if strings.TrimSpace(cfg.SSH.Password) != "" {
			sshSettings.PasswordMode = "saved"
		}
		result.SSH = sshSettings
	}

	return result
}

func redactPassword(cfg *config.ConnectionConfig) RedactedPassword {
	switch {
	case cfg.PasswordPrompt:
		return RedactedPassword{Mode: "prompt"}
	case strings.TrimSpace(cfg.PasswordEnv) != "":
		return RedactedPassword{Mode: "env", Env: cfg.PasswordEnv}
	case strings.TrimSpace(cfg.Password) != "":
		return RedactedPassword{Mode: "saved", Value: "[redacted]"}
	default:
		return RedactedPassword{Mode: "none"}
	}
}
