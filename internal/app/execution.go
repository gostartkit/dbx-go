package app

import (
	"context"
	"database/sql"

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

func (a *Application) executePlan(ctx context.Context, plan *tpl.ExecutionPlan, runner transactionStarter) error {
	if plan.Transaction {
		tx, err := runner.BeginTx(ctx, nil)
		if err != nil {
			return util.WrapLayer("sql execution", "begin transaction", err)
		}

		for _, action := range plan.Actions {
			if _, err := tx.ExecContext(ctx, action.SQL); err != nil {
				a.prompt.Printf("[FAIL] %s\n", action.Description)
				if rollbackErr := tx.Rollback(); rollbackErr == nil {
					a.prompt.Println("Rolled back transaction.")
				}
				return util.WrapLayer("sql execution", "execute statement", err)
			}
			a.prompt.Printf("[OK] %s\n", action.Description)
		}

		if err := tx.Commit(); err != nil {
			return util.WrapLayer("sql execution", "commit transaction", err)
		}
		a.prompt.Println("Committed transaction.")
		return nil
	}

	for _, action := range plan.Actions {
		if _, err := runner.ExecContext(ctx, action.SQL); err != nil {
			a.prompt.Printf("[FAIL] %s\n", action.Description)
			return util.WrapLayer("sql execution", "execute statement", err)
		}
		a.prompt.Printf("[OK] %s\n", action.Description)
	}
	return nil
}

func (a *Application) runPlan(ctx context.Context, plan *tpl.ExecutionPlan, runner transactionStarter, dryRun bool) error {
	if dryRun {
		a.reportDryRun(plan)
		return nil
	}
	return a.executePlan(ctx, plan, runner)
}
