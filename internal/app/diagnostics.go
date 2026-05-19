package app

import (
	"context"
	"errors"
	"fmt"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) diagnoseConnection(ctx context.Context, cfg *config.ConnectionConfig) (*DiagnosticResult, error) {
	result := &DiagnosticResult{
		OK:         false,
		Connection: cfg.Name,
		Steps:      make([]DiagnosticStep, 0, len(expectedDiagnosticLayers(cfg))+1),
	}

	runtimeCfg, err := a.prepareConnectionForOpen(ctx, cfg)
	if err != nil {
		result.Steps = append(result.Steps, DiagnosticStep{
			Name:   "config",
			Status: "fail",
			Error:  diagnosticRootError(err),
		})
		return result, err
	}

	if err := runtimeCfg.Validate(); err != nil {
		result.Steps = append(result.Steps, DiagnosticStep{
			Name:   "config",
			Status: "fail",
			Error:  diagnosticRootError(err),
		})
		return result, util.WrapLayer("config", "validate connection config", err)
	}

	db, err := a.connector.Open(ctx, runtimeCfg)
	if err != nil {
		failedLayer := diagnosticFailureLayer(runtimeCfg, err)
		if failedLayer == "config" {
			result.Steps = append(result.Steps, DiagnosticStep{
				Name:   "config",
				Status: "fail",
				Error:  diagnosticRootError(err),
			})
			return result, err
		}

		result.Steps = append(result.Steps, DiagnosticStep{Name: "config", Status: "ok"})
		for _, layer := range expectedDiagnosticLayers(runtimeCfg) {
			if layer == failedLayer {
				result.Steps = append(result.Steps, DiagnosticStep{
					Name:   layer,
					Status: "fail",
					Error:  diagnosticRootError(err),
				})
				return result, err
			}
			result.Steps = append(result.Steps, DiagnosticStep{Name: layer, Status: "ok"})
		}

		result.Steps = append(result.Steps, DiagnosticStep{
			Name:   failedLayer,
			Status: "fail",
			Error:  diagnosticRootError(err),
		})
		return result, err
	}
	if db != nil {
		defer db.Close()
	}

	result.Steps = append(result.Steps, DiagnosticStep{Name: "config", Status: "ok"})
	for _, layer := range expectedDiagnosticLayers(runtimeCfg) {
		result.Steps = append(result.Steps, DiagnosticStep{Name: layer, Status: "ok"})
	}
	result.OK = true
	return result, nil
}

func (a *Application) handleConnectionTest(ctx context.Context, name string) error {
	cfg, err := a.resolveConnectionForTest(ctx, name)
	if err != nil {
		return err
	}

	result, diagErr := a.diagnoseConnection(ctx, cfg)
	a.printDiagnosticResult(result)
	if diagErr != nil {
		a.prompt.Println(diagErr.Error())
		return nil
	}

	a.prompt.Println("Connection successful.")
	return nil
}

func (a *Application) resolveConnectionForTest(ctx context.Context, name string) (*config.ConnectionConfig, error) {
	if name != "" {
		cfg, err := a.store.LoadConnection(name)
		if err != nil {
			return nil, util.WrapLayer("config", "load connection "+name, err)
		}
		return cfg, nil
	}

	connections, err := a.store.ListConnections()
	if err != nil {
		return nil, util.WrapLayer("config", "list configured connections", err)
	}
	if len(connections) == 0 {
		return nil, util.WrapLayer("config", "connection test", fmt.Errorf("no saved connections; run connection create"))
	}

	selected, err := a.promptForConnectionSelection(ctx, connections)
	if err != nil {
		return nil, err
	}
	cfg, err := a.store.LoadConnection(selected)
	if err != nil {
		return nil, util.WrapLayer("config", "load connection "+selected, err)
	}
	return cfg, nil
}

func (a *Application) printDiagnosticResult(result *DiagnosticResult) {
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

func diagnosticFailureLayer(cfg *config.ConnectionConfig, err error) string {
	expected := expectedDiagnosticLayers(cfg)
	found := ""

	for current := err; current != nil; current = errors.Unwrap(current) {
		layerErr, ok := current.(*util.LayerError)
		if !ok {
			continue
		}
		switch layerErr.Layer {
		case "validation", "config":
			found = "config"
		case "proxy", "ssh", "mysql":
			found = layerErr.Layer
		}
	}

	if found != "" {
		return found
	}
	return expected[len(expected)-1]
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
