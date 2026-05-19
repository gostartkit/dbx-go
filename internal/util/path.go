package util

import (
	"os"
	"path/filepath"
	"strings"
)

func ExpandHome(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return home, nil
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
