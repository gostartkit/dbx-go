package app

import (
	"context"
	"fmt"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

type templateRunFlags struct {
	inputs   inputValues
	preview  bool
	verbose  bool
	validate bool
}

type showTemplatesFlags struct {
	tag string
}

func (b *cliBuilder) runGroupCommand() *cmd.Command {
	subcommands := []*cmd.Command{
		b.runTemplateCommand(),
	}
	return &cmd.Command{
		Name:        "run",
		UsageLine:   "dbx run <subcommand>",
		Short:       "Run workflows",
		Long:        helpEntries["run"].body,
		SubCommands: subcommands,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) == 0 {
				usage := "dbx run <subcommand>"
				if b.mode == ModeREPL {
					usage = "run <subcommand>"
				}
				return util.WrapLayer("validation", "run", fmt.Errorf("usage: %s", usage))
			}
			return util.WrapLayer("validation", "run", unknownTargetError("run", args[0], subcommands))
		},
	}
}

func (b *cliBuilder) showTemplatesCommand() *cmd.Command {
	flags := &showTemplatesFlags{}
	return &cmd.Command{
		Name:        "templates",
		UsageLine:   "dbx show templates [query] [--tag value]",
		Short:       "List resolved workflow templates",
		Long:        helpEntries["show templates"].body,
		Positionals: []cmd.PositionalArg{{Name: "query", Usage: "optional substring filter"}},
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.tag, "tag", "", "filter by template tag", "")
			f.SetCompletion("tag", b.completeTemplateTags)
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				filters := templateListFilters{Tag: flags.tag}
				if len(args) > 1 {
					return util.WrapLayer("validation", "show templates", fmt.Errorf("usage: show templates [query] [--tag value]"))
				}
				if len(args) == 1 {
					filters.Query = args[0]
				}
				return b.application.handleShowTemplates(ctx, filters)
			}
			if len(args) > 1 {
				return util.WrapLayer("validation", "show templates", fmt.Errorf("usage: dbx show templates [query] [--tag value]"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show templates"}, func(application *Application, meta *auditMetadata) error {
				cfg, err := application.templateScopeConfig(b.globals.Connection)
				if err != nil {
					return err
				}
				if cfg != nil && cfg.Name != "" {
					meta.Connection = cfg.Name
					meta.Mode = cfg.Mode
				}
				filters := templateListFilters{Tag: flags.tag}
				if len(args) == 1 {
					filters.Query = args[0]
				}
				result, err := application.showTemplatesResult(cfg, filters)
				if err != nil {
					return err
				}
				return b.writeOutput(result, func() error {
					application.printTemplatesCatalog(result)
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) runTemplateCommand() *cmd.Command {
	flags := &templateRunFlags{inputs: inputValues{}}
	return &cmd.Command{
		Name:        "template",
		UsageLine:   "dbx run template <name> [flags]",
		Short:       "Run a workflow template",
		Long:        helpEntries["template run"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "template name", Required: true, Completion: b.completeTemplates}},
		SetFlags: func(f *cmd.FlagSet) {
			bindInputFlag(f, flags.inputs)
			f.BoolVar(&flags.preview, "preview", false, "render the workflow plan without executing", "")
			f.BoolVar(&flags.verbose, "verbose", false, "include redacted SQL preview", "")
			f.BoolVar(&flags.validate, "validate", false, "validate the template without running it", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "run template", fmt.Errorf("usage: dbx run template <name> [flags]"))
			}
			if flags.validate {
				if b.mode == ModeREPL {
					return b.application.handleTemplateValidate(ctx, args[0])
				}
				return b.withAuditedApplication(ctx, auditMetadata{Command: "run template"}, func(application *Application, meta *auditMetadata) error {
					cfg, err := application.templateScopeConfig(b.globals.Connection)
					if err != nil {
						return err
					}
					if cfg != nil && cfg.Name != "" {
						meta.Connection = cfg.Name
						meta.Mode = cfg.Mode
					}
					result, err := application.templateValidateResult(cfg, args[0])
					if err != nil {
						return err
					}
					return b.writeOutput(result, func() error {
						application.printTemplateValidation(result)
						return nil
					})
				})
			}
			if b.mode == ModeREPL {
				return b.application.handleTemplateRun(ctx, args[0], flags.preview, flags.verbose, b.globals.DryRun)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "run template", DryRun: b.globals.DryRun || flags.preview}, func(application *Application, meta *auditMetadata) error {
				return b.runTemplateWorkflow(ctx, application, args[0], flags, meta)
			})
		},
	}
}

func (b *cliBuilder) runTemplateWorkflow(ctx context.Context, application *Application, name string, flags *templateRunFlags, meta *auditMetadata) error {
	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if !flags.preview {
		if err := b.requireCLIConfirmation("run template"); err != nil {
			return err
		}
	}

	result, err := application.templateRunResult(ctx, cfg, name, flags.inputs, flags.preview, b.globals.DryRun, flags.verbose, b.globals.Database)
	if err != nil && result == nil {
		return err
	}
	if writeErr := b.writeOutput(result, func() error {
		application.printTemplateRunResult(result)
		return err
	}); writeErr != nil {
		return writeErr
	}
	if err != nil {
		return util.MarkOutputHandled(err)
	}
	return err
}
