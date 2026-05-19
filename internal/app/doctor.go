package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleConnectionDoctor(ctx context.Context, name string) error {
	selected, err := a.resolveConnectionNameForSelection(ctx, name, "connection doctor")
	if err != nil {
		return err
	}

	result, doctorErr := a.doctorConnection(selected)
	a.printDoctorResult(result)
	if doctorErr != nil {
		return nil
	}
	return nil
}

func (a *Application) doctorConnection(name string) (*DoctorResult, error) {
	result := &DoctorResult{
		OK:         true,
		Connection: name,
		Checks:     make([]DoctorCheck, 0, 16),
	}

	path := a.store.ConnectionConfigPath(name)
	if _, err := os.Stat(path); err != nil {
		result.OK = false
		result.Checks = append(result.Checks, DoctorCheck{
			Name:   "config file exists",
			Status: "fail",
		})
		return result, util.WrapLayer("config", "read connection config "+path, err)
	}
	result.Checks = append(result.Checks, DoctorCheck{Name: "config file exists", Status: "ok"})

	data, err := os.ReadFile(path)
	if err != nil {
		result.OK = false
		result.Checks = append(result.Checks, DoctorCheck{Name: "config JSON can be parsed", Status: "fail"})
		return result, util.WrapLayer("config", "read connection config "+path, err)
	}

	var cfg config.ConnectionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		result.OK = false
		result.Checks = append(result.Checks, DoctorCheck{Name: "config JSON can be parsed", Status: "fail"})
		return result, util.WrapLayer("config", "parse connection config "+path, err)
	}
	result.Checks = append(result.Checks, DoctorCheck{Name: "config JSON can be parsed", Status: "ok"})
	if strings.TrimSpace(cfg.Name) != "" {
		result.Connection = cfg.Name
	}

	cfg.ApplyDefaults()

	addDoctorCheck(result, "connection name is valid", checkIdentifier(cfg.Name), "")
	addDoctorCheck(result, "driver mysql", checkDriver(cfg.Driver), "")
	addDoctorCheck(result, "mode "+cfg.Mode, checkMode(cfg.Mode), "")
	addDoctorCheck(result, "database host "+cfg.Host, checkNonEmpty(cfg.Host), "")
	addDoctorCheck(result, fmt.Sprintf("database port %d", cfg.Port), checkPort(cfg.Port), "")
	addDoctorCheck(result, "database user "+cfg.User, checkNonEmpty(cfg.User), "")
	checkDatabasePassword(result, &cfg)
	checkTimeouts(result, &cfg)

	switch cfg.Mode {
	case "direct":
		checkDirectMode(result, &cfg)
	case "ssh":
		checkSSHProxyAbsent(result, &cfg)
		checkSSHMode(result, &cfg)
	case "proxy":
		checkProxyMode(result, &cfg)
		checkProxySSHAbsent(result, &cfg)
	case "proxy-ssh":
		checkProxyMode(result, &cfg)
		checkSSHMode(result, &cfg)
	}

	if result.OK {
		return result, nil
	}
	return result, fmt.Errorf("connection doctor found static configuration issues")
}

func (a *Application) resolveConnectionNameForSelection(ctx context.Context, name string, action string) (string, error) {
	if strings.TrimSpace(name) != "" {
		return name, nil
	}

	connections, err := a.store.ListConnections()
	if err != nil {
		return "", util.WrapLayer("config", "list configured connections", err)
	}
	if len(connections) == 0 {
		return "", util.WrapLayer("config", action, fmt.Errorf("no saved connections; run connection create"))
	}

	selected, err := a.promptForConnectionSelection(ctx, connections)
	if err != nil {
		return "", err
	}
	return selected, nil
}

func (a *Application) printDoctorResult(result *DoctorResult) {
	if result == nil {
		return
	}

	a.prompt.Printf("Connection doctor: %s\n", result.Connection)
	a.prompt.Println()
	for _, check := range result.Checks {
		label := "[OK]"
		if check.Status == "warn" {
			label = "[WARN]"
		}
		if check.Status == "fail" {
			label = "[FAIL]"
		}
		a.prompt.Printf("%s %s\n", label, check.Name)
		if strings.TrimSpace(check.Suggestion) != "" {
			a.prompt.Printf("       suggestion: %s\n", check.Suggestion)
		}
	}
}

func addDoctorCheck(result *DoctorResult, name string, status string, suggestion string) {
	result.Checks = append(result.Checks, DoctorCheck{
		Name:       name,
		Status:     status,
		Suggestion: suggestion,
	})
	if status == "fail" {
		result.OK = false
	}
}

func checkIdentifier(value string) string {
	if err := util.ValidateIdentifier(value); err != nil {
		return "fail"
	}
	return "ok"
}

func checkDriver(value string) string {
	if strings.TrimSpace(value) != "mysql" {
		return "fail"
	}
	return "ok"
}

func checkMode(value string) string {
	switch strings.TrimSpace(value) {
	case "direct", "ssh", "proxy", "proxy-ssh":
		return "ok"
	default:
		return "fail"
	}
}

func checkNonEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "fail"
	}
	return "ok"
}

func checkPort(port int) string {
	if port < 1 || port > 65535 {
		return "fail"
	}
	return "ok"
}

func checkDatabasePassword(result *DoctorResult, cfg *config.ConnectionConfig) {
	switch {
	case strings.TrimSpace(cfg.Password) != "":
		addDoctorCheck(result, "database password is inline in config", "warn", "prefer password_env or password_prompt")
	case strings.TrimSpace(cfg.PasswordEnv) != "":
		if os.Getenv(cfg.PasswordEnv) == "" {
			addDoctorCheck(result, "database password env "+cfg.PasswordEnv+" is set", "warn", "export "+cfg.PasswordEnv+" before connecting")
			return
		}
		addDoctorCheck(result, "database password env "+cfg.PasswordEnv+" is set", "ok", "")
	case cfg.PasswordPrompt:
		addDoctorCheck(result, "database password prompt enabled", "ok", "")
	default:
		addDoctorCheck(result, "database password source is configured", "warn", "set password_env or enable password_prompt")
	}
}

func checkTimeouts(result *DoctorResult, cfg *config.ConnectionConfig) {
	if cfg.Timeout == nil {
		return
	}
	if cfg.Timeout.ConnectSeconds <= 0 {
		addDoctorCheck(result, "connect timeout is valid", "fail", "")
	} else {
		addDoctorCheck(result, fmt.Sprintf("connect timeout %d seconds", cfg.Timeout.ConnectSeconds), "ok", "")
	}
	if cfg.Timeout.QuerySeconds <= 0 {
		addDoctorCheck(result, "query timeout is valid", "fail", "")
	} else {
		addDoctorCheck(result, fmt.Sprintf("query timeout %d seconds", cfg.Timeout.QuerySeconds), "ok", "")
	}
}

func checkDirectMode(result *DoctorResult, cfg *config.ConnectionConfig) {
	addDoctorCheck(result, "no SSH config required", "ok", "")
	if cfg.Proxy != nil && strings.TrimSpace(cfg.Proxy.URL) != "" {
		addDoctorCheck(result, "proxy config must be empty for direct mode", "fail", "remove proxy config or use mode proxy")
	}
}

func checkSSHProxyAbsent(result *DoctorResult, cfg *config.ConnectionConfig) {
	if cfg.Proxy != nil && strings.TrimSpace(cfg.Proxy.URL) != "" {
		addDoctorCheck(result, "proxy config must be empty for ssh mode", "fail", "remove proxy config or use mode proxy-ssh")
	}
}

func checkProxySSHAbsent(result *DoctorResult, cfg *config.ConnectionConfig) {
	if cfg.SSH != nil {
		addDoctorCheck(result, "ssh config must be empty for proxy mode", "fail", "remove ssh config or use mode proxy-ssh")
	}
}

func checkSSHMode(result *DoctorResult, cfg *config.ConnectionConfig) {
	if cfg.SSH == nil {
		addDoctorCheck(result, "ssh settings are configured", "fail", "set ssh.host, ssh.user, and an SSH auth method")
		return
	}

	addDoctorCheck(result, "ssh host "+cfg.SSH.Host, checkNonEmpty(cfg.SSH.Host), "")
	addDoctorCheck(result, fmt.Sprintf("ssh port %d", cfg.SSH.Port), checkPort(cfg.SSH.Port), "")
	addDoctorCheck(result, "ssh user "+cfg.SSH.User, checkNonEmpty(cfg.SSH.User), "")

	hasAuth := false
	if strings.TrimSpace(cfg.SSH.PrivateKey) != "" {
		hasAuth = true
		privateKeyPath, err := cfg.SSH.PrivateKeyPath()
		if err != nil {
			addDoctorCheck(result, "ssh private key path expands", "fail", "")
		} else if info, err := os.Stat(privateKeyPath); err != nil {
			addDoctorCheck(result, "ssh private key exists "+privateKeyPath, "fail", "")
		} else {
			addDoctorCheck(result, "ssh private key exists "+privateKeyPath, "ok", "")
			if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
				addDoctorCheck(result, "ssh private key permissions are strict", "warn", "restrict permissions with chmod 600 "+privateKeyPath)
			}
		}
	}

	switch {
	case strings.TrimSpace(cfg.SSH.Password) != "":
		hasAuth = true
		addDoctorCheck(result, "ssh password is inline in config", "warn", "prefer ssh password_env or a private key")
	case strings.TrimSpace(cfg.SSH.PasswordEnv) != "":
		hasAuth = true
		if os.Getenv(cfg.SSH.PasswordEnv) == "" {
			addDoctorCheck(result, "ssh password env "+cfg.SSH.PasswordEnv+" is set", "warn", "export "+cfg.SSH.PasswordEnv+" before connecting")
		} else {
			addDoctorCheck(result, "ssh password env "+cfg.SSH.PasswordEnv+" is set", "ok", "")
		}
	}

	if !hasAuth {
		addDoctorCheck(result, "ssh auth method is configured", "fail", "set ssh.private_key, ssh.password_env, or ssh.password")
	}

	for _, check := range knownHostsChecks(cfg.SSH.Host, cfg.SSH.Port) {
		addDoctorCheck(result, check.Name, check.Status, check.Suggestion)
	}
}

func checkProxyMode(result *DoctorResult, cfg *config.ConnectionConfig) {
	if cfg.Proxy == nil || strings.TrimSpace(cfg.Proxy.URL) == "" {
		addDoctorCheck(result, "proxy URL is configured", "fail", "set proxy.url for mode "+cfg.Mode)
		return
	}

	parsed, err := config.ParseProxyURL(cfg.Proxy.URL)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported proxy scheme") {
			addDoctorCheck(result, "proxy scheme is supported", "fail", "supported schemes: socks5")
		} else {
			addDoctorCheck(result, "proxy URL is valid", "fail", "")
		}
		return
	}

	addDoctorCheck(result, "proxy scheme "+parsed.Scheme, "ok", "")
	host, portText, splitErr := net.SplitHostPort(parsed.Address)
	if splitErr != nil || strings.TrimSpace(host) == "" {
		addDoctorCheck(result, "proxy host is valid", "fail", "")
		return
	}
	addDoctorCheck(result, "proxy host "+host, "ok", "")

	port, atoiErr := strconv.Atoi(portText)
	if atoiErr != nil || checkPort(port) != "ok" {
		addDoctorCheck(result, "proxy port is valid", "fail", "")
	} else {
		addDoctorCheck(result, fmt.Sprintf("proxy port %d", port), "ok", "")
	}

	if parsed.Password != "" {
		addDoctorCheck(result, "proxy URL contains inline password", "warn", "avoid saving inline proxy passwords in config")
	}
}

func knownHostsChecks(host string, port int) []DoctorCheck {
	checks := make([]DoctorCheck, 0, 1)
	path, err := util.ExpandHome("~/.ssh/known_hosts")
	if err != nil {
		checks = append(checks, DoctorCheck{
			Name:   "known_hosts path resolves",
			Status: "warn",
		})
		return checks
	}

	file, err := os.Open(path)
	if err != nil {
		checks = append(checks, DoctorCheck{
			Name:       "known_hosts file not found",
			Status:     "warn",
			Suggestion: fmt.Sprintf("create ~/.ssh/known_hosts or run ssh-keyscan %s >> ~/.ssh/known_hosts", host),
		})
		return checks
	}
	defer file.Close()

	hashed := false
	matched := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if strings.HasPrefix(fields[0], "|1|") {
			hashed = true
			continue
		}
		for _, entry := range strings.Split(fields[0], ",") {
			if matchesKnownHostEntry(entry, host, port) {
				matched = true
				break
			}
		}
		if matched {
			break
		}
	}

	if matched {
		checks = append(checks, DoctorCheck{
			Name:   "known_hosts entry exists for " + host,
			Status: "ok",
		})
		return checks
	}
	if hashed {
		checks = append(checks, DoctorCheck{
			Name:       "hashed known_hosts entries cannot be statically verified",
			Status:     "warn",
			Suggestion: "use connection test to verify SSH host key handling",
		})
		return checks
	}
	checks = append(checks, DoctorCheck{
		Name:       "known_hosts entry missing for " + host,
		Status:     "warn",
		Suggestion: fmt.Sprintf("run: ssh-keyscan %s >> ~/.ssh/known_hosts", host),
	})
	return checks
}

func matchesKnownHostEntry(entry string, host string, port int) bool {
	if entry == host {
		return port == 22
	}
	return entry == fmt.Sprintf("[%s]:%d", host, port)
}
