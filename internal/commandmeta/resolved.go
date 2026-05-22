package commandmeta

import "strings"

type ResolvedCommand struct {
	Path          []string
	CanonicalPath []string
	Command       *Command
	Alias         bool
	Hidden        bool
}

func FlattenCommands(manifest *Manifest) []ResolvedCommand {
	if manifest == nil {
		return nil
	}
	commands := make([]ResolvedCommand, 0, len(manifest.Commands))
	for _, command := range manifest.Commands {
		appendResolvedCommands(&commands, nil, false, command)
	}
	return commands
}

func LookupCommandPath(manifest *Manifest, path string) (ResolvedCommand, bool) {
	tokens := strings.Fields(strings.TrimSpace(path))
	if len(tokens) == 0 {
		return ResolvedCommand{}, false
	}
	for _, command := range FlattenCommands(manifest) {
		if len(command.Path) != len(tokens) {
			continue
		}
		matched := true
		for index, token := range tokens {
			if command.Path[index] != token {
				matched = false
				break
			}
		}
		if matched {
			return command, true
		}
	}
	return ResolvedCommand{}, false
}

func appendResolvedCommands(dst *[]ResolvedCommand, parent []string, inheritedHidden bool, command *Command) {
	if command == nil {
		return
	}
	hidden := inheritedHidden || command.Hidden
	canonicalPath := append(append([]string(nil), parent...), command.Name)
	*dst = append(*dst, ResolvedCommand{
		Path:          append([]string(nil), canonicalPath...),
		CanonicalPath: append([]string(nil), canonicalPath...),
		Command:       command,
		Hidden:        hidden,
	})

	for _, alias := range command.Aliases {
		aliasTokens := strings.Fields(strings.TrimSpace(alias))
		if len(aliasTokens) == 0 {
			continue
		}
		aliasPath := append(append([]string(nil), parent...), aliasTokens...)
		*dst = append(*dst, ResolvedCommand{
			Path:          aliasPath,
			CanonicalPath: append([]string(nil), canonicalPath...),
			Command:       command,
			Alias:         true,
			Hidden:        hidden,
		})
	}

	for _, subcommand := range command.Subcommands {
		appendResolvedCommands(dst, canonicalPath, hidden, subcommand)
	}
}
