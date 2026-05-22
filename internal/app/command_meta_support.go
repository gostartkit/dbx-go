package app

import (
	"context"
	"strconv"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/commandmeta"
)

type manifestCommandOptions struct {
	Path                string
	Name                string
	UsageFallback       string
	PreferFallbackUsage bool
	ShortFallback       string
	Long                string
	Aliases             []string
	Positionals         []cmd.PositionalArg
	SubCommands         []*cmd.Command
	SetFlags            func(*cmd.FlagSet)
	Run                 func(ctx context.Context, cmd *cmd.Command, args []string) error
}

func manifestCommand(path string) (commandmeta.ResolvedCommand, bool) {
	return commandmeta.LookupCommandPath(commandmeta.DefaultManifest(), normalizeHelpTopic(path))
}

func manifestShort(path string, fallback string) string {
	command, ok := manifestCommand(path)
	if !ok || command.Command == nil {
		return fallback
	}
	if description := strings.TrimSpace(command.Command.Description); description != "" {
		return description
	}
	return fallback
}

func manifestUsageLine(path string, fallback string) string {
	command, ok := manifestCommand(path)
	if !ok || command.Command == nil {
		return fallback
	}
	if usage := strings.TrimSpace(command.Command.UsageLine); usage != "" {
		return "dbx " + usage
	}
	return fallback
}

func manifestAliases(path string, fallback []string) []string {
	command, ok := manifestCommand(path)
	if !ok || command.Command == nil || len(command.Command.Aliases) == 0 {
		return append([]string(nil), fallback...)
	}
	return append([]string(nil), command.Command.Aliases...)
}

func manifestFlag(path string, name string) (*commandmeta.Flag, bool) {
	command, ok := manifestCommand(path)
	if !ok || command.Command == nil {
		return nil, false
	}
	target := normalizeManifestFlagName(name)
	for _, flag := range command.Command.Flags {
		if flag == nil {
			continue
		}
		if normalizeManifestFlagName(flag.Name) == target {
			return flag, true
		}
	}
	return nil, false
}

func (b *cliBuilder) newManifestCommand(options manifestCommandOptions) *cmd.Command {
	path := normalizeHelpTopic(options.Path)
	command := &cmd.Command{
		Name:        options.Name,
		UsageLine:   options.UsageFallback,
		Short:       manifestShort(path, options.ShortFallback),
		Long:        options.Long,
		Aliases:     append([]string(nil), options.Aliases...),
		Positionals: append([]cmd.PositionalArg(nil), options.Positionals...),
		SubCommands: append([]*cmd.Command(nil), options.SubCommands...),
		SetFlags:    options.SetFlags,
		Run:         options.Run,
	}

	if command.Name == "" {
		resolved, ok := manifestCommand(path)
		if ok && resolved.Command != nil {
			command.Name = resolved.Command.Name
		}
	}
	if command.Name == "" {
		tokens := strings.Fields(path)
		if len(tokens) > 0 {
			command.Name = tokens[len(tokens)-1]
		}
	}
	if !options.PreferFallbackUsage {
		command.UsageLine = manifestUsageLine(path, options.UsageFallback)
	}
	if len(command.Aliases) == 0 {
		command.Aliases = manifestAliases(path, nil)
	}
	if len(command.Positionals) == 0 {
		command.Positionals = b.manifestPositionals(path, nil)
	}
	if command.Long == "" {
		command.Long = helpLong(path)
	}
	return command
}

func (b *cliBuilder) manifestPositionals(path string, fallback []cmd.PositionalArg) []cmd.PositionalArg {
	command, ok := manifestCommand(path)
	if !ok || command.Command == nil || len(command.Command.Args) == 0 {
		return append([]cmd.PositionalArg(nil), fallback...)
	}

	positionals := make([]cmd.PositionalArg, 0, len(command.Command.Args))
	for _, arg := range command.Command.Args {
		if arg == nil {
			continue
		}
		positionals = append(positionals, cmd.PositionalArg{
			Name:       arg.Name,
			Usage:      arg.Description,
			Required:   arg.Required,
			Variadic:   arg.Repeatable,
			Enum:       append([]string(nil), arg.EnumValues...),
			Completion: b.manifestCompletion(arg.CompletionProvider),
		})
	}
	if len(positionals) == 0 {
		return append([]cmd.PositionalArg(nil), fallback...)
	}
	return positionals
}

func (b *cliBuilder) manifestCompletion(name string) cmd.CompletionFunc {
	switch strings.TrimSpace(name) {
	case "connection":
		return b.completeConnections
	case "database":
		return b.completeDatabases
	case "table":
		return b.completeTables
	case "user":
		return b.completeUsers
	case "template":
		return b.completeTemplates
	case "template-tag":
		return b.completeTemplateTags
	case "operation":
		return b.completeOperations
	case "topic":
		return b.completeHelpTopics
	default:
		return nil
	}
}

func (b *cliBuilder) bindManifestStringFlag(f *cmd.FlagSet, path string, name string, target *string, fallbackDefault string, fallbackUsage string) {
	flag, ok := manifestFlag(path, name)
	if !ok || flag == nil {
		f.StringVar(target, normalizeManifestFlagName(name), fallbackDefault, fallbackUsage, "")
		return
	}
	defaultValue := fallbackDefault
	if short := normalizeManifestFlagShort(flag.Short); short != "" {
		f.StringVar(target, normalizeManifestFlagName(flag.Name), defaultValue, flag.Description, short)
	} else {
		f.StringVar(target, normalizeManifestFlagName(flag.Name), defaultValue, flag.Description, "")
	}
	if len(flag.EnumValues) > 0 {
		f.SetEnum(normalizeManifestFlagName(flag.Name), flag.EnumValues...)
	}
	if completion := b.manifestCompletion(flag.CompletionProvider); completion != nil {
		f.SetCompletion(normalizeManifestFlagName(flag.Name), completion)
	}
}

func (b *cliBuilder) bindManifestIntFlag(f *cmd.FlagSet, path string, name string, target *int, fallbackDefault int, fallbackUsage string) {
	flag, ok := manifestFlag(path, name)
	if !ok || flag == nil {
		f.IntVar(target, normalizeManifestFlagName(name), fallbackDefault, fallbackUsage, "")
		return
	}
	if short := normalizeManifestFlagShort(flag.Short); short != "" {
		f.IntVar(target, normalizeManifestFlagName(flag.Name), fallbackDefault, flag.Description, short)
	} else {
		f.IntVar(target, normalizeManifestFlagName(flag.Name), fallbackDefault, flag.Description, "")
	}
}

func (b *cliBuilder) bindManifestBoolFlag(f *cmd.FlagSet, path string, name string, target *bool, fallbackDefault bool, fallbackUsage string) {
	flag, ok := manifestFlag(path, name)
	if !ok || flag == nil {
		f.BoolVar(target, normalizeManifestFlagName(name), fallbackDefault, fallbackUsage, "")
		return
	}
	if short := normalizeManifestFlagShort(flag.Short); short != "" {
		f.BoolVar(target, normalizeManifestFlagName(flag.Name), fallbackDefault, flag.Description, short)
	} else {
		f.BoolVar(target, normalizeManifestFlagName(flag.Name), fallbackDefault, flag.Description, "")
	}
}

func normalizeManifestFlagName(name string) string {
	return strings.TrimLeft(strings.TrimSpace(name), "-")
}

func normalizeManifestFlagShort(short string) string {
	trimmed := strings.TrimLeft(strings.TrimSpace(short), "-")
	if len(trimmed) > 1 {
		return ""
	}
	return trimmed
}

func manifestFlagDefaultInt(path string, name string, fallback int) int {
	flag, ok := manifestFlag(path, name)
	if !ok || flag == nil {
		return fallback
	}
	if value, ok := flagDefaultInt(flag); ok {
		return value
	}
	return fallback
}

func manifestFlagDefaultString(path string, name string, fallback string) string {
	flag, ok := manifestFlag(path, name)
	if !ok || flag == nil {
		return fallback
	}
	defaultValue := strings.TrimSpace(flag.DefaultValue)
	if defaultValue == "" {
		return fallback
	}
	return defaultValue
}

func flagDefaultInt(flag *commandmeta.Flag) (int, bool) {
	if flag == nil {
		return 0, false
	}
	defaultValue := strings.TrimSpace(flag.DefaultValue)
	if defaultValue == "" {
		return 0, false
	}
	value, err := strconv.Atoi(defaultValue)
	if err != nil {
		return 0, false
	}
	return value, true
}
