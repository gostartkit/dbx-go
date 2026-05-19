package app

import (
	"fmt"
	"strconv"
	"strings"

	"pkg.gostartkit.com/cmd"
)

type optionalString struct {
	Value string
	Set   bool
}

type optionalInt struct {
	Value int
	Set   bool
}

func bindOptionalStringFlag(f *cmd.FlagSet, target *optionalString, name, usage string) {
	f.Func(name, usage, func(value string) error {
		target.Value = strings.TrimSpace(value)
		target.Set = true
		return nil
	}, "")
}

func bindOptionalIntFlag(f *cmd.FlagSet, target *optionalInt, name, usage string) {
	f.Func(name, usage, func(value string) error {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return err
		}
		target.Value = parsed
		target.Set = true
		return nil
	}, "")
}

type inputValues map[string]string

func bindInputFlag(f *cmd.FlagSet, values inputValues) {
	f.Func("input", "template input in key=value form; repeatable", func(value string) error {
		key, parsedValue, err := splitInputValue(value)
		if err != nil {
			return err
		}
		values[key] = parsedValue
		return nil
	}, "")
}

func splitInputValue(value string) (string, string, error) {
	key, rest, found := strings.Cut(value, "=")
	if !found {
		return "", "", fmt.Errorf("input must be key=value")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", fmt.Errorf("input key is required")
	}

	return key, rest, nil
}
