package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/app"
	"pkg.gostartkit.com/dbx/internal/util"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cli := app.NewCommandApp(os.Stdin, os.Stdout, os.Stderr)

	if err := cli.RunWith(ctx, cmd.AutoRuntime{
		Args: os.Args[1:],
		In:   os.Stdin,
		Out:  os.Stdout,
		Err:  os.Stderr,
	}); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if util.IsOutputHandled(err) {
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
