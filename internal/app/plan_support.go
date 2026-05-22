package app

import (
	"context"
	"database/sql"
	"fmt"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

func buildPlans(template *tpl.Template, cfg *config.ConnectionConfig, values map[string]string) (*tpl.ExecutionPlan, *tpl.ExecutionPlan, error) {
	plan, err := tpl.BuildPlan(template, cfg, values)
	if err != nil {
		return nil, nil, util.WrapLayer("template", "build execution plan", err)
	}
	previewPlan, err := tpl.BuildPlan(template, cfg, redactTemplateValues(template, values))
	if err != nil {
		return nil, nil, util.WrapLayer("template", "build redacted execution preview", err)
	}
	return plan, previewPlan, nil
}

func cloneInputValues(values inputValues) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func applyPreviewSQL(result *PlanExecutionResult, previewPlan *tpl.ExecutionPlan) {
	if result == nil || previewPlan == nil {
		return
	}

	for index := range result.Actions {
		if index >= len(previewPlan.Actions) {
			return
		}
		result.Actions[index].SQL = previewPlan.Actions[index].SQL
	}
}

type noopTransactionStarter struct{}

func (noopTransactionStarter) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, fmt.Errorf("dry-run execution does not use a SQL runner")
}

func (noopTransactionStarter) BeginTx(context.Context, *sql.TxOptions) (transaction, error) {
	return nil, fmt.Errorf("dry-run execution does not use transactions")
}
