package util

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var databaseNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func ValidateIdentifier(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("identifier is required")
	}
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("invalid identifier %q", value)
	}
	return nil
}

func ValidateDatabaseName(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("database name is required")
	}
	if !databaseNamePattern.MatchString(value) {
		return fmt.Errorf("invalid database name %q", value)
	}
	return nil
}

func EscapeMySQLString(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"'", "''",
		"\x00", "\\0",
	)
	return replacer.Replace(value)
}
