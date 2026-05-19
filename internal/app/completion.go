package app

import (
	"slices"
	"strings"

	"pkg.gostartkit.com/dbx/internal/ui"
)

var topLevelCommands = []string{
	"connect",
	"connections",
	"connection create",
	"connection doctor",
	"connection edit",
	"connection delete",
	"connection show",
	"connection test",
	"create database",
	"list databases",
	"drop database",
	"status",
	"dry-run on",
	"dry-run off",
	"help",
	"exit",
}

func calculateCompletion(line string, savedConnections []string) ui.Completion {
	trailingSpace := strings.HasSuffix(line, " ")
	trimmed := strings.TrimLeft(line, " ")
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ui.Completion{Prefix: "", Candidates: append([]string(nil), topLevelCommands...)}
	}

	if len(fields) == 1 && !trailingSpace {
		prefix := fields[0]
		candidates := filterByPrefix(topLevelCommands, prefix)
		return ui.Completion{Prefix: prefix, Candidates: candidates}
	}

	first := fields[0]
	last := fields[len(fields)-1]
	prefix := last
	if trailingSpace {
		prefix = ""
	}

	switch first {
	case "connection":
		switch len(fields) {
		case 1:
			return ui.Completion{Prefix: prefix, Candidates: filterByPrefix([]string{"create", "doctor", "edit", "delete", "show", "test"}, prefix)}
		case 2:
			if trailingSpace && slices.Contains([]string{"doctor", "edit", "delete", "show", "test"}, fields[1]) {
				return ui.Completion{Prefix: "", Candidates: filterByPrefix(sortedStrings(savedConnections), "")}
			}
			if !trailingSpace && len(fields) == 2 {
				if slices.Contains([]string{"doctor", "edit", "delete", "show", "test"}, fields[1]) {
					return ui.Completion{Prefix: fields[1], Candidates: filterByPrefix([]string{"doctor", "edit", "delete", "show", "test"}, fields[1])}
				}
				return ui.Completion{Prefix: fields[1], Candidates: filterByPrefix([]string{"create", "doctor", "edit", "delete", "show", "test"}, fields[1])}
			}
		case 3:
			if slices.Contains([]string{"doctor", "edit", "delete", "show", "test"}, fields[1]) {
				return ui.Completion{Prefix: prefix, Candidates: filterByPrefix(sortedStrings(savedConnections), prefix)}
			}
		}
	case "connect":
		if len(fields) >= 2 || trailingSpace {
			return ui.Completion{Prefix: prefix, Candidates: filterByPrefix(sortedStrings(savedConnections), prefix)}
		}
	case "create":
		return ui.Completion{Prefix: prefix, Candidates: filterByPrefix([]string{"database"}, prefix)}
	case "list":
		return ui.Completion{Prefix: prefix, Candidates: filterByPrefix([]string{"databases"}, prefix)}
	case "drop":
		return ui.Completion{Prefix: prefix, Candidates: filterByPrefix([]string{"database"}, prefix)}
	case "dry-run":
		return ui.Completion{Prefix: prefix, Candidates: filterByPrefix([]string{"on", "off"}, prefix)}
	case "dry":
		return ui.Completion{Prefix: prefix, Candidates: filterByPrefix([]string{"on", "off"}, prefix)}
	case "help":
		topics := []string{
			"connect",
			"connections",
			"connection",
			"connection create",
			"connection doctor",
			"connection edit",
			"connection delete",
			"connection show",
			"connection test",
			"create database",
			"list databases",
			"drop database",
			"status",
			"dry-run",
			"aliases",
			"exit",
		}
		return ui.Completion{Prefix: prefix, Candidates: filterByPrefix(topics, strings.TrimSpace(strings.Join(fields[1:], " ")))}
	}

	return ui.Completion{Prefix: prefix, Candidates: nil}
}

func filterByPrefix(candidates []string, prefix string) []string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return append([]string(nil), candidates...)
	}

	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			out = append(out, candidate)
		}
	}
	return out
}

func sortedStrings(values []string) []string {
	sorted := append([]string(nil), values...)
	slices.Sort(sorted)
	return sorted
}
