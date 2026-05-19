package repl

import (
	"context"
	"errors"
	"io"

	"pkg.gostartkit.com/dbx/internal/ui"
)

type Handler func(ctx context.Context, line string) (bool, error)

type REPL struct {
	prompt  *ui.Prompt
	handler Handler
}

func New(prompt *ui.Prompt, handler Handler) *REPL {
	return &REPL{
		prompt:  prompt,
		handler: handler,
	}
}

func (r *REPL) Run(ctx context.Context) error {
	for {
		type promptResult struct {
			line string
			err  error
		}

		resultCh := make(chan promptResult, 1)
		go func() {
			line, err := r.prompt.ReadPrompt("dbx> ")
			resultCh <- promptResult{line: line, err: err}
		}()

		var (
			line string
			err  error
		)

		select {
		case <-ctx.Done():
			r.prompt.Println()
			return nil
		case result := <-resultCh:
			line = result.line
			err = result.err
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				r.prompt.Println()
				return nil
			}
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				r.prompt.Println()
				return nil
			}
			return err
		}

		exit, err := r.handler(ctx, line)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				r.prompt.Println()
				return nil
			}
			r.prompt.Printf("Error: %v\n", err)
			continue
		}
		if exit {
			return nil
		}
	}
}
