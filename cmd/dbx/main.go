package main

import (
	"context"
	"fmt"
	"os"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/app"
)

func main() {
	application, err := app.New(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cli := cmd.NewApp("dbx")
	cli.Short = "Interactive MySQL database REPL with native SSH support"
	cli.Long = "dbx starts in interactive mode and guides database operations without requiring raw SQL from users."
	cli.Root = &cmd.Command{
		UsageLine: "dbx",
		Short:     cli.Short,
		Run: func(ctx context.Context, _ *cmd.Command, _ []string) error {
			return application.Run(ctx)
		},
	}

	if err := cli.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
