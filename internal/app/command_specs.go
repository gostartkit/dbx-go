package app

import (
	"slices"
	"strings"
	"sync"

	"pkg.gostartkit.com/cmd"
)

type CommandSpec struct {
	Path        string
	UsageLine   string
	Description string
	Category    string
	Hidden      bool
}

type indexedCommandSpec struct {
	spec   CommandSpec
	tokens []string
}

type commandSpecCatalog struct {
	specs           []CommandSpec
	indexed         []indexedCommandSpec
	byPath          map[string]CommandSpec
	helpSuggestions []Suggestion
}

var (
	replCommandSpecCatalogOnce sync.Once
	replCommandSpecCatalogData commandSpecCatalog
)

func replCommandSpecs() []CommandSpec {
	return replCommandSpecCatalog().specs
}

func commandSpecByPath(path string) (CommandSpec, bool) {
	spec, ok := replCommandSpecCatalog().byPath[normalizeHelpTopic(path)]
	return spec, ok
}

func commandSpecForInput(line string) (CommandSpec, bool) {
	tokens := strings.Fields(strings.TrimSpace(line))
	if len(tokens) == 0 {
		return CommandSpec{}, false
	}

	best := CommandSpec{}
	bestLen := 0
	for _, entry := range replCommandSpecCatalog().indexed {
		if len(entry.tokens) == 0 || len(entry.tokens) > len(tokens) {
			continue
		}
		if !slices.Equal(entry.tokens, tokens[:len(entry.tokens)]) {
			continue
		}
		if len(entry.tokens) > bestLen {
			best = entry.spec
			bestLen = len(entry.tokens)
		}
	}
	if bestLen == 0 {
		return CommandSpec{}, false
	}
	return best, true
}

func helpCompletionTopics() []Suggestion {
	return replCommandSpecCatalog().helpSuggestions
}

func replCommandSpecCatalog() *commandSpecCatalog {
	replCommandSpecCatalogOnce.Do(func() {
		specs := replSpecsFromCommandTree()

		indexed := make([]indexedCommandSpec, 0, len(specs))
		byPath := make(map[string]CommandSpec, len(specs))
		suggestions := make([]Suggestion, 0, len(specs))
		seenSuggestions := make(map[string]struct{}, len(specs))
		for _, spec := range specs {
			indexed = append(indexed, indexedCommandSpec{
				spec:   spec,
				tokens: strings.Fields(spec.Path),
			})

			normalizedPath := normalizeHelpTopic(spec.Path)
			if normalizedPath != "" {
				byPath[normalizedPath] = spec
			}
			if spec.Hidden || normalizedPath == "" {
				continue
			}
			if _, ok := seenSuggestions[normalizedPath]; ok {
				continue
			}
			seenSuggestions[normalizedPath] = struct{}{}
			suggestions = append(suggestions, Suggestion{
				Value:       normalizedPath,
				Description: spec.Description,
				Category:    "topic",
			})
		}

		replCommandSpecCatalogData = commandSpecCatalog{
			specs:           specs,
			indexed:         indexed,
			byPath:          byPath,
			helpSuggestions: suggestions,
		}
	})
	return &replCommandSpecCatalogData
}

func replSpecsFromCommandTree() []CommandSpec {
	spec := (&cliBuilder{
		mode:    ModeREPL,
		globals: &cliGlobals{Format: "text"},
	}).buildApp().SpecFor(cmd.SurfaceREPL)

	specs := make([]CommandSpec, 0, 64)
	for _, command := range spec.Commands {
		collectCommandSpecs(&specs, "", command)
	}
	return specs
}

func collectCommandSpecs(dst *[]CommandSpec, prefix string, command cmd.CommandSpec) {
	path := normalizeHelpTopic(strings.Join(command.Path, " "))
	if path != "" {
		*dst = append(*dst, CommandSpec{
			Path:        path,
			UsageLine:   command.UsageLine,
			Description: command.Short,
			Category:    "command",
			Hidden:      command.Hidden,
		})
	}

	for _, alias := range command.Aliases {
		aliasPath := normalizeHelpTopic(prefix + " " + alias)
		if aliasPath != "" {
			*dst = append(*dst, CommandSpec{
				Path:        aliasPath,
				UsageLine:   command.UsageLine,
				Description: command.Short,
				Category:    "alias",
				Hidden:      command.Hidden,
			})
		}
	}

	for _, subcommand := range command.SubCommands {
		nextPrefix := path
		if nextPrefix == "" {
			nextPrefix = normalizeHelpTopic(prefix + " " + command.Name)
		}
		collectCommandSpecs(dst, nextPrefix, subcommand)
	}
}
