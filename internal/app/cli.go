package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"pkg.gostartkit.com/cmd"
)

type cliGlobals struct {
	Connection string
	ConfigDir  string
	DryRun     bool
	Yes        bool
	Format     string
}

type cliBuilder struct {
	in      io.Reader
	out     io.Writer
	err     io.Writer
	globals *cliGlobals
	options Options
}

func NewCommandApp(in io.Reader, out io.Writer, err io.Writer) *cmd.App {
	return newCommandAppWithOptions(in, out, err, Options{})
}

func newCommandAppWithOptions(in io.Reader, out io.Writer, err io.Writer, options Options) *cmd.App {
	globals := &cliGlobals{
		Format: "text",
	}

	builder := &cliBuilder{
		in:      in,
		out:     out,
		err:     err,
		globals: globals,
		options: options,
	}

	cli := cmd.NewApp("dbx")
	cli.Out = out
	cli.Err = err
	cli.Short = "Interactive MySQL database REPL with native SSH support"
	cli.Long = "dbx starts in interactive mode and guides database operations without requiring raw SQL from users."
	cli.SetFlags = builder.setGlobalFlags
	cli.Root = &cmd.Command{
		UsageLine: "dbx [flags] [command]",
		Short:     cli.Short,
		Long:      helpEntries[""].body,
		Run:       builder.runRoot,
	}
	cli.Commands = []*cmd.Command{
		builder.connectCommand(),
		builder.connectionsCommand(),
		builder.connectionGroupCommand(),
		builder.createGroupCommand(),
		builder.listGroupCommand(),
		builder.dropGroupCommand(),
		builder.statusCommand(),
	}

	return cli
}

func (b *cliBuilder) setGlobalFlags(f *cmd.FlagSet) {
	f.StringVar(&b.globals.Connection, "connection", "", "saved connection name", "")
	f.StringVar(&b.globals.ConfigDir, "config-dir", "", "override config directory", "")
	f.BoolVar(&b.globals.DryRun, "dry-run", false, "render SQL without executing it", "")
	f.BoolVar(&b.globals.Yes, "yes", false, "skip confirmation prompts", "y")
	f.StringVar(&b.globals.Format, "format", "text", "output format", "")
	f.SetEnum("format", "text", "json")
}

func (b *cliBuilder) runRoot(ctx context.Context, _ *cmd.Command, args []string) error {
	application, err := NewWithOptions(b.in, b.out, b.err, b.applicationOptions())
	if err != nil {
		return err
	}
	defer application.Close()

	application.dryRun = b.globals.DryRun
	return application.Run(ctx)
}

func (b *cliBuilder) withApplication(ctx context.Context, fn func(application *Application) error) error {
	application, err := NewWithOptions(b.in, b.out, b.err, b.applicationOptions())
	if err != nil {
		return err
	}
	defer application.Close()

	application.dryRun = b.globals.DryRun
	return fn(application)
}

func (b *cliBuilder) applicationOptions() Options {
	options := b.options
	options.ConfigDir = b.globals.ConfigDir
	return options
}

func (b *cliBuilder) writeOutput(value any, text func() error) error {
	if strings.EqualFold(b.globals.Format, "json") {
		encoder := json.NewEncoder(b.out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	return text()
}

func (b *cliBuilder) confirm(ctx context.Context, application *Application, label string, defaultYes bool) (bool, error) {
	if b.globals.Yes {
		return true, nil
	}
	return application.confirm(ctx, label, defaultYes)
}

func (b *cliBuilder) requireNoArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))
}
