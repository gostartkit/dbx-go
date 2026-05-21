package app

import (
	"fmt"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
)

const operationProviderTemplate = "template"

type executableSpec struct {
	operation string
	provider  string
	scope     string
	category  string
	source    string
	template  *tpl.Template
}

func newTemplateExecutableSpec(template *tpl.Template) *executableSpec {
	if template == nil {
		return nil
	}
	return &executableSpec{
		operation: template.Name,
		provider:  operationProviderTemplate,
		scope:     template.Layer,
		category:  template.EffectiveCategory(),
		source:    template.Source,
		template:  template,
	}
}

func (s *executableSpec) Operation() string {
	if s == nil {
		return ""
	}
	return s.operation
}

func (s *executableSpec) Provider() string {
	if s == nil {
		return ""
	}
	return s.provider
}

func (s *executableSpec) Scope() string {
	if s == nil {
		return ""
	}
	return s.scope
}

func (s *executableSpec) Category() string {
	if s == nil {
		return ""
	}
	return s.category
}

func (s *executableSpec) Source() string {
	if s == nil {
		return ""
	}
	return s.source
}

func (s *executableSpec) Transaction() bool {
	if s == nil || s.template == nil {
		return false
	}
	return s.template.Transaction
}

func (s *executableSpec) Inputs() []tpl.Input {
	if s == nil || s.template == nil {
		return nil
	}
	return s.template.Inputs
}

func (s *executableSpec) Template() *tpl.Template {
	if s == nil {
		return nil
	}
	return s.template
}

func (s *executableSpec) Validate() error {
	if s == nil || s.template == nil {
		return fmt.Errorf("executable spec is required")
	}
	if err := s.template.Validate(); err != nil {
		return err
	}
	spec, ok := commandSpecByPath(s.template.Match.Command)
	if !ok || spec.Category != "command" {
		return fmt.Errorf("unsupported match command %q", s.template.Match.Command)
	}
	return nil
}

func (s *executableSpec) BuildPlans(cfg *config.ConnectionConfig, values map[string]string) (*tpl.ExecutionPlan, *tpl.ExecutionPlan, error) {
	if s == nil || s.template == nil {
		return nil, nil, fmt.Errorf("executable spec is required")
	}
	return buildPlans(s.template, cfg, values)
}

func (a *Application) resolveExecutableSpec(cfg *config.ConnectionConfig, name string) (*executableSpec, error) {
	selectedTemplate, err := a.templates.ResolveNamedAny(cfg, name)
	if err != nil {
		return nil, err
	}
	spec := newTemplateExecutableSpec(selectedTemplate)
	if spec == nil {
		return nil, fmt.Errorf("executable spec %q not found", name)
	}
	return spec, nil
}
