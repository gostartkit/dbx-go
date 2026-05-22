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
	"time"
)

const historyLimit = 1000

type AuditRecord struct {
	Timestamp  time.Time `json:"timestamp"`
	Command    string    `json:"command"`
	Connection string    `json:"connection,omitempty"`
	Mode       string    `json:"mode,omitempty"`
	DryRun     bool      `json:"dry_run"`
	Success    bool      `json:"success"`
	DurationMS int64     `json:"duration_ms"`
}

type Store struct {
	RootDir       string
	historyAppend *os.File
	auditAppend   *os.File
}

type ConnectionRecord struct {
	Name   string
	Config *ConnectionConfig
	Error  error
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

func (s *Store) Close() error {
	var errs []error
	if s.historyAppend != nil {
		errs = append(errs, s.historyAppend.Close())
		s.historyAppend = nil
	}
	if s.auditAppend != nil {
		errs = append(errs, s.auditAppend.Close())
		s.auditAppend = nil
	}
	return errors.Join(errs...)
}

func (s *Store) EnsureLayout() error {
	if err := os.MkdirAll(s.RootDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.GlobalTemplatesDir(), 0o755); err != nil {
		return err
	}
	return os.MkdirAll(s.LogsDir(), 0o755)
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

func (s *Store) LogsDir() string {
	return filepath.Join(s.RootDir, "logs")
}

func (s *Store) AuditLogPath() string {
	return filepath.Join(s.LogsDir(), "audit.jsonl")
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
	cfg, err := s.LoadConnectionUnchecked(name)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *Store) LoadConnectionUnchecked(name string) (*ConnectionConfig, error) {
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
	records, err := s.ListConnectionRecords()
	if err != nil {
		return nil, err
	}

	connections := make([]ConnectionConfig, 0, len(records))
	for _, record := range records {
		if record.Error != nil || record.Config == nil {
			continue
		}
		connections = append(connections, *record.Config)
	}
	return connections, nil
}

func (s *Store) LoadConnectionRecord(name string) (*ConnectionRecord, error) {
	path := s.ConnectionConfigPath(name)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	record := &ConnectionRecord{Name: name}
	cfg, err := s.LoadConnectionUnchecked(name)
	if cfg != nil {
		record.Config = cfg
		if strings.TrimSpace(cfg.Name) != "" {
			record.Name = cfg.Name
		}
		if validateErr := cfg.Validate(); validateErr != nil {
			record.Error = validateErr
		}
		return record, nil
	}

	record.Error = err
	return record, nil
}

func (s *Store) ListConnectionRecords() ([]ConnectionRecord, error) {
	entries, err := os.ReadDir(s.RootDir)
	if err != nil {
		return nil, err
	}

	records := make([]ConnectionRecord, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "templates" || entry.Name() == "logs" {
			continue
		}

		record, err := s.LoadConnectionRecord(entry.Name())
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Name < records[j].Name
	})
	return records, nil
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

	file, err := s.appendFile(s.HistoryPath(), &s.historyAppend)
	if err != nil {
		return err
	}
	_, err = file.WriteString(command + "\n")
	return err
}

func (s *Store) AppendAudit(record *AuditRecord) error {
	if record == nil {
		return fmt.Errorf("audit record is required")
	}
	if err := os.MkdirAll(s.LogsDir(), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	file, err := s.appendFile(s.AuditLogPath(), &s.auditAppend)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}

func (s *Store) LoadAudit(limit int) ([]AuditRecord, error) {
	file, err := os.Open(s.AuditLogPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	records := make([]AuditRecord, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record AuditRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(records) > limit {
		records = append([]AuditRecord(nil), records[len(records)-limit:]...)
	}
	return records, nil
}

func (s *Store) appendFile(path string, current **os.File) (*os.File, error) {
	if current != nil && *current != nil {
		return *current, nil
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if current != nil {
		*current = file
	}
	return file, nil
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
