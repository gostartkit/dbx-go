package app

import (
	"context"
	"fmt"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
	"pkg.gostartkit.com/dbx/internal/util"
)

type diagnosticOptions struct {
	Verbose    bool
	ConfigPath string
}

func (a *Application) diagnoseConnection(ctx context.Context, cfg *config.ConnectionConfig, options diagnosticOptions) (*DiagnosticResult, error) {
	result := &DiagnosticResult{
		OK:         false,
		Connection: cfg.Name,
		Steps:      make([]DiagnosticStep, 0, len(expectedDiagnosticLayers(cfg))+1),
	}

	runtimeCfg, err := a.prepareConnectionForOpen(ctx, cfg)
	if err != nil {
		result.Steps = append(result.Steps, maybeWithDetails(options.Verbose, DiagnosticStep{
			Name:   "config",
			Status: "fail",
			Error:  diagnosticRootError(err),
		}, configStepDetails(cfg, options.ConfigPath)))
		return result, err
	}

	if err := runtimeCfg.Validate(); err != nil {
		result.Steps = append(result.Steps, maybeWithDetails(options.Verbose, DiagnosticStep{
			Name:   "config",
			Status: "fail",
			Error:  diagnosticRootError(err),
		}, configStepDetails(runtimeCfg, options.ConfigPath)))
		return result, util.WrapLayer("config", "validate connection config", err)
	}

	result.Steps = append(result.Steps, maybeWithDetails(options.Verbose, DiagnosticStep{
		Name:   "config",
		Status: "ok",
	}, configStepDetails(runtimeCfg, options.ConfigPath)))

	trace, err := a.connector.Diagnose(ctx, runtimeCfg)
	if err != nil {
		if trace != nil {
			for _, step := range trace.Steps {
				result.Steps = append(result.Steps, sanitizeDiagnosticStep(step, options.Verbose))
			}
		}
		return result, err
	}
	if trace != nil {
		for _, step := range trace.Steps {
			result.Steps = append(result.Steps, sanitizeDiagnosticStep(step, options.Verbose))
		}
	}
	result.OK = true
	return result, nil
}

func (a *Application) handleConnectionTest(ctx context.Context, name string, verbose bool) error {
	return a.auditCommand(ctx, auditMetadata{Command: "connection test"}, func(meta *auditMetadata) error {
		cfg, configPath, err := a.resolveConnectionForTest(ctx, name)
		if err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode

		result, diagErr := a.diagnoseConnection(ctx, cfg, diagnosticOptions{
			Verbose:    verbose,
			ConfigPath: configPath,
		})
		a.printDiagnosticResult(result, verbose)
		if diagErr != nil {
			failed := false
			meta.Success = &failed
			a.prompt.Println(diagErr.Error())
			return nil
		}

		succeeded := true
		meta.Success = &succeeded
		a.prompt.Println("Connection successful.")
		return nil
	})
}

func (a *Application) resolveConnectionForTest(ctx context.Context, name string) (*config.ConnectionConfig, string, error) {
	if name != "" {
		cfg, err := a.store.LoadConnection(name)
		if err != nil {
			return nil, "", util.WrapLayer("config", "load connection "+name, err)
		}
		return cfg, a.store.ConnectionConfigPath(name), nil
	}

	connections, err := a.store.ListConnections()
	if err != nil {
		return nil, "", util.WrapLayer("config", "list configured connections", err)
	}
	if len(connections) == 0 {
		return nil, "", util.WrapLayer("config", "connection test", fmt.Errorf("no saved connections; run connection create"))
	}

	selected, err := a.promptForConnectionSelection(ctx, connections)
	if err != nil {
		return nil, "", err
	}
	cfg, err := a.store.LoadConnection(selected)
	if err != nil {
		return nil, "", util.WrapLayer("config", "load connection "+selected, err)
	}
	return cfg, a.store.ConnectionConfigPath(selected), nil
}

func (a *Application) printDiagnosticResult(result *DiagnosticResult, verbose bool) {
	if result == nil {
		return
	}
	for _, step := range result.Steps {
		switch step.Status {
		case "ok":
			a.prompt.Printf("[OK] %s\n", step.Name)
		default:
			a.prompt.Printf("[FAIL] %s\n", step.Name)
		}
		if verbose {
			for _, line := range diagnosticDetailLines(step) {
				a.prompt.Printf("     %s\n", line)
			}
		}
	}
}

func expectedDiagnosticLayers(cfg *config.ConnectionConfig) []string {
	switch cfg.Mode {
	case "proxy-ssh":
		return []string{"proxy", "ssh", "mysql"}
	case "proxy":
		return []string{"proxy", "mysql"}
	case "ssh":
		return []string{"ssh", "mysql"}
	default:
		return []string{"mysql"}
	}
}

func diagnosticRootError(err error) string {
	current := err
	for {
		layerErr, ok := current.(*util.LayerError)
		if !ok || layerErr.Err == nil {
			break
		}
		current = layerErr.Err
	}
	if current == nil {
		return ""
	}
	return current.Error()
}

func sanitizeDiagnosticStep(step driver.DiagnosticStep, verbose bool) DiagnosticStep {
	result := DiagnosticStep{
		Name:   step.Name,
		Status: step.Status,
		Error:  step.Error,
	}
	if verbose && len(step.Details) > 0 {
		result.Details = step.Details
	}
	return result
}

func maybeWithDetails(verbose bool, step DiagnosticStep, details map[string]any) DiagnosticStep {
	if verbose && len(details) > 0 {
		step.Details = details
	}
	return step
}

func configStepDetails(cfg *config.ConnectionConfig, configPath string) map[string]any {
	if cfg == nil {
		return nil
	}

	details := map[string]any{
		"driver": cfg.Driver,
		"mode":   cfg.Mode,
	}
	if configPath != "" {
		details["config_path"] = configPath
	}
	return details
}

func diagnosticDetailLines(step DiagnosticStep) []string {
	lines := make([]string, 0, 4)

	switch step.Name {
	case "config":
		lines = appendDiagnosticDetail(lines, step.Details, "mode", "%v")
		lines = appendDiagnosticDetail(lines, step.Details, "driver", "%v")
		lines = appendDiagnosticDetail(lines, step.Details, "config_path", "%v")
	case "proxy":
		lines = appendDiagnosticDetail(lines, step.Details, "url", "%v")
		lines = appendDiagnosticDetail(lines, step.Details, "target", "%v")
		lines = appendDurationDetail(lines, step.Details, "duration")
	case "ssh":
		lines = appendDiagnosticDetail(lines, step.Details, "host", "%v")
		lines = appendDiagnosticDetail(lines, step.Details, "user", "%v")
		lines = appendDurationDetail(lines, step.Details, "duration")
	case "mysql":
		lines = appendDiagnosticDetail(lines, step.Details, "target", "%v")
		lines = appendDurationDetail(lines, step.Details, "ping")
	}

	if step.Status != "ok" && step.Error != "" {
		lines = append(lines, "error: "+step.Error)
	}
	return lines
}

func appendDiagnosticDetail(lines []string, details map[string]any, key string, format string) []string {
	if len(details) == 0 {
		return lines
	}
	value, ok := details[key]
	if !ok {
		return lines
	}
	return append(lines, fmt.Sprintf(key+": "+format, value))
}

func appendDurationDetail(lines []string, details map[string]any, label string) []string {
	if len(details) == 0 {
		return lines
	}
	value, ok := details["duration_ms"]
	if !ok {
		return lines
	}
	ms, ok := value.(int64)
	if !ok {
		return lines
	}
	return append(lines, fmt.Sprintf("%s: %dms", label, ms))
}

func parseConnectionTestArgs(args []string) (name string, verbose bool, ok bool) {
	switch len(args) {
	case 0:
		return "", false, true
	case 1:
		if args[0] == "verbose" {
			return "", true, true
		}
		return args[0], false, true
	case 2:
		if args[1] != "verbose" {
			return "", false, false
		}
		return args[0], true, true
	default:
		return "", false, false
	}
}
