package template

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Template struct {
	Version     int      `json:"version,omitempty"`
	Name        string   `json:"name"`
	Transaction bool     `json:"transaction,omitempty"`
	Match       Match    `json:"match"`
	Inputs      []Input  `json:"inputs,omitempty"`
	Actions     []Action `json:"actions"`

	Layer  string `json:"-"`
	Source string `json:"-"`
}

const CurrentTemplateSchemaVersion = 1

func (t *Template) ApplyDefaults() {
	if t.Version == 0 {
		t.Version = CurrentTemplateSchemaVersion
	}
}

func (t *Template) Validate() error {
	if t == nil {
		return fmt.Errorf("template is required")
	}
	t.ApplyDefaults()
	if t.Version != CurrentTemplateSchemaVersion {
		return fmt.Errorf("unsupported version %d", t.Version)
	}
	return nil
}

func (t *Template) UnmarshalJSON(data []byte) error {
	type rawTemplate Template
	var raw rawTemplate
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*t = Template(raw)
	t.ApplyDefaults()
	return nil
}

type Match struct {
	Command string `json:"command"`
	Driver  string `json:"driver,omitempty"`
}

type Input struct {
	Name       string   `json:"name"`
	Type       string   `json:"type,omitempty"`
	Prompt     string   `json:"prompt"`
	Secret     bool     `json:"secret,omitempty"`
	Default    any      `json:"default,omitempty"`
	Options    []string `json:"options,omitempty"`
	Choices    []string `json:"choices,omitempty"`
	Identifier bool     `json:"identifier,omitempty"`
}

func (i Input) EffectiveType() string {
	switch strings.TrimSpace(i.Type) {
	case "string", "secret", "select", "confirm", "identifier", "int":
		return strings.TrimSpace(i.Type)
	}
	if i.Secret {
		return "secret"
	}
	if i.Identifier {
		return "identifier"
	}
	if len(i.Options) > 0 || len(i.Choices) > 0 {
		return "select"
	}
	return "string"
}

func (i Input) SelectOptions() []string {
	if len(i.Options) > 0 {
		return append([]string(nil), i.Options...)
	}
	return append([]string(nil), i.Choices...)
}

func (i Input) DefaultString() string {
	switch value := i.Default.(type) {
	case string:
		return value
	case float64:
		return strconv.Itoa(int(value))
	case int:
		return strconv.Itoa(value)
	case bool:
		if value {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func (i Input) DefaultBool() bool {
	switch value := i.Default.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true") || strings.EqualFold(value, "yes") || value == "1"
	case float64:
		return value != 0
	default:
		return false
	}
}

func (i Input) DefaultInt() (int, bool) {
	switch value := i.Default.(type) {
	case int:
		return value, true
	case float64:
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

type Action struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	SQL         string `json:"sql,omitempty"`
}

type ExecutionPlan struct {
	TemplateName string
	Layer        string
	Source       string
	Transaction  bool
	Actions      []RenderedAction
}

type RenderedAction struct {
	Description string
	SQL         string
}
