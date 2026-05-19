package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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

func (s *Store) writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
