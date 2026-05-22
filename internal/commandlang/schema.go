package commandlang

import (
	"fmt"
	"strings"
	"sync"

	"pkg.gostartkit.com/dbx/internal/commandmeta"
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
		defaultRegistryData = registryFromManifest(commandmeta.DefaultManifest())
	})
	return defaultRegistryData
}

func registryFromManifest(manifest *commandmeta.Manifest) *Registry {
	if manifest == nil {
		return &Registry{}
	}
	commands := make([]*CommandSpec, 0, len(manifest.Commands))
	for _, command := range manifest.Commands {
		commands = append(commands, commandSpecFromManifest(command))
	}
	return &Registry{Commands: commands}
}

func commandSpecFromManifest(command *commandmeta.Command) *CommandSpec {
	if command == nil {
		return nil
	}
	spec := &CommandSpec{
		Name:        command.Name,
		Aliases:     append([]string(nil), command.Aliases...),
		Description: command.Description,
		HandlerName: command.HandlerName,
		Hidden:      command.Hidden,
		Flags:       make([]*FlagSpec, 0, len(command.Flags)),
		Args:        make([]*ArgSpec, 0, len(command.Args)),
		Subcommands: make([]*CommandSpec, 0, len(command.Subcommands)),
	}
	for _, flag := range command.Flags {
		spec.Flags = append(spec.Flags, flagSpecFromManifest(flag))
	}
	for _, arg := range command.Args {
		spec.Args = append(spec.Args, argSpecFromManifest(arg))
	}
	for _, sub := range command.Subcommands {
		spec.Subcommands = append(spec.Subcommands, commandSpecFromManifest(sub))
	}
	return spec
}

func flagSpecFromManifest(flag *commandmeta.Flag) *FlagSpec {
	if flag == nil {
		return nil
	}
	return &FlagSpec{
		Name:               flag.Name,
		Short:              flag.Short,
		Description:        flag.Description,
		ValueType:          valueTypeFromManifest(flag.ValueType),
		Required:           flag.Required,
		Repeatable:         flag.Repeatable,
		EnumValues:         append([]string(nil), flag.EnumValues...),
		CompletionProvider: flag.CompletionProvider,
	}
}

func argSpecFromManifest(arg *commandmeta.Arg) *ArgSpec {
	if arg == nil {
		return nil
	}
	return &ArgSpec{
		Name:               arg.Name,
		Description:        arg.Description,
		Required:           arg.Required,
		Repeatable:         arg.Repeatable,
		ValueType:          valueTypeFromManifest(arg.ValueType),
		EnumValues:         append([]string(nil), arg.EnumValues...),
		CompletionProvider: arg.CompletionProvider,
	}
}

func valueTypeFromManifest(value commandmeta.ValueType) ValueType {
	return ValueType(value)
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
