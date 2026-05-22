package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"pkg.gostartkit.com/dbx/internal/util"
)

type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
)

type Diagnostic struct {
	Code       string           `json:"code,omitempty"`
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Suggestion string           `json:"suggestion,omitempty"`
	Message    string           `json:"-"`
}

type ValidationReport struct {
	Checks []Diagnostic `json:"checks"`
}

func (r *ValidationReport) HasFailures() bool {
	if r == nil {
		return false
	}
	for _, check := range r.Checks {
		if check.Status == DiagnosticStatusFail {
			return true
		}
	}
	return false
}

func (r *ValidationReport) Error() error {
	if r == nil {
		return nil
	}
	for _, check := range r.Checks {
		if check.Status != DiagnosticStatusFail {
			continue
		}
		if strings.TrimSpace(check.Message) != "" {
			return fmt.Errorf("%s", check.Message)
		}
		return fmt.Errorf("%s", check.Name)
	}
	return nil
}

func (c *ConnectionConfig) ValidateDetailed() *ValidationReport {
	report := &ValidationReport{Checks: make([]Diagnostic, 0, 24)}
	if c == nil {
		addDiagnosticFail(report, "connection_config_required", "connection config is required", "", "connection config is required")
		return report
	}

	c.ApplyDefaults()

	validateConnectionName(report, c.Name)
	validateDriver(report, c.Driver)
	validateMode(report, c.Mode)
	validateHost(report, c.Host)
	validatePort(report, c.Port)
	validateUser(report, c.User)
	checkDatabasePassword(report, c)
	checkTimeouts(report, c)

	switch c.Mode {
	case "direct":
		checkDirectMode(report, c)
	case "ssh":
		checkSSHProxyAbsent(report, c)
		checkSSHMode(report, c)
	case "proxy":
		checkProxyMode(report, c)
		checkProxySSHAbsent(report, c)
	case "proxy-ssh":
		checkProxyMode(report, c)
		checkSSHMode(report, c)
	}

	return report
}

func validateConnectionName(report *ValidationReport, value string) {
	if strings.TrimSpace(value) == "" {
		addDiagnosticFail(report, "connection_name_required", "connection name is valid", "", "connection name is required")
		return
	}
	if err := util.ValidateIdentifier(value); err != nil {
		addDiagnosticFail(report, "connection_name_invalid", "connection name is valid", "", err.Error())
		return
	}
	addDiagnosticOK(report, "connection_name_valid", "connection name is valid")
}

func validateDriver(report *ValidationReport, value string) {
	name := "driver " + strings.TrimSpace(value)
	if strings.TrimSpace(value) == "" {
		name = "driver <empty>"
	}
	if value != "mysql" {
		addDiagnosticFail(report, "driver_unsupported", name, "", fmt.Sprintf("unsupported driver %q", value))
		return
	}
	addDiagnosticOK(report, "driver_mysql", "driver mysql")
}

func validateMode(report *ValidationReport, value string) {
	name := "mode " + strings.TrimSpace(value)
	if strings.TrimSpace(value) == "" {
		name = "mode <empty>"
	}
	switch value {
	case "direct", "ssh", "proxy", "proxy-ssh":
		addDiagnosticOK(report, "mode_valid", name)
	default:
		addDiagnosticFail(report, "mode_unsupported", name, "", fmt.Sprintf("unsupported connection mode %q", value))
	}
}

func validateHost(report *ValidationReport, value string) {
	name := "database host " + value
	if strings.TrimSpace(value) == "" {
		addDiagnosticFail(report, "host_required", name, "", "host is required")
		return
	}
	addDiagnosticOK(report, "host_valid", name)
}

func validatePort(report *ValidationReport, port int) {
	name := fmt.Sprintf("database port %d", port)
	if port < 1 || port > 65535 {
		addDiagnosticFail(report, "port_invalid", name, "", "port must be greater than zero")
		return
	}
	addDiagnosticOK(report, "port_valid", name)
}

func validateUser(report *ValidationReport, value string) {
	name := "database user " + value
	if strings.TrimSpace(value) == "" {
		addDiagnosticFail(report, "user_required", name, "", "user is required")
		return
	}
	addDiagnosticOK(report, "user_valid", name)
}

func checkDatabasePassword(report *ValidationReport, cfg *ConnectionConfig) {
	switch {
	case strings.TrimSpace(cfg.Password) != "":
		addDiagnosticWarn(report, "database_password_inline", "database password is inline in config", "prefer password_env or password_prompt")
	case strings.TrimSpace(cfg.PasswordEnv) != "":
		if os.Getenv(cfg.PasswordEnv) == "" {
			addDiagnosticWarn(report, "database_password_env_missing", "database password env "+cfg.PasswordEnv+" is set", "export "+cfg.PasswordEnv+" before connecting")
			return
		}
		addDiagnosticOK(report, "database_password_env_set", "database password env "+cfg.PasswordEnv+" is set")
	case cfg.PasswordPrompt:
		addDiagnosticOK(report, "database_password_prompt", "database password prompt enabled")
	default:
		addDiagnosticWarn(report, "database_password_source_missing", "database password source is configured", "set password_env or enable password_prompt")
	}
}

func checkTimeouts(report *ValidationReport, cfg *ConnectionConfig) {
	if cfg.Timeout == nil {
		return
	}
	if cfg.Timeout.ConnectSeconds <= 0 {
		addDiagnosticFail(report, "connect_timeout_invalid", "connect timeout is valid", "", "timeout.connect_seconds must be greater than zero")
	} else {
		addDiagnosticOK(report, "connect_timeout_valid", fmt.Sprintf("connect timeout %d seconds", cfg.Timeout.ConnectSeconds))
	}
	if cfg.Timeout.QuerySeconds <= 0 {
		addDiagnosticFail(report, "query_timeout_invalid", "query timeout is valid", "", "timeout.query_seconds must be greater than zero")
	} else {
		addDiagnosticOK(report, "query_timeout_valid", fmt.Sprintf("query timeout %d seconds", cfg.Timeout.QuerySeconds))
	}
}

func checkDirectMode(report *ValidationReport, cfg *ConnectionConfig) {
	addDiagnosticOK(report, "direct_mode_no_ssh_required", "no SSH config required")
	if cfg.Proxy != nil && strings.TrimSpace(cfg.Proxy.URL) != "" {
		addDiagnosticFail(report, "direct_mode_proxy_forbidden", "proxy config must be empty for direct mode", "remove proxy config or use mode proxy", "proxy settings are not supported for direct mode")
	}
}

func checkSSHProxyAbsent(report *ValidationReport, cfg *ConnectionConfig) {
	if cfg.Proxy != nil && strings.TrimSpace(cfg.Proxy.URL) != "" {
		addDiagnosticFail(report, "ssh_mode_proxy_forbidden", "proxy config must be empty for ssh mode", "remove proxy config or use mode proxy-ssh", "proxy settings are not supported for ssh mode")
	}
}

func checkProxySSHAbsent(report *ValidationReport, cfg *ConnectionConfig) {
	if cfg.SSH != nil {
		addDiagnosticFail(report, "proxy_mode_ssh_forbidden", "ssh config must be empty for proxy mode", "remove ssh config or use mode proxy-ssh", "ssh settings are not supported for proxy mode")
	}
}

func checkSSHMode(report *ValidationReport, cfg *ConnectionConfig) {
	if cfg.SSH == nil {
		addDiagnosticFail(report, "ssh_required", "ssh settings are configured", "set ssh.host, ssh.user, and an SSH auth method", fmt.Sprintf("ssh settings are required for %s mode", cfg.Mode))
		return
	}

	if strings.TrimSpace(cfg.SSH.Host) == "" {
		addDiagnosticFail(report, "ssh_host_required", "ssh host "+cfg.SSH.Host, "", "ssh.host is required")
	} else {
		addDiagnosticOK(report, "ssh_host_valid", "ssh host "+cfg.SSH.Host)
	}
	if cfg.SSH.Port < 1 || cfg.SSH.Port > 65535 {
		addDiagnosticFail(report, "ssh_port_invalid", fmt.Sprintf("ssh port %d", cfg.SSH.Port), "", "ssh.port must be greater than zero")
	} else {
		addDiagnosticOK(report, "ssh_port_valid", fmt.Sprintf("ssh port %d", cfg.SSH.Port))
	}
	if strings.TrimSpace(cfg.SSH.User) == "" {
		addDiagnosticFail(report, "ssh_user_required", "ssh user "+cfg.SSH.User, "", "ssh.user is required")
	} else {
		addDiagnosticOK(report, "ssh_user_valid", "ssh user "+cfg.SSH.User)
	}

	hasAuth := false
	if strings.TrimSpace(cfg.SSH.PrivateKey) != "" {
		hasAuth = true
	}

	switch {
	case strings.TrimSpace(cfg.SSH.Password) != "":
		hasAuth = true
		addDiagnosticWarn(report, "ssh_password_inline", "ssh password is inline in config", "prefer ssh password_env or a private key")
	case strings.TrimSpace(cfg.SSH.PasswordEnv) != "":
		hasAuth = true
		if os.Getenv(cfg.SSH.PasswordEnv) == "" {
			addDiagnosticWarn(report, "ssh_password_env_missing", "ssh password env "+cfg.SSH.PasswordEnv+" is set", "export "+cfg.SSH.PasswordEnv+" before connecting")
		} else {
			addDiagnosticOK(report, "ssh_password_env_set", "ssh password env "+cfg.SSH.PasswordEnv+" is set")
		}
	}

	if !hasAuth {
		addDiagnosticFail(report, "ssh_auth_missing", "ssh auth method is configured", "set ssh.private_key, ssh.password_env, or ssh.password", "ssh.private_key or ssh.password_env or ssh.password is required")
	}
}

func checkProxyMode(report *ValidationReport, cfg *ConnectionConfig) {
	if cfg.Proxy == nil || strings.TrimSpace(cfg.Proxy.URL) == "" {
		addDiagnosticFail(report, "proxy_url_required", "proxy URL is configured", "set proxy.url for mode "+cfg.Mode, fmt.Sprintf("proxy.url is required for %s mode", cfg.Mode))
		return
	}

	parsed, err := ParseProxyURL(cfg.Proxy.URL)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported proxy scheme") {
			addDiagnosticFail(report, "proxy_scheme_unsupported", "proxy scheme is supported", "supported schemes: socks5", "proxy.url is invalid: "+err.Error())
		} else {
			addDiagnosticFail(report, "proxy_url_invalid", "proxy URL is valid", "", "proxy.url is invalid: "+err.Error())
		}
		return
	}

	addDiagnosticOK(report, "proxy_scheme_valid", "proxy scheme "+parsed.Scheme)
	host, portText, splitErr := net.SplitHostPort(parsed.Address)
	if splitErr != nil || strings.TrimSpace(host) == "" {
		addDiagnosticFail(report, "proxy_host_invalid", "proxy host is valid", "", "proxy.url is invalid: host is required")
		return
	}
	addDiagnosticOK(report, "proxy_host_valid", "proxy host "+host)

	port, atoiErr := strconv.Atoi(portText)
	if atoiErr != nil || port < 1 || port > 65535 {
		addDiagnosticFail(report, "proxy_port_invalid", "proxy port is valid", "", "proxy.url is invalid: port must be between 1 and 65535")
	} else {
		addDiagnosticOK(report, "proxy_port_valid", fmt.Sprintf("proxy port %d", port))
	}

	if parsed.Password != "" {
		addDiagnosticWarn(report, "proxy_password_inline", "proxy URL contains inline password", "avoid saving inline proxy passwords in config")
	}
}

func addDiagnosticOK(report *ValidationReport, code string, name string) {
	report.Checks = append(report.Checks, Diagnostic{Code: code, Name: name, Status: DiagnosticStatusOK})
}

func addDiagnosticWarn(report *ValidationReport, code string, name string, suggestion string) {
	report.Checks = append(report.Checks, Diagnostic{Code: code, Name: name, Status: DiagnosticStatusWarn, Suggestion: suggestion})
}

func addDiagnosticFail(report *ValidationReport, code string, name string, suggestion string, message string) {
	report.Checks = append(report.Checks, Diagnostic{
		Code:       code,
		Name:       name,
		Status:     DiagnosticStatusFail,
		Suggestion: suggestion,
		Message:    message,
	})
}
