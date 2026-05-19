package driver

type DiagnosticStep struct {
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	Error   string         `json:"error,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type DiagnosticTrace struct {
	Steps []DiagnosticStep
}
