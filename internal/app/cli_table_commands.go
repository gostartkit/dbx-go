package app

import (
	"context"
	"fmt"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (b *cliBuilder) showCreateGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "create",
		UsageLine: "dbx show create <subcommand>",
		Short:     "Show CREATE statements",
		SubCommands: []*cmd.Command{
			b.showCreateTableCommand(),
		},
	}
}

func (b *cliBuilder) showTableGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "table",
		UsageLine: "dbx show table <subcommand>",
		Short:     "Show table details",
		SubCommands: []*cmd.Command{
			b.showTableStatusCommand(),
		},
	}
}

func (b *cliBuilder) truncateGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "truncate",
		UsageLine: "dbx truncate <subcommand>",
		Short:     "Truncate database resources",
		SubCommands: []*cmd.Command{
			b.truncateTableCommand(),
		},
	}
}

func (b *cliBuilder) renameGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "rename",
		UsageLine: "dbx rename <subcommand>",
		Short:     "Rename database resources",
		SubCommands: []*cmd.Command{
			b.renameTableCommand(),
		},
	}
}

func (b *cliBuilder) showCreateTableCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "table",
		UsageLine:   "dbx show create table <table>",
		Short:       "Show CREATE TABLE for a table",
		Long:        helpEntries["show create table"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "show create table", fmt.Errorf("usage: dbx show create table <table>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show create table", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowCreateTable(ctx, application, args[0], meta)
			})
		},
	}
}

func (b *cliBuilder) showTableStatusCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "status",
		UsageLine:   "dbx show table status [table]",
		Short:       "Show table status for one or more tables",
		Long:        helpEntries["show table status"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name"}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) > 1 {
				return util.WrapLayer("validation", "show table status", fmt.Errorf("usage: dbx show table status [table]"))
			}
			table := ""
			if len(args) == 1 {
				table = args[0]
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show table status", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runShowTableStatus(ctx, application, table, meta)
			})
		},
	}
}

func (b *cliBuilder) truncateTableCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "table",
		UsageLine:   "dbx truncate table <table>",
		Short:       "Delete all rows from a table",
		Long:        helpEntries["truncate table"].body,
		Positionals: []cmd.PositionalArg{{Name: "table", Usage: "table name", Required: true}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "truncate table", fmt.Errorf("usage: dbx truncate table <table>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "truncate table", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runTruncateTable(ctx, application, args[0], meta)
			})
		},
	}
}

func (b *cliBuilder) renameTableCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "table",
		UsageLine:   "dbx rename table <from> <to>",
		Short:       "Rename a table",
		Long:        helpEntries["rename table"].body,
		Positionals: []cmd.PositionalArg{
			{Name: "from", Usage: "existing table name", Required: true},
			{Name: "to", Usage: "new table name", Required: true},
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 2 {
				return util.WrapLayer("validation", "rename table", fmt.Errorf("usage: dbx rename table <from> <to>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "rename table", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runRenameTable(ctx, application, args[0], args[1], meta)
			})
		},
	}
}

