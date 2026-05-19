package app

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/util"
)

type statementExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type transaction interface {
	statementExecutor
	Commit() error
	Rollback() error
}

type transactionStarter interface {
	statementExecutor
	BeginTx(ctx context.Context, opts *sql.TxOptions) (transaction, error)
}

type sqlRunner struct {
	db *sql.DB
}

func (r sqlRunner) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.db.ExecContext(ctx, query, args...)
}

func (r sqlRunner) BeginTx(ctx context.Context, opts *sql.TxOptions) (transaction, error) {
	tx, err := r.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return sqlTransaction{tx: tx}, nil
}

type sqlTransaction struct {
	tx *sql.Tx
}

func (t sqlTransaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t sqlTransaction) Commit() error {
	return t.tx.Commit()
}

func (t sqlTransaction) Rollback() error {
	return t.tx.Rollback()
}

func (a *Application) executePlan(ctx context.Context, plan *tpl.ExecutionPlan, runner transactionStarter) (*PlanExecutionResult, error) {
	result := &PlanExecutionResult{
		OK:          true,
		Template:    plan.TemplateName,
		Layer:       plan.Layer,
		Source:      plan.Source,
		Transaction: plan.Transaction,
		Actions:     make([]ActionResult, 0, len(plan.Actions)),
	}

	if plan.Transaction {
		tx, err := runner.BeginTx(ctx, nil)
		if err != nil {
			result.OK = false
			return result, util.WrapLayer("sql execution", "begin transaction", err)
		}

		for _, action := range plan.Actions {
			startedAt := time.Now()
			if _, err := tx.ExecContext(ctx, action.SQL); err != nil {
				result.OK = false
				result.Actions = append(result.Actions, ActionResult{
					Description: action.Description,
					SQL:         action.SQL,
					Status:      ActionStatusFailed,
					DurationMS:  measuredMilliseconds(startedAt),
				})
				if rollbackErr := tx.Rollback(); rollbackErr == nil {
					result.RolledBack = true
				}
				return result, util.WrapLayer("sql execution", "execute statement", err)
			}
			result.Actions = append(result.Actions, ActionResult{
				Description: action.Description,
				SQL:         action.SQL,
				Status:      ActionStatusOK,
				DurationMS:  measuredMilliseconds(startedAt),
			})
		}

		if err := tx.Commit(); err != nil {
			result.OK = false
			return result, util.WrapLayer("sql execution", "commit transaction", err)
		}
		result.Committed = true
		return result, nil
	}

	for _, action := range plan.Actions {
		startedAt := time.Now()
		if _, err := runner.ExecContext(ctx, action.SQL); err != nil {
			result.OK = false
			result.Actions = append(result.Actions, ActionResult{
				Description: action.Description,
				SQL:         action.SQL,
				Status:      ActionStatusFailed,
				DurationMS:  measuredMilliseconds(startedAt),
			})
			return result, util.WrapLayer("sql execution", "execute statement", err)
		}
		result.Actions = append(result.Actions, ActionResult{
			Description: action.Description,
			SQL:         action.SQL,
			Status:      ActionStatusOK,
			DurationMS:  measuredMilliseconds(startedAt),
		})
	}
	return result, nil
}

func (a *Application) runPlan(ctx context.Context, plan *tpl.ExecutionPlan, runner transactionStarter, dryRun bool) (*PlanExecutionResult, error) {
	result := &PlanExecutionResult{
		OK:          true,
		Template:    plan.TemplateName,
		Layer:       plan.Layer,
		Source:      plan.Source,
		Transaction: plan.Transaction,
		DryRun:      dryRun,
		Actions:     make([]ActionResult, 0, len(plan.Actions)),
	}

	if dryRun {
		for _, action := range plan.Actions {
			result.Actions = append(result.Actions, ActionResult{
				Description: action.Description,
				SQL:         action.SQL,
				Status:      ActionStatusDryRun,
			})
		}
		return result, nil
	}
	return a.executePlan(ctx, plan, runner)
}

func (a *Application) printPlanResult(result *PlanExecutionResult) {
	if result == nil {
		return
	}

	for _, action := range result.Actions {
		switch action.Status {
		case ActionStatusDryRun:
			a.prompt.Printf("[DRY-RUN] %s\n", action.Description)
		case ActionStatusFailed:
			a.prompt.Printf("[FAIL] %s%s\n", action.Description, formatActionDuration(action.DurationMS))
		default:
			a.prompt.Printf("[OK] %s%s\n", action.Description, formatActionDuration(action.DurationMS))
		}
	}

	if result.RolledBack {
		a.prompt.Println("Rolled back transaction.")
	}
	if result.Committed {
		a.prompt.Println("Committed transaction.")
	}
}

func measuredMilliseconds(startedAt time.Time) int64 {
	duration := time.Since(startedAt).Milliseconds()
	if duration <= 0 {
		return 1
	}
	return duration
}

func formatActionDuration(durationMS int64) string {
	if durationMS <= 0 {
		return ""
	}
	return " (" + strconv.FormatInt(durationMS, 10) + "ms)"
}
