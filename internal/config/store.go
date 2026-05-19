package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const historyLimit = 1000

type Store struct {
	RootDir string
}

func DefaultRootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "dbx"), nil
}

func NewStore(rootDir string) *Store {
	return &Store{RootDir: rootDir}
}

func (s *Store) EnsureLayout() error {
	if err := os.MkdirAll(s.RootDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(s.GlobalTemplatesDir(), 0o755)
}

func (s *Store) SessionPath() string {
	return filepath.Join(s.RootDir, "session.json")
}

func (s *Store) HistoryPath() string {
	return filepath.Join(s.RootDir, "history")
}

func (s *Store) GlobalTemplatesDir() string {
	return filepath.Join(s.RootDir, "templates")
}

func (s *Store) ConnectionDir(name string) string {
	return filepath.Join(s.RootDir, name)
}

func (s *Store) ConnectionConfigPath(name string) string {
	return filepath.Join(s.ConnectionDir(name), "config.json")
}

func (s *Store) ConnectionTemplatesDir(name string) string {
	return filepath.Join(s.ConnectionDir(name), "templates")
}

func (s *Store) LoadSession() (*SessionFile, error) {
	path := s.SessionPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &SessionFile{}, nil
		}
		return nil, err
	}

	var session SessionFile
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Store) SaveSession(session *SessionFile) error {
	if session == nil {
		session = &SessionFile{}
	}
	return s.writeJSON(s.SessionPath(), session)
}

func (s *Store) LoadConnection(name string) (*ConnectionConfig, error) {
	path := s.ConnectionConfigPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ConnectionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *Store) SaveConnection(cfg *ConnectionConfig) error {
	if cfg == nil {
		return fmt.Errorf("connection config is required")
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	return s.writeJSON(s.ConnectionConfigPath(cfg.Name), cfg)
}

func (s *Store) DeleteConnection(name string) error {
	path := s.ConnectionDir(name)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("connection directory %s is not a directory", path)
	}
	return os.RemoveAll(path)
}

func (s *Store) ConnectionExists(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	info, err := os.Stat(s.ConnectionConfigPath(name))
	return err == nil && !info.IsDir()
}

func (s *Store) ListConnections() ([]ConnectionConfig, error) {
	entries, err := os.ReadDir(s.RootDir)
	if err != nil {
		return nil, err
	}

	connections := make([]ConnectionConfig, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "templates" {
			continue
		}

		cfg, err := s.LoadConnection(entry.Name())
		if err != nil {
			continue
		}
		connections = append(connections, *cfg)
	}

	sort.Slice(connections, func(i, j int) bool {
		return connections[i].Name < connections[j].Name
	})
	return connections, nil
}

func (s *Store) LoadHistory() ([]string, error) {
	file, err := os.Open(s.HistoryPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	history := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		history = append(history, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(history) > historyLimit {
		history = append([]string(nil), history[len(history)-historyLimit:]...)
	}
	return history, nil
}

func (s *Store) AppendHistory(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}
	if err := os.MkdirAll(s.RootDir, 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(s.HistoryPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(command + "\n")
	return err
}

func (s *Store) writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".dbx-*.tmp")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return err
	}
	if err := tempFile.Chmod(0o644); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return err
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}
	return nil
}
