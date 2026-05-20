package app

import (
	"strings"
	"testing"
)

func TestCommandSpecsCoverHelpRootCommandsAndAliases(t *testing.T) {
	t.Parallel()

	specs := replCommandSpecs()
	specSet := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		specSet[normalizeHelpTopic(spec.Path)] = struct{}{}
	}

	for _, path := range extractRootHelpCommandPaths(helpEntries[""].body) {
		if _, ok := specSet[path]; ok {
			continue
		}
		if _, ok := helpEntries[path]; ok {
			continue
		}
		if normalizeHelpTopic(path) == "help aliases" {
			continue
		}
		if _, ok := specSet[strings.TrimSpace(strings.TrimPrefix(path, "help "))]; ok && strings.HasPrefix(path, "help ") {
			continue
		}
		if _, ok := helpEntries[strings.TrimSpace(strings.TrimPrefix(path, "help "))]; ok && strings.HasPrefix(path, "help ") {
			continue
		}
		if _, ok := specSet[path]; !ok {
			t.Fatalf("help root command %q missing from command specs", path)
		}
	}

	for alias := range commandAliases {
		if _, ok := specSet[normalizeHelpTopic(alias)]; !ok {
			t.Fatalf("alias %q missing from command specs", alias)
		}
	}
}

func TestRootCompletionContainsAllCommandSpecs(t *testing.T) {
	t.Parallel()

	values := suggestionValues(calculateCompletion("", CompletionContext{}))
	have := make(map[string]struct{}, len(values))
	for _, value := range values {
		have[value] = struct{}{}
	}

	for _, spec := range replCommandSpecs() {
		if spec.Hidden {
			continue
		}
		if _, ok := have[spec.Path]; !ok {
			t.Fatalf("root completion missing %q", spec.Path)
		}
	}
}

func TestHelpCompletionContainsAllHelpTopicsAndAliases(t *testing.T) {
	t.Parallel()

	values := suggestionValues(calculateCompletion("help ", CompletionContext{}))
	have := make(map[string]struct{}, len(values))
	for _, value := range values {
		have[value] = struct{}{}
	}

	for topic := range helpEntries {
		if topic == "" {
			continue
		}
		if _, ok := have[topic]; !ok {
			t.Fatalf("help completion missing topic %q", topic)
		}
	}
	for alias := range commandAliases {
		if _, ok := have[normalizeHelpTopic(alias)]; !ok {
			t.Fatalf("help completion missing alias topic %q", alias)
		}
	}
}

func extractRootHelpCommandPaths(body string) []string {
	lines := strings.Split(body, "\n")
	paths := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		left := line
		if idx := strings.Index(left, "  "); idx >= 0 {
			left = strings.TrimSpace(left[:idx])
		}
		tokens := strings.Fields(left)
		if len(tokens) == 0 {
			continue
		}
		pathTokens := make([]string, 0, len(tokens))
		for _, token := range tokens {
			if strings.HasPrefix(token, "<") || strings.HasPrefix(token, "[") {
				break
			}
			pathTokens = append(pathTokens, token)
		}
		if len(pathTokens) == 0 {
			continue
		}
		path := strings.Join(pathTokens, " ")
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}
