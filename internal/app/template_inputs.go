package app

import (
	"fmt"
	"strconv"
	"strings"

	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

func normalizeTemplateInputValue(input tpl.Input, value string) (string, error) {
	value = strings.TrimSpace(value)

	switch input.EffectiveType() {
	case "secret", "string":
		return value, nil
	case "select":
		options := input.SelectOptions()
		for _, option := range options {
			if value == option {
				return value, nil
			}
		}
		return "", util.WrapLayer("validation", "validate template input "+input.Name, fmt.Errorf("value must be one of %s", strings.Join(options, ", ")))
	case "confirm":
		switch strings.ToLower(value) {
		case "y", "yes", "true", "1":
			return "true", nil
		case "n", "no", "false", "0":
			return "false", nil
		default:
			return "", util.WrapLayer("validation", "validate template input "+input.Name, fmt.Errorf("value must be true/false"))
		}
	case "identifier":
		if err := util.ValidateIdentifier(value); err != nil {
			return "", util.WrapLayer("validation", "validate template input "+input.Name, err)
		}
		return value, nil
	case "int":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return "", util.WrapLayer("validation", "validate template input "+input.Name, err)
		}
		return strconv.Itoa(parsed), nil
	default:
		return value, nil
	}
}

func defaultTemplateInputValue(input tpl.Input) (string, bool, error) {
	switch input.EffectiveType() {
	case "confirm":
		if input.Default != nil {
			if input.DefaultBool() {
				return "true", true, nil
			}
			return "false", true, nil
		}
		return "", false, nil
	case "int":
		if value, ok := input.DefaultInt(); ok {
			return strconv.Itoa(value), true, nil
		}
		return "", false, nil
	default:
		if input.Default == nil {
			return "", false, nil
		}
		value := input.DefaultString()
		normalized, err := normalizeTemplateInputValue(input, value)
		if err != nil {
			return "", false, err
		}
		return normalized, true, nil
	}
}

func mergeTemplateInputs(template *tpl.Template, values map[string]string, requireAll bool) (map[string]string, error) {
	merged := make(map[string]string, len(values))
	for key, value := range values {
		merged[key] = value
	}

	for _, input := range template.Inputs {
		if current, exists := merged[input.Name]; exists {
			normalized, err := normalizeTemplateInputValue(input, current)
			if err != nil {
				return nil, err
			}
			merged[input.Name] = normalized
			continue
		}

		defaultValue, hasDefault, err := defaultTemplateInputValue(input)
		if err != nil {
			return nil, err
		}
		if hasDefault {
			merged[input.Name] = defaultValue
			continue
		}
		if requireAll && input.IsRequired() {
			return nil, util.WrapLayer("validation", "collect template inputs", fmt.Errorf("missing required template input %q", input.Name))
		}
	}

	return merged, nil
}
