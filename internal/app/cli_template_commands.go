package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

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
		b.runSQLCommand(),
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

func (b *cliBuilder) runSQLCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "sql",
		UsageLine:   "dbx run sql <sql-or-file>",
		Short:       "Run raw SQL or a SQL file",
		Long:        helpEntries["run sql"].body,
		Positionals: []cmd.PositionalArg{{Name: "sql-or-file", Usage: "SQL text, @file.sql, or path to a .sql file", Required: true}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) == 0 {
				return util.WrapLayer("validation", "run sql", fmt.Errorf("usage: dbx run sql <sql-or-file>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "run sql", DryRun: b.globals.DryRun}, func(application *Application, meta *auditMetadata) error {
				return b.runSQL(ctx, application, args, meta)
			})
		},
	}
}

func (b *cliBuilder) runSQL(ctx context.Context, application *Application, args []string, meta *auditMetadata) error {
	cfg, err := application.resolveConnectionConfig(b.globals.Connection)
	if err != nil {
		return err
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}
	if err := application.applyCLIDatabaseSelection(ctx, cfg, b.globals.Database); err != nil {
		return err
	}
	query, source, err := resolveSQLInput(args)
	if err != nil {
		return err
	}
	if b.globals.DryRun {
		result := &PlanExecutionResult{
			OK:         true,
			Connection: cfg.Name,
			Command:    "run sql",
			Source:     source,
			DryRun:     true,
			Actions: []ActionResult{{
				Description: "Run SQL",
				SQL:         query,
				Status:      ActionStatusDryRun,
			}},
		}
		return b.writeOutput(result, func() error {
			application.printPlanResult(result)
			return nil
		})
	}
	db, err := application.openConnection(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	if looksLikeQuery(query) {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return util.WrapLayer("sql execution", "run sql query", err)
		}
		defer rows.Close()
		lines, err := formatSQLRows(rows)
		if err != nil {
			return err
		}
		return b.writeOutput(map[string]any{"ok": true, "connection": cfg.Name, "source": source, "lines": lines}, func() error {
			for _, line := range lines {
				fmt.Fprintln(b.out, line)
			}
			if len(lines) == 0 {
				fmt.Fprintln(b.out, "No rows returned.")
			}
			return nil
		})
	}
	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return util.WrapLayer("sql execution", "run sql statement", err)
	}
	affected, _ := result.RowsAffected()
	return b.writeOutput(map[string]any{"ok": true, "connection": cfg.Name, "source": source, "rows_affected": affected}, func() error {
		fmt.Fprintf(b.out, "Rows affected: %d\n", affected)
		return nil
	})
}

func resolveSQLInput(args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("missing SQL input")
	}
	if len(args) == 1 {
		candidate := strings.TrimSpace(args[0])
		if strings.HasPrefix(candidate, "@") {
			data, err := os.ReadFile(strings.TrimPrefix(candidate, "@"))
			if err != nil {
				return "", "", util.WrapLayer("sql execution", "read SQL file", err)
			}
			return strings.TrimSpace(string(data)), candidate, nil
		}
		if strings.HasSuffix(candidate, ".sql") {
			if data, err := os.ReadFile(candidate); err == nil {
				return strings.TrimSpace(string(data)), candidate, nil
			}
		}
		return candidate, "inline", nil
	}
	return strings.TrimSpace(strings.Join(args, " ")), "inline", nil
}

func looksLikeQuery(query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(query, "select") || strings.HasPrefix(query, "show") || strings.HasPrefix(query, "describe") || strings.HasPrefix(query, "explain")
}

func formatSQLRows(rows *sql.Rows) ([]string, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, util.WrapLayer("sql execution", "read result columns", err)
	}
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	lines := make([]string, 0)
	lines = append(lines, strings.Join(columns, "\t"))
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, util.WrapLayer("sql execution", "scan SQL row", err)
		}
		parts := make([]string, 0, len(columns))
		for _, value := range values {
			switch typed := value.(type) {
			case nil:
				parts = append(parts, "NULL")
			case []byte:
				parts = append(parts, string(typed))
			default:
				parts = append(parts, fmt.Sprint(typed))
			}
		}
		lines = append(lines, strings.Join(parts, "\t"))
	}
	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "iterate SQL rows", err)
	}
	return lines, nil
}
