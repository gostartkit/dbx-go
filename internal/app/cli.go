package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
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
	out := io.Discard
	if application != nil {
		connector = application.connector
		if application.out != nil {
			out = application.out
		}
		if application.store != nil {
			configDir = application.store.RootDir
		}
	}
	return &cliBuilder{
		mode:        ModeREPL,
		in:          nil,
		out:         out,
		err:         out,
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
	cli.Long = "dbx is a lightweight MySQL shell with shared CLI and REPL commands."
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
		b.useGroupCommand(),
		b.showGroupCommand(),
		b.createGroupCommand(),
		b.dropGroupCommand(),
		b.runGroupCommand(),
		b.doctorGroupCommand(),
		b.auditGroupCommand(),
		b.helpCommand(),
		b.exitCommand(),
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

func (b *cliBuilder) runRoot(ctx context.Context, root *cmd.Command, args []string) error {
	if b.mode == ModeREPL {
		if len(args) > 0 {
			return util.WrapLayer("validation", "command", unknownCommandError(args[0], root.SubCommands))
		}
		return nil
	}
	if len(args) > 0 {
		return util.WrapLayer("validation", "command", unknownCommandError(args[0], root.SubCommands))
	}
	application, err := NewWithOptions(b.in, b.out, b.err, b.applicationOptions())
	if err != nil {
		return err
	}
	defer application.Close()

	application.dryRun = b.globals.DryRun
	return application.Run(ctx)
}

func unknownCommandError(name string, commands []*cmd.Command) error {
	suggestions := suggestCommands(name, commands)
	if len(suggestions) == 0 {
		return fmt.Errorf("unknown command %q", name)
	}
	if len(suggestions) == 1 {
		return fmt.Errorf("unknown command %q. Did you mean %s?", name, suggestions[0])
	}
	return fmt.Errorf("unknown command %q. Did you mean %s?", name, strings.Join(suggestions, " or "))
}

func suggestCommands(name string, commands []*cmd.Command) []string {
	type candidate struct {
		name     string
		distance int
	}

	seen := map[string]struct{}{}
	candidates := make([]candidate, 0, len(commands))
	for _, command := range commands {
		if command == nil || command.Hidden {
			continue
		}
		candidateName := strings.TrimSpace(command.Name)
		if candidateName == "" {
			continue
		}
		if _, ok := seen[candidateName]; ok {
			continue
		}
		seen[candidateName] = struct{}{}

		distance := editDistance(strings.ToLower(name), strings.ToLower(candidateName))
		if strings.HasPrefix(strings.ToLower(candidateName), strings.ToLower(name)) || strings.HasPrefix(strings.ToLower(name), strings.ToLower(candidateName)) {
			distance = 0
		}
		if distance > 2 {
			continue
		}
		candidates = append(candidates, candidate{name: candidateName, distance: distance})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].distance != candidates[j].distance {
			return candidates[i].distance < candidates[j].distance
		}
		return candidates[i].name < candidates[j].name
	})

	limit := 3
	if len(candidates) < limit {
		limit = len(candidates)
	}
	results := make([]string, 0, limit)
	for _, candidate := range candidates[:limit] {
		results = append(results, candidate.name)
	}
	return results
}

func editDistance(a string, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = minInt(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func minInt(values ...int) int {
	best := values[0]
	for _, value := range values[1:] {
		if value < best {
			best = value
		}
	}
	return best
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

func (b *cliBuilder) exitCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "exit",
		Aliases:   []string{"quit", "q"},
		UsageLine: "dbx exit",
		Short:     "Exit the shell",
		Run: func(context.Context, *cmd.Command, []string) error {
			if b.mode == ModeREPL {
				return errREPLExit
			}
			return nil
		},
	}
}

func (b *cliBuilder) helpCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "help",
		UsageLine: "dbx help [command...]",
		Short:     "Show command help",
		Long:      helpEntries[""].body,
		Run: func(_ context.Context, _ *cmd.Command, args []string) error {
			topic := normalizeHelpTopic(strings.Join(args, " "))
			if b.mode == ModeREPL {
				if b.application == nil {
					return fmt.Errorf("repl application is not configured")
				}
				return b.application.handleHelp(topic)
			}
			return printHelpTopic(writerPrinter{w: b.out}, topic)
		},
	}
}
