package app

import (
	"slices"
	"strings"
	"sync"

	"pkg.gostartkit.com/dbx/internal/commandmeta"
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
		specs := make([]CommandSpec, 0, 64)
		for _, command := range commandmeta.FlattenCommands(commandmeta.DefaultManifest()) {
			path := normalizeHelpTopic(strings.Join(command.Path, " "))
			specs = append(specs, CommandSpec{
				Path:        path,
				UsageLine:   command.Command.UsageLine,
				Description: command.Command.Description,
				Category:    commandCategory(command.Alias),
				Hidden:      command.Hidden,
			})
		}

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

func commandCategory(alias bool) string {
	if alias {
		return "alias"
	}
	return "command"
}
