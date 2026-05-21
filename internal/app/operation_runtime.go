package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
)

type OperationKind string

const (
	OperationBuiltin  OperationKind = "builtin"
	OperationTemplate OperationKind = "template"
	OperationDSL      OperationKind = "dsl"
	OperationFile     OperationKind = "file"
)

type OperationDefinition struct {
	Name        string
	Kind        OperationKind
	Description string
	Source      string
	Metadata    map[string]string

	template *tpl.Template
}

type OperationArgs struct {
	Config *config.ConnectionConfig
	Values map[string]string
	DryRun bool
}

type OperationPlan struct {
	Definition  *OperationDefinition
	Config      *config.ConnectionConfig
	Values      map[string]string
	Execution   *tpl.ExecutionPlan
	Preview     *tpl.ExecutionPlan
	DryRun      bool
	Result      *PlanExecutionResult
	Transaction bool
}

type OperationImplementation interface {
	Kind() OperationKind
	Resolve(ctx context.Context, name string, cfg *config.ConnectionConfig) (*OperationDefinition, error)
	Plan(ctx context.Context, def *OperationDefinition, args OperationArgs) (*OperationPlan, error)
	Execute(ctx context.Context, plan *OperationPlan) error
}

type listableOperationImplementation interface {
	OperationImplementation
	List(ctx context.Context, cfg *config.ConnectionConfig) ([]OperationDefinition, error)
}

type operationRuntime struct {
	implementations []OperationImplementation
}

var errOperationNotFound = errors.New("operation not found")

type builtinOperationImplementation struct {
	app *Application
}

type templateOperationImplementation struct {
	app *Application
}

type dslOperationImplementation struct{}
type fileOperationImplementation struct{}

func (a *Application) operationRuntime() *operationRuntime {
	return &operationRuntime{
		implementations: []OperationImplementation{
			builtinOperationImplementation{app: a},
			templateOperationImplementation{app: a},
			dslOperationImplementation{},
			fileOperationImplementation{},
		},
	}
}

func (r *operationRuntime) Resolve(ctx context.Context, cfg *config.ConnectionConfig, name string) (*OperationDefinition, OperationImplementation, error) {
	for _, implementation := range r.implementations {
		definition, err := implementation.Resolve(ctx, name, cfg)
		if err == nil && definition != nil {
			return definition, implementation, nil
		}
		if err != nil && !errors.Is(err, errOperationNotFound) {
			return nil, nil, err
		}
	}
	return nil, nil, fmt.Errorf("operation %q not found", name)
}

func (r *operationRuntime) List(ctx context.Context, cfg *config.ConnectionConfig) ([]operationNameEntry, error) {
	seen := make(map[string]struct{})
	results := make([]operationNameEntry, 0)
	for _, implementation := range r.implementations {
		listable, ok := implementation.(listableOperationImplementation)
		if !ok {
			continue
		}
		definitions, err := listable.List(ctx, cfg)
		if err != nil {
			return nil, err
		}
		for _, definition := range definitions {
			if _, ok := seen[definition.Name]; ok {
				continue
			}
			seen[definition.Name] = struct{}{}
			results = append(results, operationNameEntry{
				Name:        definition.Name,
				Description: definition.Description,
			})
		}
	}
	slices.SortFunc(results, func(left operationNameEntry, right operationNameEntry) int {
		return strings.Compare(left.Name, right.Name)
	})
	return results, nil
}

func (builtinOperationImplementation) Kind() OperationKind  { return OperationBuiltin }
func (templateOperationImplementation) Kind() OperationKind { return OperationTemplate }
func (dslOperationImplementation) Kind() OperationKind      { return OperationDSL }
func (fileOperationImplementation) Kind() OperationKind     { return OperationFile }

func (i builtinOperationImplementation) Resolve(_ context.Context, name string, _ *config.ConnectionConfig) (*OperationDefinition, error) {
	for _, candidate := range tpl.Builtins() {
		if candidate.Name != name {
			continue
		}
		tplCopy := candidate
		return operationDefinitionFromTemplate(&tplCopy, OperationBuiltin), nil
	}
	return nil, fmt.Errorf("%w: builtin %q", errOperationNotFound, name)
}

func (i builtinOperationImplementation) List(_ context.Context, _ *config.ConnectionConfig) ([]OperationDefinition, error) {
	results := make([]OperationDefinition, 0, len(tpl.Builtins()))
	for _, candidate := range tpl.Builtins() {
		tplCopy := candidate
		results = append(results, *operationDefinitionFromTemplate(&tplCopy, OperationBuiltin))
	}
	return results, nil
}

func (i templateOperationImplementation) Resolve(_ context.Context, name string, cfg *config.ConnectionConfig) (*OperationDefinition, error) {
	selectedTemplate, err := i.app.templates.ResolveNamedAny(cfg, name)
	if err != nil {
		return nil, err
	}
	if selectedTemplate.Layer == "builtin" {
		return nil, fmt.Errorf("%w: template %q", errOperationNotFound, name)
	}
	return operationDefinitionFromTemplate(selectedTemplate, OperationTemplate), nil
}

func (i templateOperationImplementation) List(_ context.Context, cfg *config.ConnectionConfig) ([]OperationDefinition, error) {
	templates, err := i.app.templates.ListResolved(cfg)
	if err != nil {
		return nil, err
	}
	results := make([]OperationDefinition, 0, len(templates))
	for _, candidate := range templates {
		if candidate.Layer == "builtin" {
			continue
		}
		tplCopy := candidate
		results = append(results, *operationDefinitionFromTemplate(&tplCopy, OperationTemplate))
	}
	return results, nil
}

func (i builtinOperationImplementation) Plan(_ context.Context, def *OperationDefinition, args OperationArgs) (*OperationPlan, error) {
	return buildOperationPlan(def, args)
}

func (i templateOperationImplementation) Plan(_ context.Context, def *OperationDefinition, args OperationArgs) (*OperationPlan, error) {
	return buildOperationPlan(def, args)
}

func (i builtinOperationImplementation) Execute(ctx context.Context, plan *OperationPlan) error {
	return executeOperationPlan(ctx, i.app, plan)
}

func (i templateOperationImplementation) Execute(ctx context.Context, plan *OperationPlan) error {
	return executeOperationPlan(ctx, i.app, plan)
}

func (dslOperationImplementation) Resolve(context.Context, string, *config.ConnectionConfig) (*OperationDefinition, error) {
	return nil, fmt.Errorf("%w: dsl", errOperationNotFound)
}

func (fileOperationImplementation) Resolve(context.Context, string, *config.ConnectionConfig) (*OperationDefinition, error) {
	return nil, fmt.Errorf("%w: file", errOperationNotFound)
}

func (dslOperationImplementation) Plan(context.Context, *OperationDefinition, OperationArgs) (*OperationPlan, error) {
	return nil, fmt.Errorf("dsl operations are not implemented yet")
}

func (fileOperationImplementation) Plan(context.Context, *OperationDefinition, OperationArgs) (*OperationPlan, error) {
	return nil, fmt.Errorf("file operations are not implemented yet")
}

func (dslOperationImplementation) Execute(context.Context, *OperationPlan) error {
	return fmt.Errorf("dsl operations are not implemented yet")
}

func (fileOperationImplementation) Execute(context.Context, *OperationPlan) error {
	return fmt.Errorf("file operations are not implemented yet")
}

func operationDefinitionFromTemplate(template *tpl.Template, kind OperationKind) *OperationDefinition {
	if template == nil {
		return nil
	}
	return &OperationDefinition{
		Name:        template.Name,
		Kind:        kind,
		Description: template.Description,
		Source:      template.Source,
		Metadata: map[string]string{
			"scope":    template.Layer,
			"category": template.EffectiveCategory(),
			"command":  template.Match.Command,
		},
		template: template,
	}
}

func buildOperationPlan(def *OperationDefinition, args OperationArgs) (*OperationPlan, error) {
	if def == nil || def.template == nil {
		return nil, fmt.Errorf("operation definition is required")
	}
	plan, preview, err := buildPlans(def.template, args.Config, args.Values)
	if err != nil {
		return nil, err
	}
	return &OperationPlan{
		Definition:  def,
		Config:      args.Config,
		Values:      cloneTemplateValues(args.Values),
		Execution:   plan,
		Preview:     preview,
		DryRun:      args.DryRun,
		Transaction: plan.Transaction,
	}, nil
}

func executeOperationPlan(ctx context.Context, app *Application, plan *OperationPlan) error {
	if app == nil || plan == nil {
		return fmt.Errorf("operation plan is required")
	}
	if plan.DryRun {
		result, err := app.runPlan(ctx, plan.Execution, noopTransactionStarter{}, true)
		plan.Result = result
		return err
	}
	db, err := app.openConnection(ctx, plan.Config)
	if err != nil {
		return err
	}
	defer db.Close()
	result, err := app.executePlan(ctx, plan.Execution, sqlRunner{db: db})
	plan.Result = result
	return err
}

func cloneTemplateValues(values map[string]string) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func (a *Application) listOperationNames(ctx context.Context) ([]operationNameEntry, error) {
	cfg, err := a.templateScopeConfig("")
	if err != nil {
		return nil, err
	}
	return a.operationRuntime().List(ctx, cfg)
}

func operationProviderName(kind OperationKind) string {
	switch kind {
	case OperationBuiltin:
		return "builtin"
	case OperationTemplate:
		return "template"
	case OperationDSL:
		return "dsl"
	case OperationFile:
		return "file"
	default:
		return string(kind)
	}
}

func operationScope(def *OperationDefinition) string {
	if def == nil {
		return ""
	}
	return def.Metadata["scope"]
}

func operationCategory(def *OperationDefinition) string {
	if def == nil {
		return ""
	}
	return def.Metadata["category"]
}

func operationCommand(def *OperationDefinition) string {
	if def == nil {
		return ""
	}
	return def.Metadata["command"]
}

func (plan *OperationPlan) TransactionEnabled() bool {
	if plan == nil {
		return false
	}
	return plan.Transaction
}

func (plan *OperationPlan) OpenDB(ctx context.Context) (*sql.DB, error) {
	_ = ctx
	return nil, fmt.Errorf("OpenDB is not supported directly")
}
