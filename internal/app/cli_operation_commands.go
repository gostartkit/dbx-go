package app

import (
	"context"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

type execOperationFlags struct {
	inputs   inputValues
	preview  bool
	verbose  bool
	validate bool
}

type showTemplatesFlags struct {
	tag string
}

func (b *cliBuilder) execGroupCommand() *cmd.Command {
	flags := &execOperationFlags{inputs: inputValues{}}
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "exec",
		UsageFallback: "dbx exec <operation> [flags]",
		ShortFallback: "Execute an operation",
		Positionals: b.manifestPositionals("exec", []cmd.PositionalArg{{
			Name:       "operation",
			Usage:      "operation name",
			Required:   true,
			Completion: b.completeOperations,
		}}),
		SetFlags: func(f *cmd.FlagSet) {
			bindInputFlag(f, flags.inputs)
			b.bindManifestBoolFlag(f, "exec", "preview", &flags.preview, false, "render the workflow plan without executing")
			b.bindManifestBoolFlag(f, "exec", "verbose", &flags.verbose, false, "include redacted SQL preview")
			b.bindManifestBoolFlag(f, "exec", "validate", &flags.validate, false, "validate the operation without running it")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if flags.validate {
				if b.mode == ModeREPL {
					return b.application.handleExecValidate(ctx, args[0])
				}
				return b.withAuditedApplication(ctx, auditMetadata{Command: "exec"}, func(application *Application, meta *auditMetadata) error {
					cfg, err := application.commandContext().resolveTemplateScope(b.globals.Connection)
					if err != nil {
						return err
					}
					if cfg != nil && cfg.Name != "" {
						meta.Connection = cfg.Name
						meta.Mode = cfg.Mode
					}
					result, err := application.operationValidateResult(cfg, args[0])
					if err != nil {
						return err
					}
					return b.writeOutput(result, func() error {
						application.printOperationValidation(result)
						return nil
					})
				})
			}
			if b.mode == ModeREPL {
				return b.application.handleExec(ctx, args[0], flags.preview, flags.verbose, b.globals.DryRun)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "exec", DryRun: b.globals.DryRun || flags.preview}, func(application *Application, meta *auditMetadata) error {
				return b.execOperation(ctx, application, args[0], flags, meta)
			})
		},
	})
}

func (b *cliBuilder) showTemplatesCommand() *cmd.Command {
	flags := &showTemplatesFlags{}
	return b.newManifestCommand(manifestCommandOptions{
		Path:          "show templates",
		UsageFallback: "dbx show templates [query] [--tag value]",
		ShortFallback: "List resolved workflow templates",
		Positionals:   b.manifestPositionals("show templates", []cmd.PositionalArg{{Name: "query", Usage: "optional substring filter"}}),
		SetFlags: func(f *cmd.FlagSet) {
			b.bindManifestStringFlag(f, "show templates", "tag", &flags.tag, "", "filter by template tag")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				filters := templateListFilters{Tag: flags.tag}
				if len(args) == 1 {
					filters.Query = args[0]
				}
				return b.application.handleShowTemplates(ctx, filters)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show templates"}, func(application *Application, meta *auditMetadata) error {
				cfg, err := application.commandContext().resolveTemplateScope(b.globals.Connection)
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
	})
}

func (b *cliBuilder) execOperation(ctx context.Context, application *Application, name string, flags *execOperationFlags, meta *auditMetadata) error {
	cfg, err := application.commandContext().resolveCLIConnection(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if !flags.preview {
		if err := b.requireCLIConfirmation("exec"); err != nil {
			return err
		}
	}

	result, err := application.execOperationResult(ctx, cfg, name, flags.inputs, flags.preview, b.globals.DryRun, flags.verbose, b.globals.Database)
	if err != nil && result == nil {
		return err
	}
	if writeErr := b.writeOutput(result, func() error {
		application.printOperationRunResult(result)
		return err
	}); writeErr != nil {
		return writeErr
	}
	if err != nil {
		return util.MarkOutputHandled(err)
	}
	return err
}
