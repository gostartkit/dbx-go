package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/ui"
)

type fakeRunner struct {
	beginCount    int
	execCount     int
	commitCount   int
	rollbackCount int
	failOnExec    int
}

func (r *fakeRunner) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	r.execCount++
	if r.failOnExec > 0 && r.execCount == r.failOnExec {
		return nil, errors.New("exec failed")
	}
	return fakeResult(0), nil
}

func (r *fakeRunner) BeginTx(ctx context.Context, opts *sql.TxOptions) (transaction, error) {
	r.beginCount++
	return &fakeTx{runner: r}, nil
}

type fakeTx struct {
	runner *fakeRunner
}

func (t *fakeTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.runner.ExecContext(ctx, query, args...)
}

func (t *fakeTx) Commit() error {
	t.runner.commitCount++
	return nil
}

func (t *fakeTx) Rollback() error {
	t.runner.rollbackCount++
	return nil
}

type fakeResult int64

func (r fakeResult) LastInsertId() (int64, error) {
	return int64(r), nil
}

func (r fakeResult) RowsAffected() (int64, error) {
	return int64(r), nil
}

func newTestApplication() *Application {
	var out bytes.Buffer
	return &Application{
		prompt: ui.NewPrompt(strings.NewReader(""), &out),
	}
}

func TestExecutePlanTransaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		transaction  bool
		failOnExec   int
		wantBegin    int
		wantCommit   int
		wantRollback int
		wantErr      bool
	}{
		{name: "commit on success", transaction: true, wantBegin: 1, wantCommit: 1},
		{name: "rollback on failure", transaction: true, failOnExec: 2, wantBegin: 1, wantRollback: 1, wantErr: true},
		{name: "no transaction", transaction: false, wantBegin: 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := newTestApplication()
			runner := &fakeRunner{failOnExec: tc.failOnExec}
			plan := &tpl.ExecutionPlan{
				Transaction: tc.transaction,
				Actions: []tpl.RenderedAction{
					{Description: "one", SQL: "ONE"},
					{Description: "two", SQL: "TWO"},
				},
			}

			_, err := app.executePlan(context.Background(), plan, runner)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if runner.beginCount != tc.wantBegin || runner.commitCount != tc.wantCommit || runner.rollbackCount != tc.wantRollback {
				t.Fatalf("runner counts = %+v", runner)
			}
		})
	}
}

func TestRunPlanDryRunSkipsTransaction(t *testing.T) {
	t.Parallel()

	app := newTestApplication()
	runner := &fakeRunner{}
	plan := &tpl.ExecutionPlan{
		Transaction: true,
		Actions: []tpl.RenderedAction{
			{Description: "one", SQL: "ONE"},
		},
	}

	if _, err := app.runPlan(context.Background(), plan, runner, true); err != nil {
		t.Fatalf("runPlan returned error: %v", err)
	}
	if runner.beginCount != 0 || runner.execCount != 0 {
		t.Fatalf("runner counts = %+v", runner)
	}
}

func TestExecutePlanIncludesActionDurations(t *testing.T) {
	t.Parallel()

	app := newTestApplication()
	runner := &fakeRunner{}
	plan := &tpl.ExecutionPlan{
		Actions: []tpl.RenderedAction{
			{Description: "one", SQL: "ONE"},
		},
	}

	result, err := app.executePlan(context.Background(), plan, runner)
	if err != nil {
		t.Fatalf("executePlan returned error: %v", err)
	}
	if len(result.Actions) != 1 || result.Actions[0].DurationMS <= 0 {
		t.Fatalf("unexpected action duration: %+v", result.Actions)
	}
}

func TestPrintPlanResultIncludesDuration(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := &Application{
		prompt: ui.NewPrompt(strings.NewReader(""), &out),
	}

	app.printPlanResult(&PlanExecutionResult{
		Actions: []ActionResult{
			{Description: "Create database", Status: ActionStatusOK, DurationMS: 124},
		},
	})

	if !strings.Contains(out.String(), "[OK] Create database (124ms)") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
