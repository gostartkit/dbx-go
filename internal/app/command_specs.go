package app

import (
	"io"
	"slices"
	"strings"

	"pkg.gostartkit.com/cmd"
)

type CommandSpec struct {
	Path        string
	UsageLine   string
	Description string
	Category    string
	Hidden      bool
}

func replCommandSpecs() []CommandSpec {
	app := (&cliBuilder{
		mode:    ModeREPL,
		out:     io.Discard,
		err:     io.Discard,
		globals: &cliGlobals{Format: "text"},
	}).buildApp()

	specs := make([]CommandSpec, 0)
	for _, command := range app.Spec().Commands {
		flattenCommandSpec(&specs, "", command, false)
	}
	return specs
}

func flattenCommandSpec(dst *[]CommandSpec, prefix string, spec cmd.CommandSpec, aliased bool) {
	path := strings.TrimSpace(strings.Join([]string{prefix, spec.Name}, " "))
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
			Path:        strings.TrimSpace(strings.Join([]string{prefix, alias}, " ")),
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
	normalized := normalizeHelpTopic(path)
	for _, spec := range replCommandSpecs() {
		if normalizeHelpTopic(spec.Path) == normalized {
			return spec, true
		}
	}
	return CommandSpec{}, false
}

func commandSpecForInput(line string) (CommandSpec, bool) {
	tokens := strings.Fields(strings.TrimSpace(line))
	if len(tokens) == 0 {
		return CommandSpec{}, false
	}

	best := CommandSpec{}
	bestLen := 0
	for _, spec := range replCommandSpecs() {
		pathTokens := strings.Fields(spec.Path)
		if len(pathTokens) == 0 || len(pathTokens) > len(tokens) {
			continue
		}
		if !slices.Equal(pathTokens, tokens[:len(pathTokens)]) {
			continue
		}
		if len(pathTokens) > bestLen {
			best = spec
			bestLen = len(pathTokens)
		}
	}
	if bestLen == 0 {
		return CommandSpec{}, false
	}
	return best, true
}

func helpCompletionTopics() []Suggestion {
	suggestions := make([]Suggestion, 0)
	seen := map[string]struct{}{}
	add := func(value string, description string) {
		value = normalizeHelpTopic(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		suggestions = append(suggestions, Suggestion{
			Value:       value,
			Description: description,
			Category:    "topic",
		})
	}

	for _, spec := range replCommandSpecs() {
		if spec.Hidden {
			continue
		}
		add(spec.Path, spec.Description)
	}
	return suggestions
}
