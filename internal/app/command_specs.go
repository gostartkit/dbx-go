package app

import (
	"io"
	"strings"

	"pkg.gostartkit.com/cmd"
)

type CommandSpec struct {
	Path        string
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
	specs = append(specs, CommandSpec{Path: "help", Description: "show command help", Category: "builtin"})
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
		Description: spec.Short,
		Category:    category,
		Hidden:      spec.Hidden,
	})

	for _, alias := range spec.Aliases {
		*dst = append(*dst, CommandSpec{
			Path:        strings.TrimSpace(strings.Join([]string{prefix, alias}, " ")),
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
