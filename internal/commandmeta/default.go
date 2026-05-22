package commandmeta

import "sync"

type ValueType string

const (
	ValueString     ValueType = "string"
	ValueBool       ValueType = "bool"
	ValueEnum       ValueType = "enum"
	ValueConnection ValueType = "connection"
	ValueDatabase   ValueType = "database"
	ValueTable      ValueType = "table"
	ValueUser       ValueType = "user"
	ValueSchema     ValueType = "schema"
	ValueOperation  ValueType = "operation"
	ValueTemplate   ValueType = "template"
)

type Manifest struct {
	Commands []*Command
}

type Command struct {
	Name        string
	UsageLine   string
	Aliases     []string
	Description string
	Subcommands []*Command
	Flags       []*Flag
	Args        []*Arg
	HandlerName string
	Hidden      bool
}

type Flag struct {
	Name               string
	Short              string
	Description        string
	DefaultValue       string
	ValueType          ValueType
	Required           bool
	Repeatable         bool
	EnumValues         []string
	CompletionProvider string
}

type Arg struct {
	Name               string
	Description        string
	Required           bool
	Repeatable         bool
	ValueType          ValueType
	EnumValues         []string
	CompletionProvider string
}

var (
	defaultManifestOnce sync.Once
	defaultManifestData *Manifest
)

func DefaultManifest() *Manifest {
	defaultManifestOnce.Do(func() {
		defaultManifestData = &Manifest{
			Commands: []*Command{
				{
					Name:        "exec",
					UsageLine:   "exec <operation> [flags]",
					Description: "Execute a named operation.",
					HandlerName: "exec",
					Args: []*Arg{{
						Name:               "operation",
						Description:        "Operation name to execute.",
						Required:           true,
						ValueType:          ValueOperation,
						CompletionProvider: "operation",
					}},
					Flags: []*Flag{
						{Name: "--dry-run", Description: "Render and validate without applying changes.", ValueType: ValueBool},
						{Name: "--validate", Description: "Validate the resolved operation and exit.", ValueType: ValueBool},
						{Name: "--yes", Description: "Skip confirmation prompts.", ValueType: ValueBool},
						{Name: "--preview", Description: "Show the execution preview before running.", ValueType: ValueBool},
						{Name: "--verbose", Description: "Include detailed execution output.", ValueType: ValueBool},
					},
				},
				{
					Name:        "help",
					UsageLine:   "help [topic]",
					Description: "Show help for a command or topic.",
					HandlerName: "help",
					Args: []*Arg{{
						Name:               "topic",
						Description:        "Command or topic to describe.",
						Required:           false,
						ValueType:          ValueString,
						CompletionProvider: "topic",
					}},
				},
				{
					Name:        "template",
					UsageLine:   "template <subcommand>",
					Description: "Template maintenance and rendering commands.",
					HandlerName: "template",
					Hidden:      true,
					Subcommands: []*Command{
						{
							Name:        "list",
							UsageLine:   "template list",
							Description: "List available templates.",
							HandlerName: "template.list",
						},
						{
							Name:        "show",
							UsageLine:   "template show <template>",
							Description: "Show template details.",
							HandlerName: "template.show",
							Args: []*Arg{{
								Name:               "template",
								Description:        "Template name.",
								Required:           true,
								ValueType:          ValueTemplate,
								CompletionProvider: "template",
							}},
						},
						{
							Name:        "render",
							UsageLine:   "template render <template>",
							Description: "Render a template preview.",
							HandlerName: "template.render",
							Args: []*Arg{{
								Name:               "template",
								Description:        "Template name.",
								Required:           true,
								ValueType:          ValueTemplate,
								CompletionProvider: "template",
							}},
							Flags: []*Flag{
								{Name: "--var", Description: "Template input override in key=value form.", ValueType: ValueString},
							},
						},
					},
				},
				{
					Name:        "connect",
					UsageLine:   "connect [name]",
					Description: "Connect to a saved connection.",
					HandlerName: "connect",
					Args: []*Arg{{
						Name:               "name",
						Description:        "Saved connection name.",
						Required:           false,
						ValueType:          ValueConnection,
						CompletionProvider: "connection",
					}},
				},
				{
					Name:        "connection",
					UsageLine:   "connection <subcommand>",
					Description: "Connection management commands.",
					HandlerName: "connection",
					Hidden:      true,
					Subcommands: []*Command{
						{
							Name:        "list",
							UsageLine:   "connection list",
							Description: "List saved connections.",
							HandlerName: "connection.list",
						},
						{
							Name:        "use",
							UsageLine:   "connection use <connection>",
							Description: "Select a saved connection.",
							HandlerName: "connection.use",
							Args: []*Arg{{
								Name:               "name",
								Description:        "Saved connection name.",
								Required:           true,
								ValueType:          ValueConnection,
								CompletionProvider: "connection",
							}},
						},
						{
							Name:        "create",
							UsageLine:   "connection create",
							Description: "Create a saved connection.",
							HandlerName: "connection.create",
						},
						{
							Name:        "delete",
							UsageLine:   "connection delete <connection>",
							Description: "Delete a saved connection.",
							HandlerName: "connection.delete",
							Args: []*Arg{{
								Name:               "name",
								Description:        "Saved connection name.",
								Required:           true,
								ValueType:          ValueConnection,
								CompletionProvider: "connection",
							}},
						},
						{
							Name:        "show",
							UsageLine:   "connection show <connection>",
							Description: "Show a saved connection.",
							HandlerName: "connection.show",
							Args: []*Arg{{
								Name:               "name",
								Description:        "Saved connection name.",
								Required:           true,
								ValueType:          ValueConnection,
								CompletionProvider: "connection",
							}},
						},
					},
				},
				{
					Name:        "database",
					UsageLine:   "database <subcommand>",
					Description: "Database selection commands.",
					HandlerName: "database",
					Hidden:      true,
					Subcommands: []*Command{
						{
							Name:        "use",
							UsageLine:   "database use <database>",
							Description: "Select the current database.",
							HandlerName: "database.use",
							Args: []*Arg{{
								Name:               "database",
								Description:        "Database name.",
								Required:           true,
								ValueType:          ValueDatabase,
								CompletionProvider: "database",
							}},
						},
					},
				},
				{
					Name:        "use",
					UsageLine:   "use <name>",
					Description: "Select the current database.",
					HandlerName: "use",
					Args: []*Arg{{
						Name:               "name",
						Description:        "Database name.",
						Required:           true,
						ValueType:          ValueDatabase,
						CompletionProvider: "database",
					}},
				},
				{
					Name:        "show",
					UsageLine:   "show <subcommand>",
					Description: "Inspect configuration and database state.",
					HandlerName: "show",
					Subcommands: []*Command{
						{
							Name:        "connection",
							UsageLine:   "show connection <name>",
							Description: "Show a saved connection.",
							HandlerName: "show.connection",
							Args: []*Arg{{
								Name:               "name",
								Description:        "Saved connection name.",
								Required:           true,
								ValueType:          ValueConnection,
								CompletionProvider: "connection",
							}},
						},
						{
							Name:        "columns",
							UsageLine:   "show columns <table>",
							Description: "Show columns for a table.",
							HandlerName: "show.columns",
							Args: []*Arg{{
								Name:               "table",
								Description:        "Table name.",
								Required:           true,
								ValueType:          ValueTable,
								CompletionProvider: "table",
							}},
						},
						{
							Name:        "connections",
							UsageLine:   "show connections",
							Description: "Show saved connections.",
							HandlerName: "show.connections",
						},
						{
							Name:        "context",
							UsageLine:   "show context",
							Description: "Show current session context.",
							HandlerName: "show.context",
						},
						{
							Name:        "databases",
							UsageLine:   "show databases",
							Description: "Show databases on the selected connection.",
							HandlerName: "show.databases",
							Flags: []*Flag{
								{Name: "--template", Description: "Template name.", DefaultValue: "", ValueType: ValueString, CompletionProvider: "template"},
							},
						},
						{
							Name:        "users",
							UsageLine:   "show users",
							Description: "Show MySQL users.",
							HandlerName: "show.users",
						},
						{
							Name:        "rows",
							UsageLine:   "show rows <table> [--limit n]",
							Description: "Show rows from a table.",
							HandlerName: "show.rows",
							Args: []*Arg{{
								Name:               "table",
								Description:        "Table name.",
								Required:           true,
								ValueType:          ValueTable,
								CompletionProvider: "table",
							}},
							Flags: []*Flag{
								{Name: "--limit", Description: "Limit the number of rows returned.", DefaultValue: "10", ValueType: ValueString},
							},
						},
						{
							Name:        "table",
							UsageLine:   "show table <table>",
							Description: "Show CREATE TABLE output for one table.",
							HandlerName: "show.table",
							Args: []*Arg{{
								Name:               "table",
								Description:        "Table name.",
								Required:           true,
								ValueType:          ValueTable,
								CompletionProvider: "table",
							}},
						},
						{
							Name:        "tables",
							UsageLine:   "show tables",
							Description: "Show tables in the selected database.",
							HandlerName: "show.tables",
						},
						{
							Name:        "templates",
							UsageLine:   "show templates [query] [--tag value]",
							Description: "Show resolved workflow templates.",
							HandlerName: "show.templates",
							Args: []*Arg{{
								Name:               "query",
								Description:        "Optional template search query.",
								Required:           false,
								ValueType:          ValueString,
								CompletionProvider: "template",
							}},
							Flags: []*Flag{
								{Name: "--tag", Description: "Filter templates by tag.", DefaultValue: "", ValueType: ValueString, CompletionProvider: "template-tag"},
							},
						},
					},
				},
				{
					Name:        "create",
					UsageLine:   "create <subcommand>",
					Description: "Create database resources.",
					HandlerName: "create",
					Subcommands: []*Command{
						{
							Name:        "connection",
							UsageLine:   "create connection",
							Description: "Create a saved connection.",
							HandlerName: "create.connection",
						},
						{
							Name:        "database",
							UsageLine:   "create database",
							Description: "Create a database from a template.",
							HandlerName: "create.database",
							Flags: []*Flag{
								{Name: "--template", Description: "Template name.", DefaultValue: "", ValueType: ValueString, CompletionProvider: "template"},
								{Name: "--charset", Description: "Database charset.", DefaultValue: "utf8mb4", ValueType: ValueString},
								{Name: "--collation", Description: "Database collation.", DefaultValue: "utf8mb4_unicode_ci", ValueType: ValueString},
								{Name: "--if-not-exists", Description: "Use IF NOT EXISTS when supported by the template.", DefaultValue: "false", ValueType: ValueBool},
							},
						},
						{
							Name:        "user",
							UsageLine:   "create user [name]",
							Description: "Create a MySQL user.",
							HandlerName: "create.user",
							Args: []*Arg{{
								Name:               "name",
								Description:        "MySQL username.",
								Required:           false,
								ValueType:          ValueUser,
								CompletionProvider: "user",
							}},
							Flags: []*Flag{
								{Name: "--template", Description: "Template name.", DefaultValue: "", ValueType: ValueString, CompletionProvider: "template"},
								{Name: "--host", Description: "MySQL user host.", DefaultValue: "%", ValueType: ValueString},
								{Name: "--password", Description: "MySQL user password.", DefaultValue: "", ValueType: ValueString},
								{Name: "--password-env", Description: "Environment variable containing the MySQL user password.", DefaultValue: "", ValueType: ValueString},
								{Name: "--generate-password", Description: "Generate a password automatically.", DefaultValue: "false", ValueType: ValueBool},
								{Name: "--grant", Description: "Database grant mode.", DefaultValue: "", ValueType: ValueString, EnumValues: []string{"all", "readonly"}},
							},
						},
					},
				},
				{
					Name:        "drop",
					UsageLine:   "drop <subcommand>",
					Description: "Drop resources.",
					HandlerName: "drop",
					Subcommands: []*Command{
						{
							Name:        "connection",
							UsageLine:   "drop connection <name>",
							Description: "Drop a saved connection.",
							HandlerName: "drop.connection",
							Args: []*Arg{{
								Name:               "connection",
								Description:        "Saved connection name.",
								Required:           true,
								ValueType:          ValueConnection,
								CompletionProvider: "connection",
							}},
						},
						{
							Name:        "database",
							UsageLine:   "drop database",
							Description: "Drop a database from a template.",
							HandlerName: "drop.database",
							Flags: []*Flag{
								{Name: "--template", Description: "Template name.", DefaultValue: "", ValueType: ValueString, CompletionProvider: "template"},
							},
						},
						{
							Name:        "user",
							UsageLine:   "drop user [name]",
							Description: "Drop a MySQL user.",
							HandlerName: "drop.user",
							Args: []*Arg{{
								Name:               "name",
								Description:        "MySQL username.",
								Required:           false,
								ValueType:          ValueUser,
								CompletionProvider: "user",
							}},
							Flags: []*Flag{
								{Name: "--template", Description: "Template name.", DefaultValue: "", ValueType: ValueString, CompletionProvider: "template"},
								{Name: "--host", Description: "MySQL user host.", DefaultValue: "%", ValueType: ValueString},
							},
						},
					},
				},
				{
					Name:        "doctor",
					UsageLine:   "doctor",
					Description: "Inspect the selected connection statically.",
					HandlerName: "doctor",
				},
				{
					Name:        "audit",
					UsageLine:   "audit <subcommand>",
					Description: "Audit and history commands.",
					HandlerName: "audit",
					Subcommands: []*Command{
						{
							Name:        "log",
							UsageLine:   "audit log",
							Description: "Show audit history.",
							HandlerName: "audit.log",
						},
					},
				},
				{
					Name:        "exit",
					UsageLine:   "exit",
					Aliases:     []string{"quit", "q"},
					Description: "Exit the REPL.",
					HandlerName: "exit",
				},
			},
		}
	})
	return defaultManifestData
}
