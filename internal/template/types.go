package template

type Template struct {
	Name    string   `json:"name"`
	Match   Match    `json:"match"`
	Inputs  []Input  `json:"inputs,omitempty"`
	Actions []Action `json:"actions"`

	Layer  string `json:"-"`
	Source string `json:"-"`
}

type Match struct {
	Command string `json:"command"`
	Driver  string `json:"driver,omitempty"`
}

type Input struct {
	Name       string   `json:"name"`
	Prompt     string   `json:"prompt"`
	Secret     bool     `json:"secret,omitempty"`
	Default    string   `json:"default,omitempty"`
	Choices    []string `json:"choices,omitempty"`
	Identifier bool     `json:"identifier,omitempty"`
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
	Actions      []RenderedAction
}

type RenderedAction struct {
	Description string
	SQL         string
}
