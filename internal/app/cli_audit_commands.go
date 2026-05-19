package app

import (
	"context"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (b *cliBuilder) auditGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "audit",
		UsageLine: "dbx audit <subcommand>",
		Short:     "Inspect local audit records",
		SubCommands: []*cmd.Command{
			b.auditLogCommand(),
		},
	}
}

func (b *cliBuilder) auditLogCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "log",
		UsageLine: "dbx audit log",
		Short:     "Show recent audit log entries",
		Long:      helpEntries["audit log"].body,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "audit log", err)
			}
			return b.withApplication(ctx, func(application *Application) error {
				return application.auditCommand(ctx, auditMetadata{Command: "audit log"}, func(meta *auditMetadata) error {
					result, err := application.loadAuditLog()
					if err != nil {
						return util.WrapLayer("config", "load audit log", err)
					}
					return b.writeOutput(result, func() error {
						application.printAuditLog(result)
						return nil
					})
				})
			})
		},
	}
}
