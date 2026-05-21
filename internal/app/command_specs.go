package app

import (
	"io"
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

func flattenCommandSpec(dst *[]CommandSpec, prefix string, spec cmd.CommandSpec, aliased bool) {
	path := joinCommandPath(prefix, spec.Name)
	category := "command"
	if aliased {
		category = "alias"
	}
	*dst = append(*dst, CommandSpec{
		Path:        path,
		UsageLine:   spec.UsageLine,
		Description: spec.Short,
		Category:    category,
		Hidden:      spec.Hidden,
	})

	for _, alias := range spec.Aliases {
		*dst = append(*dst, CommandSpec{
			Path:        joinCommandPath(prefix, alias),
			UsageLine:   spec.UsageLine,
			Description: spec.Short,
			Category:    "alias",
			Hidden:      spec.Hidden,
		})
	}
	for _, subCommand := range spec.SubCommands {
		flattenCommandSpec(dst, path, subCommand, aliased)
	}
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
		app := (&cliBuilder{
			mode:    ModeREPL,
			out:     io.Discard,
			err:     io.Discard,
			globals: &cliGlobals{Format: "text"},
		}).buildApp()

		specs := make([]CommandSpec, 0, 64)
		for _, command := range app.Spec().Commands {
			flattenCommandSpec(&specs, "", command, false)
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

func joinCommandPath(prefix string, name string) string {
	switch {
	case prefix == "":
		return strings.TrimSpace(name)
	case name == "":
		return strings.TrimSpace(prefix)
	default:
		return prefix + " " + strings.TrimSpace(name)
	}
}
