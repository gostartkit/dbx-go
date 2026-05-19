package template

import (
	"bytes"
	"fmt"
	"regexp"
	ttpl "text/template"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

var mustachePattern = regexp.MustCompile(`{{\s*([a-zA-Z0-9_.]+)\s*}}`)

func BuildPlan(tpl *Template, cfg *config.ConnectionConfig, values map[string]string) (*ExecutionPlan, error) {
	if tpl == nil {
		return nil, fmt.Errorf("template is required")
	}

	rawData := renderData(cfg, values, false)
	sqlData := renderData(cfg, values, true)

	plan := &ExecutionPlan{
		TemplateName: tpl.Name,
		Layer:        tpl.Layer,
		Source:       tpl.Source,
		Transaction:  tpl.Transaction,
		Actions:      make([]RenderedAction, 0, len(tpl.Actions)),
	}

	for _, action := range tpl.Actions {
		if action.Type != "sql" {
			return nil, fmt.Errorf("unsupported action type %q", action.Type)
		}

		description, err := renderText(action.Description, rawData)
		if err != nil {
			return nil, fmt.Errorf("render action description %q: %w", action.Description, err)
		}
		sql, err := renderText(action.SQL, sqlData)
		if err != nil {
			return nil, fmt.Errorf("render SQL for action %q: %w", action.Description, err)
		}

		plan.Actions = append(plan.Actions, RenderedAction{
			Description: description,
			SQL:         sql,
		})
	}

	return plan, nil
}

func renderText(input string, data map[string]any) (string, error) {
	normalized := mustachePattern.ReplaceAllString(input, "{{.$1}}")
	tpl, err := ttpl.New("dbx").Option("missingkey=error").Parse(normalized)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderData(cfg *config.ConnectionConfig, values map[string]string, escapeStrings bool) map[string]any {
	connection := map[string]any{
		"name":   cfg.Name,
		"driver": cfg.Driver,
		"mode":   cfg.Mode,
		"host":   cfg.Host,
		"port":   cfg.Port,
		"user":   cfg.User,
	}

	if escapeStrings {
		connection = escapeMap(connection)
	}

	data := map[string]any{
		"connection": connection,
	}

	for key, value := range values {
		if escapeStrings {
			data[key] = util.EscapeMySQLString(value)
			continue
		}
		data[key] = value
	}

	return data
}

func escapeMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case string:
			out[key] = util.EscapeMySQLString(typed)
		default:
			out[key] = value
		}
	}
	return out
}
