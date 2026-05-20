package app

type CompletionContext struct {
	Connection   string
	Database     string
	DryRun       bool
	Connections  []CompletionConnection
	Databases    []string
	Tables       []string
	Templates    []string
	TemplateTags []string
	Users        []string
}

type CompletionConnection struct {
	Name   string
	Driver string
	Mode   string
}

type Suggestion struct {
	Value       string
	Description string
	Category    string
}
