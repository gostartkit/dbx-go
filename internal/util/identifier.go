package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var databaseNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
var mysqlUsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
var mysqlUserHostPattern = regexp.MustCompile(`^[a-zA-Z0-9._%-]+$`)
var tableNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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

func ValidateTableName(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("table name is required")
	}
	if !tableNamePattern.MatchString(value) {
		return fmt.Errorf("invalid table name %q", value)
	}
	return nil
}

func ValidateMySQLUsername(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("username is required")
	}
	if !mysqlUsernamePattern.MatchString(value) {
		return fmt.Errorf("invalid MySQL username %q", value)
	}
	return nil
}

func ValidateMySQLUserHost(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("user host is required")
	}
	if !mysqlUserHostPattern.MatchString(value) {
		return fmt.Errorf("invalid MySQL user host %q", value)
	}
	return nil
}

func GeneratePassword(length int) (string, error) {
	if length < 12 {
		length = 12
	}

	uppercase := "ABCDEFGHJKLMNPQRSTUVWXYZ"
	lowercase := "abcdefghijkmnopqrstuvwxyz"
	digits := "23456789"
	all := uppercase + lowercase + digits

	chars := make([]byte, 0, length)
	pools := []string{uppercase, lowercase, digits}
	for _, pool := range pools {
		ch, err := randomChar(pool)
		if err != nil {
			return "", err
		}
		chars = append(chars, ch)
	}
	for len(chars) < length {
		ch, err := randomChar(all)
		if err != nil {
			return "", err
		}
		chars = append(chars, ch)
	}

	for index := len(chars) - 1; index > 0; index-- {
		swapIndex, err := rand.Int(rand.Reader, big.NewInt(int64(index+1)))
		if err != nil {
			return "", err
		}
		j := int(swapIndex.Int64())
		chars[index], chars[j] = chars[j], chars[index]
	}

	return string(chars), nil
}

func randomChar(pool string) (byte, error) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
	if err != nil {
		return 0, err
	}
	return pool[index.Int64()], nil
}

func EscapeMySQLString(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"'", "''",
		"\x00", "\\0",
	)
	return replacer.Replace(value)
}

func QuoteMySQLIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}
