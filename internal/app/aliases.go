package app

import "strings"

var commandAliases = map[string]string{
	"q":                  "exit",
	"quit":               "exit",
	"conn":               "connect",
	"cx":                 "connect",
	"conns":              "connections",
	"ctx":                "context",
	"ls db":              "list databases",
	"show databases":     "list databases",
	"show dbs":           "list databases",
	"show index":         "show indexes",
	"show processes":     "show processlist",
	"show vars":          "show variables",
	"list users":         "show users",
	"show user accounts": "show users",
	"desc table":         "describe table",
	"create db":          "create database",
	"drop db":            "drop database",
	"dry on":             "dry-run on",
	"dry off":            "dry-run off",
	"test conn":          "connection test",
	"doctor conn":        "connection doctor",
}

func resolveAlias(line string) string {
	line = normalizeHelpTopic(line)
	if line == "" {
		return ""
	}

	if alias, ok := commandAliases[line]; ok {
		return alias
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return line
	}

	switch fields[0] {
	case "conn", "cx":
		fields[0] = "connect"
		return strings.Join(fields, " ")
	case "test":
		if len(fields) >= 2 && fields[1] == "conn" {
			return strings.Join(append([]string{"connection", "test"}, fields[2:]...), " ")
		}
	case "doctor":
		if len(fields) >= 2 && fields[1] == "conn" {
			return strings.Join(append([]string{"connection", "doctor"}, fields[2:]...), " ")
		}
	case "ctx":
		return "context"
	case "desc":
		if len(fields) >= 2 && fields[1] == "table" {
			return strings.Join(append([]string{"describe", "table"}, fields[2:]...), " ")
		}
	case "show":
		if len(fields) >= 2 {
			switch fields[1] {
			case "index":
				return strings.Join(append([]string{"show", "indexes"}, fields[2:]...), " ")
			case "processes":
				return strings.Join(append([]string{"show", "processlist"}, fields[2:]...), " ")
			case "vars":
				return strings.Join(append([]string{"show", "variables"}, fields[2:]...), " ")
			}
		}
	case "q", "quit":
		return "exit"
	case "conns":
		return "connections"
	}

	return line
}
