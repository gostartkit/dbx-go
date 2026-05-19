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
		line, err := r.prompt.ReadPrompt("dbx> ")
		if err != nil {
			if errors.Is(err, io.EOF) {
				r.prompt.Println()
				return nil
			}
			return err
		}

		exit, err := r.handler(ctx, line)
		if err != nil {
			r.prompt.Printf("Error: %v\n", err)
			continue
		}
		if exit {
			return nil
		}
	}
}
