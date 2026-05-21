package commandlang

import (
	"fmt"
	"strings"
	"sync"
)

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

type CommandSpec struct {
	Name        string
	Aliases     []string
	Description string
	Subcommands []*CommandSpec
	Flags       []*FlagSpec
	Args        []*ArgSpec
	HandlerName string
	Hidden      bool
}

type FlagSpec struct {
	Name               string
	Short              string
	Description        string
	ValueType          ValueType
	Required           bool
	Repeatable         bool
	EnumValues         []string
	CompletionProvider string
}

type ArgSpec struct {
	Name               string
	Description        string
	Required           bool
	Repeatable         bool
	ValueType          ValueType
	EnumValues         []string
	CompletionProvider string
}

type Registry struct {
	Commands []*CommandSpec
}

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type ValidationError struct {
	Message  string
	Range    Range
	Severity Severity
}

type AnnotatedProgram struct {
	Program  *Program
	RootSpec *Registry
	Commands []*AnnotatedCommand
	Errors   []ValidationError
}

type AnnotatedCommand struct {
	Node          *CommandNode
	Spec          *CommandSpec
	MatchedPath   []string
	Args          []*AnnotatedArg
	Flags         []*AnnotatedFlag
	UnknownTokens []string
}

type AnnotatedArg struct {
	Node  *ArgNode
	Spec  *ArgSpec
	Index int
}

type AnnotatedFlag struct {
	Node *FlagNode
	Spec *FlagSpec
}

type Handler interface{}

type HandlerRegistry interface {
	Get(name string) Handler
}

type HelpDoc struct {
	Title string
	Body  string
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistryData *Registry
)

func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistryData = &Registry{
			Commands: []*CommandSpec{
				{
					Name:        "exec",
					Description: "Execute a named operation.",
					HandlerName: "exec",
					Args: []*ArgSpec{{
						Name:               "operation",
						Description:        "Operation name to execute.",
						Required:           true,
						ValueType:          ValueOperation,
						CompletionProvider: "operation",
					}},
					Flags: []*FlagSpec{
						{Name: "--dry-run", Description: "Render and validate without applying changes.", ValueType: ValueBool},
						{Name: "--validate", Description: "Validate the resolved operation and exit.", ValueType: ValueBool},
						{Name: "--yes", Description: "Skip confirmation prompts.", ValueType: ValueBool},
						{Name: "--preview", Description: "Show the execution preview before running.", ValueType: ValueBool},
						{Name: "--verbose", Description: "Include detailed execution output.", ValueType: ValueBool},
					},
				},
				{
					Name:        "help",
					Description: "Show help for a command or topic.",
					HandlerName: "help",
					Args: []*ArgSpec{{
						Name:               "topic",
						Description:        "Command or topic to describe.",
						Required:           false,
						ValueType:          ValueString,
						CompletionProvider: "topic",
					}},
				},
				{
					Name:        "template",
					Description: "Template maintenance and rendering commands.",
					HandlerName: "template",
					Hidden:      true,
					Subcommands: []*CommandSpec{
						{
							Name:        "list",
							Description: "List available templates.",
							HandlerName: "template.list",
						},
						{
							Name:        "show",
							Description: "Show template details.",
							HandlerName: "template.show",
							Args: []*ArgSpec{{
								Name:               "template",
								Description:        "Template name.",
								Required:           true,
								ValueType:          ValueTemplate,
								CompletionProvider: "template",
							}},
						},
						{
							Name:        "render",
							Description: "Render a template preview.",
							HandlerName: "template.render",
							Args: []*ArgSpec{{
								Name:               "template",
								Description:        "Template name.",
								Required:           true,
								ValueType:          ValueTemplate,
								CompletionProvider: "template",
							}},
							Flags: []*FlagSpec{
								{Name: "--var", Description: "Template input override in key=value form.", ValueType: ValueString},
							},
						},
					},
				},
				{
					Name:        "connect",
					Description: "Connect to a saved connection.",
					HandlerName: "connect",
					Args: []*ArgSpec{{
						Name:               "connection",
						Description:        "Saved connection name.",
						Required:           true,
						ValueType:          ValueConnection,
						CompletionProvider: "connection",
					}},
				},
				{
					Name:        "connection",
					Description: "Connection management commands.",
					HandlerName: "connection",
					Hidden:      true,
					Subcommands: []*CommandSpec{
						{
							Name:        "list",
							Description: "List saved connections.",
							HandlerName: "connection.list",
						},
						{
							Name:        "use",
							Description: "Select a saved connection.",
							HandlerName: "connection.use",
							Args: []*ArgSpec{{
								Name:               "connection",
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
					Description: "Database selection commands.",
					HandlerName: "database",
					Hidden:      true,
					Subcommands: []*CommandSpec{
						{
							Name:        "use",
							Description: "Select the current database.",
							HandlerName: "database.use",
							Args: []*ArgSpec{{
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
					Description: "Select the current database.",
					HandlerName: "use",
					Args: []*ArgSpec{{
						Name:               "database",
						Description:        "Database name.",
						Required:           true,
						ValueType:          ValueDatabase,
						CompletionProvider: "database",
					}},
				},
				{
					Name:        "show",
					Description: "Inspect configuration and database state.",
					HandlerName: "show",
					Subcommands: []*CommandSpec{
						{
							Name:        "connection",
							Description: "Show a saved connection.",
							HandlerName: "show.connection",
							Args: []*ArgSpec{{
								Name:               "connection",
								Description:        "Saved connection name.",
								Required:           true,
								ValueType:          ValueConnection,
								CompletionProvider: "connection",
							}},
						},
						{
							Name:        "columns",
							Description: "Show columns for a table.",
							HandlerName: "show.columns",
							Args: []*ArgSpec{{
								Name:               "table",
								Description:        "Table name.",
								Required:           true,
								ValueType:          ValueTable,
								CompletionProvider: "table",
							}},
						},
						{
							Name:        "connections",
							Description: "Show saved connections.",
							HandlerName: "show.connections",
						},
						{
							Name:        "context",
							Description: "Show current session context.",
							HandlerName: "show.context",
						},
						{
							Name:        "databases",
							Description: "Show databases on the selected connection.",
							HandlerName: "show.databases",
						},
						{
							Name:        "rows",
							Description: "Show rows from a table.",
							HandlerName: "show.rows",
							Args: []*ArgSpec{{
								Name:               "table",
								Description:        "Table name.",
								Required:           true,
								ValueType:          ValueTable,
								CompletionProvider: "table",
							}},
						},
						{
							Name:        "table",
							Description: "Show CREATE TABLE output for one table.",
							HandlerName: "show.table",
							Args: []*ArgSpec{{
								Name:               "table",
								Description:        "Table name.",
								Required:           true,
								ValueType:          ValueTable,
								CompletionProvider: "table",
							}},
						},
						{
							Name:        "tables",
							Description: "Show tables in the selected database.",
							HandlerName: "show.tables",
						},
						{
							Name:        "templates",
							Description: "Show resolved workflow templates.",
							HandlerName: "show.templates",
							Args: []*ArgSpec{{
								Name:               "query",
								Description:        "Optional template search query.",
								Required:           false,
								ValueType:          ValueString,
								CompletionProvider: "template",
							}},
							Flags: []*FlagSpec{
								{Name: "--tag", Description: "Filter templates by tag.", ValueType: ValueString, CompletionProvider: "template-tag"},
							},
						},
					},
				},
				{
					Name:        "exit",
					Aliases:     []string{"quit", "q"},
					Description: "Exit the REPL.",
					HandlerName: "exit",
				},
			},
		}
	})
	return defaultRegistryData
}

func (r *Registry) LookupCommand(path []string) (*CommandSpec, int) {
	if r == nil {
		return nil, 0
	}
	var best *CommandSpec
	bestLen := 0
	for _, command := range r.Commands {
		spec, matched := lookupCommandSpec(command, path, 0)
		if spec != nil && matched > bestLen {
			best = spec
			bestLen = matched
		}
	}
	return best, bestLen
}

func (r *Registry) LookupVisibleSubcommands(path []string) []*CommandSpec {
	if r == nil {
		return nil
	}
	if len(path) == 0 {
		return filterVisibleCommands(r.Commands)
	}
	spec, matched := r.LookupCommand(path)
	if spec == nil || matched != len(path) {
		return nil
	}
	return filterVisibleCommands(spec.Subcommands)
}

func (r *Registry) KnownPaths(includeHidden bool) [][]string {
	if r == nil {
		return nil
	}
	paths := make([][]string, 0)
	for _, command := range r.Commands {
		collectPaths(&paths, nil, command, includeHidden)
	}
	return paths
}

func (r *Registry) LookupTopic(topic string) (*CommandSpec, []string) {
	tokens := strings.Fields(strings.TrimSpace(topic))
	if len(tokens) == 0 {
		return nil, nil
	}
	spec, matched := r.LookupCommand(tokens)
	if spec == nil || matched != len(tokens) {
		return nil, nil
	}
	return spec, tokens
}

func (r *Registry) AnnotateProgram(program *Program) *AnnotatedProgram {
	annotated := &AnnotatedProgram{
		Program:  program,
		RootSpec: r,
		Commands: make([]*AnnotatedCommand, 0),
		Errors:   make([]ValidationError, 0),
	}
	if program == nil {
		return annotated
	}
	for _, command := range program.Commands {
		entry, errors := r.annotateCommand(command)
		annotated.Commands = append(annotated.Commands, entry)
		annotated.Errors = append(annotated.Errors, errors...)
	}
	return annotated
}

func (r *Registry) ValidateProgram(program *Program) []ValidationError {
	return r.AnnotateProgram(program).Errors
}

func (r *Registry) Help(topic string) (HelpDoc, bool) {
	spec, path := r.LookupTopic(topic)
	if spec == nil {
		return HelpDoc{}, false
	}
	return buildHelpDoc(spec, path), true
}

func (s *CommandSpec) FindFlag(name string) *FlagSpec {
	if s == nil {
		return nil
	}
	for _, flag := range s.Flags {
		if flag.Name == name || (flag.Short != "" && flag.Short == name) {
			return flag
		}
	}
	return nil
}

func (s *CommandSpec) ArgSpec(index int) *ArgSpec {
	if s == nil || index < 0 || len(s.Args) == 0 {
		return nil
	}
	if index < len(s.Args) {
		return s.Args[index]
	}
	last := s.Args[len(s.Args)-1]
	if last.Repeatable {
		return last
	}
	return nil
}

func (s *CommandSpec) HasBoolFlag(name string) bool {
	flag := s.FindFlag(name)
	return flag != nil && flag.ValueType == ValueBool
}

func lookupCommandSpec(spec *CommandSpec, path []string, depth int) (*CommandSpec, int) {
	if spec == nil || depth >= len(path) {
		return nil, 0
	}
	if !matchesCommandToken(spec, path[depth]) {
		return nil, 0
	}
	best := spec
	bestLen := depth + 1
	for _, sub := range spec.Subcommands {
		if candidate, matched := lookupCommandSpec(sub, path, depth+1); candidate != nil && matched > bestLen {
			best = candidate
			bestLen = matched
		}
	}
	return best, bestLen
}

func matchesCommandToken(spec *CommandSpec, token string) bool {
	if spec == nil {
		return false
	}
	if spec.Name == token {
		return true
	}
	for _, alias := range spec.Aliases {
		if alias == token {
			return true
		}
	}
	return false
}

func filterVisibleCommands(commands []*CommandSpec) []*CommandSpec {
	visible := make([]*CommandSpec, 0, len(commands))
	for _, command := range commands {
		if command != nil && !command.Hidden {
			visible = append(visible, command)
		}
	}
	return visible
}

func collectPaths(dst *[][]string, prefix []string, spec *CommandSpec, includeHidden bool) {
	if spec == nil {
		return
	}
	path := append(append([]string(nil), prefix...), spec.Name)
	if includeHidden || !spec.Hidden {
		*dst = append(*dst, path)
	}
	for _, sub := range spec.Subcommands {
		collectPaths(dst, path, sub, includeHidden)
	}
}

func (r *Registry) annotateCommand(command *CommandNode) (*AnnotatedCommand, []ValidationError) {
	annotateCommandNode(command, r.KnownPaths(true))
	result := &AnnotatedCommand{
		Node:        command,
		Args:        make([]*AnnotatedArg, 0),
		Flags:       make([]*AnnotatedFlag, 0),
		MatchedPath: append([]string(nil), commandPath(command)...),
	}
	if command == nil {
		return result, nil
	}

	path := commandPath(command)
	spec, matched := r.LookupCommand(path)
	if matched == 0 {
		return result, []ValidationError{{
			Message:  fmt.Sprintf("unknown command %q", firstValue(path)),
			Range:    command.Range(),
			Severity: SeverityError,
		}}
	}
	result.Spec = spec
	if matched < len(path) {
		unknown := path[matched]
		rangeValue := command.Range()
		if len(command.Positionals) > matched {
			rangeValue = command.Positionals[matched].Range()
		}
		return result, []ValidationError{{
			Message:  fmt.Sprintf("unknown subcommand %q", unknown),
			Range:    rangeValue,
			Severity: SeverityError,
		}}
	}
	rawPositionals := positionalsToValues(command.Positionals)
	if matched < len(rawPositionals) && spec != nil && len(spec.Subcommands) > 0 && len(spec.Args) == 0 {
		rangeValue := command.Range()
		if len(command.Positionals) > matched {
			rangeValue = command.Positionals[matched].Range()
		}
		return result, []ValidationError{{
			Message:  fmt.Sprintf("unknown subcommand %q", rawPositionals[matched]),
			Range:    rangeValue,
			Severity: SeverityError,
		}}
	}

	errors := make([]ValidationError, 0)
	for index, arg := range command.Args {
		specArg := spec.ArgSpec(index)
		result.Args = append(result.Args, &AnnotatedArg{Node: arg, Spec: specArg, Index: index})
		if specArg == nil {
			errors = append(errors, ValidationError{
				Message:  "too many positional args",
				Range:    arg.Range(),
				Severity: SeverityError,
			})
			continue
		}
		if specArg.ValueType == ValueEnum && len(specArg.EnumValues) > 0 && !containsString(specArg.EnumValues, arg.Value) {
			errors = append(errors, ValidationError{
				Message:  fmt.Sprintf("invalid value %q for %s", arg.Value, specArg.Name),
				Range:    arg.Range(),
				Severity: SeverityError,
			})
		}
	}
	for index := 0; index < len(spec.Args); index++ {
		argSpec := spec.Args[index]
		if argSpec.Required && index >= len(command.Args) {
			errors = append(errors, ValidationError{
				Message:  fmt.Sprintf("missing required arg %q", argSpec.Name),
				Range:    command.Range(),
				Severity: SeverityError,
			})
		}
	}

	seenFlags := make(map[string]int)
	for _, flag := range command.Flags {
		flagSpec := spec.FindFlag(flag.Name)
		result.Flags = append(result.Flags, &AnnotatedFlag{Node: flag, Spec: flagSpec})
		if flagSpec == nil {
			errors = append(errors, ValidationError{
				Message:  fmt.Sprintf("unknown flag %q", flag.Name),
				Range:    flag.Range(),
				Severity: SeverityError,
			})
			continue
		}
		seenFlags[flagSpec.Name]++
		if !flagSpec.Repeatable && seenFlags[flagSpec.Name] > 1 {
			errors = append(errors, ValidationError{
				Message:  fmt.Sprintf("flag %q is not repeatable", flag.Name),
				Range:    flag.Range(),
				Severity: SeverityError,
			})
		}
		if flagSpec.ValueType == ValueBool {
			if flag.HasValue {
				errors = append(errors, ValidationError{
					Message:  fmt.Sprintf("flag %q does not take a value", flag.Name),
					Range:    flag.Range(),
					Severity: SeverityError,
				})
			}
			continue
		}
		if !flag.HasValue {
			errors = append(errors, ValidationError{
				Message:  fmt.Sprintf("missing flag value for %q", flag.Name),
				Range:    flag.Range(),
				Severity: SeverityError,
			})
			continue
		}
		if flagSpec.ValueType == ValueEnum && flag.Value != nil && len(flagSpec.EnumValues) > 0 && !containsString(flagSpec.EnumValues, flag.Value.Value) {
			errors = append(errors, ValidationError{
				Message:  fmt.Sprintf("invalid value %q for %s", flag.Value.Value, flag.Name),
				Range:    flag.Value.Range(),
				Severity: SeverityError,
			})
		}
	}
	for _, flag := range spec.Flags {
		if flag.Required && seenFlags[flag.Name] == 0 {
			errors = append(errors, ValidationError{
				Message:  fmt.Sprintf("missing required flag %q", flag.Name),
				Range:    command.Range(),
				Severity: SeverityError,
			})
		}
	}
	return result, errors
}

func buildHelpDoc(spec *CommandSpec, path []string) HelpDoc {
	title := strings.Join(path, " ")
	if title == "" && spec != nil {
		title = spec.Name
	}
	lines := make([]string, 0, 16)
	if spec.Description != "" {
		lines = append(lines, spec.Description)
	}
	usage := "Usage:\n  dbx " + buildUsagePath(path, spec)
	lines = append(lines, usage)
	if len(spec.Args) > 0 {
		lines = append(lines, "Args:")
		for _, arg := range spec.Args {
			line := fmt.Sprintf("  <%s>", arg.Name)
			if arg.Description != "" {
				line += "  " + arg.Description
			}
			lines = append(lines, line)
		}
	}
	if len(spec.Flags) > 0 {
		lines = append(lines, "Flags:")
		for _, flag := range spec.Flags {
			line := "  " + flag.Name
			if flag.ValueType != ValueBool {
				line += " <value>"
			}
			if flag.Description != "" {
				line += "  " + flag.Description
			}
			lines = append(lines, line)
		}
	}
	visibleSubs := filterVisibleCommands(spec.Subcommands)
	if len(visibleSubs) > 0 {
		lines = append(lines, "Subcommands:")
		for _, sub := range visibleSubs {
			line := "  " + sub.Name
			if sub.Description != "" {
				line += "  " + sub.Description
			}
			lines = append(lines, line)
		}
	}
	return HelpDoc{
		Title: title,
		Body:  strings.Join(lines, "\n"),
	}
}

func buildUsagePath(path []string, spec *CommandSpec) string {
	parts := append([]string(nil), path...)
	for _, arg := range spec.Args {
		part := "<" + arg.Name + ">"
		if !arg.Required {
			part = "[" + part + "]"
		}
		if arg.Repeatable {
			part += "..."
		}
		parts = append(parts, part)
	}
	if len(spec.Flags) > 0 {
		parts = append(parts, "[flags]")
	}
	if len(parts) == 0 && spec != nil {
		parts = append(parts, spec.Name)
	}
	return strings.Join(parts, " ")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
