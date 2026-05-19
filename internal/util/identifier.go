package util

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func ValidateIdentifier(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("identifier is required")
	}
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("identifier %q must match [a-zA-Z_][a-zA-Z0-9_]*", value)
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
