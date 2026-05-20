package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

type cliGlobals struct {
	Connection string
	Database   string
	ConfigDir  string
	DryRun     bool
	Yes        bool
	Format     string
}

type CommandMode string

const (
	ModeCLI  CommandMode = "cli"
	ModeREPL CommandMode = "repl"
)

type completionResolver interface {
	Connections() []CompletionConnection
	Databases() []string
	Tables() []string
	Templates() []string
	TemplateTags() []string
	Users() []string
}

type cliBuilder struct {
	mode    CommandMode
	in      io.Reader
	out     io.Writer
	err     io.Writer
	globals *cliGlobals
	options Options

	application *Application
	resolver    completionResolver
}

func NewCommandApp(in io.Reader, out io.Writer, err io.Writer) *cmd.App {
	return newCommandAppWithOptions(in, out, err, Options{})
}

func newCommandAppWithOptions(in io.Reader, out io.Writer, err io.Writer, options Options) *cmd.App {
	return newCLIBuilder(in, out, err, options).buildApp()
}

func newCLIBuilder(in io.Reader, out io.Writer, err io.Writer, options Options) *cliBuilder {
	globals := &cliGlobals{
		Format: "text",
	}

	return &cliBuilder{
		mode:    ModeCLI,
		in:      in,
		out:     out,
		err:     err,
		globals: globals,
		options: options,
	}
}

func newREPLBuilder(application *Application, resolver completionResolver) *cliBuilder {
	configDir := ""
	var connector connectorClient
	if application != nil {
		connector = application.connector
		if application.store != nil {
			configDir = application.store.RootDir
		}
	}
	return &cliBuilder{
		mode:        ModeREPL,
		in:          nil,
		out:         io.Discard,
		err:         io.Discard,
		globals:     &cliGlobals{Format: "text"},
		options:     Options{ConfigDir: configDir, Connector: connector},
		application: application,
		resolver:    resolver,
	}
}

func (b *cliBuilder) buildApp() *cmd.App {
	cli := cmd.NewApp("dbx")
	cli.Out = b.out
	cli.Err = b.err
	cli.Short = "Interactive MySQL database REPL with native SSH support"
	cli.Long = "dbx starts in interactive mode and guides database operations without requiring raw SQL from users."
	if b.mode == ModeCLI {
		cli.SetFlags = b.setGlobalFlags
	}
	cli.Root = &cmd.Command{
		UsageLine: "dbx [flags] [command]",
		Short:     cli.Short,
		Long:      helpEntries[""].body,
		Run:       b.runRoot,
	}
	cli.Commands = []*cmd.Command{
		b.connectCommand(),
		b.auditGroupCommand(),
		b.createGroupCommand(),
		b.describeCommand(),
		b.doctorGroupCommand(),
		b.dropGroupCommand(),
		b.editGroupCommand(),
		b.peekCommand(),
		b.countCommand(),
		b.renameGroupCommand(),
		b.runGroupCommand(),
		b.sampleCommand(),
		b.showGroupCommand(),
		b.testGroupCommand(),
		b.truncateGroupCommand(),
		b.useGroupCommand(),
		b.validateGroupCommand(),
		b.contextCommand(),
	}
	if b.mode == ModeREPL {
		cli.Commands = append(replOverlayCommands(b), cli.Commands...)
	}
	return cli
}

func (b *cliBuilder) syncREPLGlobals(application *Application) {
	if b.mode != ModeREPL || application == nil {
		return
	}
	b.globals.DryRun = application.dryRun
}

func (b *cliBuilder) setGlobalFlags(f *cmd.FlagSet) {
	f.StringVar(&b.globals.Connection, "connection", "", "saved connection name", "")
	f.StringVar(&b.globals.Database, "database", "", "database name for this command only", "")
	f.StringVar(&b.globals.ConfigDir, "config-dir", "", "override config directory", "")
	f.BoolVar(&b.globals.DryRun, "dry-run", false, "render SQL without executing it", "")
	f.BoolVar(&b.globals.Yes, "yes", false, "skip confirmation prompts", "y")
	f.StringVar(&b.globals.Format, "format", "text", "output format", "")
	f.SetEnum("format", "text", "json")
}

func (b *cliBuilder) runRoot(ctx context.Context, _ *cmd.Command, args []string) error {
	if b.mode == ModeREPL {
		return nil
	}
	application, err := NewWithOptions(b.in, b.out, b.err, b.applicationOptions())
	if err != nil {
		return err
	}
	defer application.Close()

	application.dryRun = b.globals.DryRun
	return application.Run(ctx)
}

func (b *cliBuilder) withApplication(ctx context.Context, fn func(application *Application) error) error {
	if b.mode == ModeREPL {
		if b.application == nil {
			return fmt.Errorf("repl application is not configured")
		}
		b.syncREPLGlobals(b.application)
		return fn(b.application)
	}

	application, err := NewWithOptions(b.in, b.out, b.err, b.applicationOptions())
	if err != nil {
		return err
	}
	defer application.Close()

	application.dryRun = b.globals.DryRun
	err = fn(application)
	if err != nil && strings.EqualFold(b.globals.Format, "json") && !util.IsOutputHandled(err) {
		if writeErr := b.writeOutput(&ErrorEnvelope{
			OK:    false,
			Error: errorResult(err),
		}, func() error {
			return nil
		}); writeErr != nil {
			return writeErr
		}
		return util.MarkOutputHandled(err)
	}
	return err
}

func (b *cliBuilder) withAuditedApplication(ctx context.Context, meta auditMetadata, fn func(application *Application, meta *auditMetadata) error) error {
	return b.withApplication(ctx, func(application *Application) error {
		return application.auditCommand(ctx, meta, func(meta *auditMetadata) error {
			return fn(application, meta)
		})
	})
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
