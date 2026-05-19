package app

import (
	"context"

	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) ask(ctx context.Context, label string, defaultValue string) (string, error) {
	return withPromptContext(ctx, func() (string, error) {
		return a.prompt.Ask(label, defaultValue)
	})
}

func (a *Application) askPassword(ctx context.Context, label string) (string, error) {
	return withPromptContext(ctx, func() (string, error) {
		return a.prompt.AskPassword(label)
	})
}

func (a *Application) choose(ctx context.Context, label string, options []string, defaultValue string) (string, error) {
	return withPromptContext(ctx, func() (string, error) {
		return a.prompt.Choose(label, options, defaultValue)
	})
}

func (a *Application) confirm(ctx context.Context, label string, defaultYes bool) (bool, error) {
	return withPromptContext(ctx, func() (bool, error) {
		return a.prompt.Confirm(label, defaultYes)
	})
}

func withPromptContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	type result struct {
		value T
		err   error
	}

	resultCh := make(chan result, 1)
	go func() {
		value, err := fn()
		resultCh <- result{value: value, err: err}
	}()

	select {
	case <-ctx.Done():
		var zero T
		return zero, util.WrapLayer("shutdown", "interactive prompt interrupted", ctx.Err())
	case res := <-resultCh:
		return res.value, res.err
	}
}
