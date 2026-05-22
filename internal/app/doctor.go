package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleConnectionDoctor(ctx context.Context, name string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "doctor"}, func(meta *auditMetadata) error {
		selected, err := a.resolveConnectionNameForSelection(ctx, name, "doctor")
		if err != nil {
			return err
		}
		meta.Connection = selected

		result, doctorErr := a.doctorConnection(selected)
		if result != nil {
			meta.Connection = result.Connection
		}
		if cfg, err := a.store.LoadConnection(selected); err == nil {
			meta.Mode = cfg.Mode
		}
		a.printDoctorResult(result)
		if doctorErr != nil {
			failed := false
			meta.Success = &failed
			return nil
		}
		succeeded := true
		meta.Success = &succeeded
		return nil
	})
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
	appendDoctorValidationChecks(result, cfg.ValidateDetailed())
	checkSSHFilesystem(result, &cfg)

	if result.OK {
		return result, nil
	}
	return result, fmt.Errorf("doctor found static configuration issues")
}

func (a *Application) resolveConnectionNameForSelection(ctx context.Context, name string, action string) (string, error) {
	if strings.TrimSpace(name) != "" {
		return name, nil
	}

	records, err := a.store.ListConnectionRecords()
	if err != nil {
		return "", util.WrapLayer("config", "list configured connections", err)
	}
	if len(records) == 0 {
		return "", util.WrapLayer("config", action, fmt.Errorf("no saved connections; run create connection <name>"))
	}

	selected, err := a.promptForConnectionRecordSelection(ctx, records)
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

func appendDoctorValidationChecks(result *DoctorResult, report *config.ValidationReport) {
	if result == nil || report == nil {
		return
	}
	for _, check := range report.Checks {
		addDoctorCheck(result, check.Name, string(check.Status), check.Suggestion)
	}
}

func checkSSHFilesystem(result *DoctorResult, cfg *config.ConnectionConfig) {
	if result == nil || cfg == nil || cfg.SSH == nil {
		return
	}
	if strings.TrimSpace(cfg.SSH.PrivateKey) != "" {
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
	for _, check := range knownHostsChecks(cfg.SSH.Host, cfg.SSH.Port) {
		addDoctorCheck(result, check.Name, check.Status, check.Suggestion)
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
