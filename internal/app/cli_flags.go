package app

import (
	"fmt"
	"strings"

	"pkg.gostartkit.com/cmd"
)

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
	f.MarkRepeatable("input")
	f.SetKind("input", "template_input")
	f.SetCompletionKey("input", "template-input")
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
