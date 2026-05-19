package app

import (
	"context"
	"fmt"

	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) confirmExecutionIfNeeded(ctx context.Context, command string) (bool, error) {
	behavior := behaviorForCommand(command)
	if !behavior.RequiresConfirmation || (a.dryRun && behavior.SkipConfirmOnDryRun) {
		return true, nil
	}
	return a.confirm(ctx, "Confirm execution?", true)
}

func (b *cliBuilder) requireCLIConfirmation(command string) error {
	behavior := behaviorForCommand(command)
	if !behavior.RequiresConfirmation || b.globals.Yes || (b.globals.DryRun && behavior.SkipConfirmOnDryRun) {
		return nil
	}
	return util.WrapLayer("validation", command, fmt.Errorf("confirmation required: rerun with --yes"))
}
