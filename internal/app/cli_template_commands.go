package app

import (
	"context"
	"fmt"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

type templateRunFlags struct {
	inputs  inputValues
	preview bool
	verbose bool
}

type showTemplatesFlags struct {
	tag string
}

type templateDescribeFlags struct {
	verbose bool
}

func (b *cliBuilder) templateGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "template",
		UsageLine: "dbx template <subcommand>",
		Short:     "Template workflow commands",
		SubCommands: []*cmd.Command{
			b.templateRunCommand(),
			b.templateShowCommand(),
			b.templateDescribeCommand(),
			b.templateValidateCommand(),
		},
	}
}

func (b *cliBuilder) runGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "run",
		UsageLine: "dbx run <subcommand>",
		Short:     "Compatibility aliases for runnable workflows",
		SubCommands: []*cmd.Command{
			b.runTemplateCommand(),
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

func (b *cliBuilder) templatesCommand() *cmd.Command {
	command := b.showTemplatesCommand()
	command.Name = "templates"
	command.UsageLine = "dbx templates"
	command.Short = "Alias for show templates"
	return command
}

func (b *cliBuilder) templateShowCommand() *cmd.Command {
	command := b.templateDescribeCommand()
	command.Name = "show"
	command.UsageLine = "dbx template show <name> [flags]"
	command.Short = "Alias for describe template"
	return command
}

func (b *cliBuilder) templateDescribeCommand() *cmd.Command {
	flags := &templateDescribeFlags{}
	return &cmd.Command{
		Name:        "describe",
		UsageLine:   "dbx template describe <name> [flags]",
		Short:       "Alias for describe template",
		Long:        helpEntries["describe template"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "template name", Required: true, Completion: b.completeTemplates}},
		SetFlags: func(f *cmd.FlagSet) {
			f.BoolVar(&flags.verbose, "verbose", false, "include redacted SQL preview", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "describe template", fmt.Errorf("usage: describe template <name> [--verbose]"))
				}
				return b.application.handleDescribeTemplate(ctx, args[0], flags.verbose)
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "describe template", fmt.Errorf("usage: dbx template describe <name> [flags]"))
			}
			return b.runDescribeTemplate(ctx, args[0], flags)
		},
	}
}

func (b *cliBuilder) runTemplateCommand() *cmd.Command {
	command := b.templateRunCommand()
	command.Name = "template"
	command.UsageLine = "dbx run template <name> [flags]"
	command.Short = "Alias for template run"
	return command
}

func (b *cliBuilder) templateRunCommand() *cmd.Command {
	flags := &templateRunFlags{inputs: inputValues{}}
	return &cmd.Command{
		Name:        "run",
		UsageLine:   "dbx template run <name> [flags]",
		Short:       "Run a workflow template",
		Long:        helpEntries["template run"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "template name", Required: true, Completion: b.completeTemplates}},
		SetFlags: func(f *cmd.FlagSet) {
			bindInputFlag(f, flags.inputs)
			f.BoolVar(&flags.preview, "preview", false, "render the workflow plan without executing", "")
			f.BoolVar(&flags.verbose, "verbose", false, "include redacted SQL preview", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "template run", fmt.Errorf("usage: template run <name> [flags]"))
				}
				return b.application.handleTemplateRun(ctx, args[0], flags.preview, flags.verbose, b.globals.DryRun)
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "template run", fmt.Errorf("usage: dbx template run <name> [flags]"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "template run", DryRun: b.globals.DryRun || flags.preview}, func(application *Application, meta *auditMetadata) error {
				return b.runTemplateWorkflow(ctx, application, args[0], flags, meta)
			})
		},
	}
}

func (b *cliBuilder) templateValidateCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "validate",
		UsageLine:   "dbx template validate <name>",
		Short:       "Validate a workflow template",
		Long:        helpEntries["template validate"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "template name", Required: true, Completion: b.completeTemplates}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				if len(args) != 1 {
					return util.WrapLayer("validation", "template validate", fmt.Errorf("usage: template validate <name>"))
				}
				return b.application.handleTemplateValidate(ctx, args[0])
			}
			if len(args) != 1 {
				return util.WrapLayer("validation", "template validate", fmt.Errorf("usage: dbx template validate <name>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "template validate"}, func(application *Application, meta *auditMetadata) error {
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
		},
	}
}

func (b *cliBuilder) runDescribeTemplate(ctx context.Context, name string, flags *templateDescribeFlags) error {
	return b.withAuditedApplication(ctx, auditMetadata{Command: "describe template"}, func(application *Application, meta *auditMetadata) error {
		cfg, err := application.templateScopeConfig(b.globals.Connection)
		if err != nil {
			return err
		}
		if cfg != nil && cfg.Name != "" {
			meta.Connection = cfg.Name
			meta.Mode = cfg.Mode
		}
		result, err := application.describeTemplateResult(cfg, name, flags.verbose)
		if err != nil {
			return err
		}
		return b.writeOutput(result, func() error {
			application.printTemplateDescription(result, flags.verbose)
			return nil
		})
	})
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
		if err := b.requireCLIConfirmation("template run"); err != nil {
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
