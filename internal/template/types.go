package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Template struct {
	Version     int      `json:"version,omitempty"`
	Name        string   `json:"name"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Description string   `json:"description,omitempty"`
	Transaction bool     `json:"transaction,omitempty"`
	Match       Match    `json:"match"`
	Inputs      []Input  `json:"inputs,omitempty"`
	Actions     []Action `json:"actions"`

	Layer  string `json:"-"`
	Source string `json:"-"`
}

const CurrentTemplateSchemaVersion = 1

const DefaultTemplateCategory = "custom"

var templateInputNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func (t *Template) ApplyDefaults() {
	if t.Version == 0 {
		t.Version = CurrentTemplateSchemaVersion
	}
	t.Category = normalizeTemplateCategory(t.Category)
	t.Tags = normalizeTemplateTags(t.Tags)
}

func (t *Template) Validate() error {
	if t == nil {
		return fmt.Errorf("template is required")
	}
	t.ApplyDefaults()
	if t.Version != CurrentTemplateSchemaVersion {
		return fmt.Errorf("unsupported version %d", t.Version)
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("template name is required")
	}
	if strings.TrimSpace(t.Match.Command) == "" {
		return fmt.Errorf("match command is required")
	}
	if len(t.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	seenInputs := make(map[string]struct{}, len(t.Inputs))
	for _, input := range t.Inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			return fmt.Errorf("input name is required")
		}
		if !templateInputNamePattern.MatchString(name) {
			return fmt.Errorf("invalid input name %q", input.Name)
		}
		if _, exists := seenInputs[name]; exists {
			return fmt.Errorf("duplicate input name %q", input.Name)
		}
		seenInputs[name] = struct{}{}

		inputType := input.EffectiveType()
		switch inputType {
		case "string", "secret", "select", "confirm", "identifier", "int":
		default:
			return fmt.Errorf("unsupported input type %q", input.Type)
		}
		if input.Secret && strings.TrimSpace(input.Type) != "" && strings.TrimSpace(input.Type) != "secret" {
			return fmt.Errorf("input %q cannot set secret=true with type %q", input.Name, input.Type)
		}
		if inputType == "select" && len(input.SelectOptions()) == 0 {
			return fmt.Errorf("select input %q must define options", input.Name)
		}
		if inputType == "int" && input.Default != nil {
			if _, ok := input.DefaultInt(); !ok {
				return fmt.Errorf("int input %q has invalid default", input.Name)
			}
		}
	}

	for _, action := range t.Actions {
		if strings.TrimSpace(action.Type) != "sql" {
			return fmt.Errorf("unsupported action type %q", action.Type)
		}
		if strings.TrimSpace(action.SQL) == "" {
			return fmt.Errorf("sql action %q must not be empty", action.Description)
		}
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
	Name        string   `json:"name"`
	Type        string   `json:"type,omitempty"`
	Prompt      string   `json:"prompt"`
	Description string   `json:"description,omitempty"`
	Required    *bool    `json:"required,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
	Default     any      `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
	Choices     []string `json:"choices,omitempty"`
	Identifier  bool     `json:"identifier,omitempty"`
}

func (i Input) IsRequired() bool {
	if i.Required != nil {
		return *i.Required
	}
	return i.Default == nil
}

func (i Input) PromptText() string {
	if strings.TrimSpace(i.Prompt) != "" {
		return i.Prompt
	}
	if strings.TrimSpace(i.Description) != "" {
		return i.Description
	}
	return i.Name
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
	OperationName string
	Layer         string
	Category      string
	Source        string
	Transaction   bool
	Actions       []RenderedAction
}

type RenderedAction struct {
	Description string
	SQL         string
}

func (t Template) EffectiveCategory() string {
	if strings.TrimSpace(t.Category) == "" {
		return DefaultTemplateCategory
	}
	return strings.TrimSpace(t.Category)
}

func (t Template) EffectiveTags() []string {
	return append([]string(nil), t.Tags...)
}

func normalizeTemplateCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		return DefaultTemplateCategory
	}
	return category
}

func normalizeTemplateTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	slices.Sort(normalized)
	return normalized
}
