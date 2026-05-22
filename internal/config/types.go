package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"pkg.gostartkit.com/dbx/internal/util"
)

const (
	defaultConnectTimeout          = 10 * time.Second
	defaultQueryTimeout            = 30 * time.Second
	CurrentConnectionSchemaVersion = 1
)

type SSHConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	PrivateKey  string `json:"private_key"`
	Password    string `json:"password,omitempty"`
	PasswordEnv string `json:"password_env,omitempty"`
}

type ProxyConfig struct {
	URL string `json:"url"`
}

type TimeoutConfig struct {
	ConnectSeconds int `json:"connect_seconds,omitempty"`
	QuerySeconds   int `json:"query_seconds,omitempty"`
}

type ConnectionConfig struct {
	Version        int            `json:"version,omitempty"`
	Name           string         `json:"name"`
	Driver         string         `json:"driver"`
	Mode           string         `json:"mode"`
	Host           string         `json:"host"`
	Port           int            `json:"port"`
	User           string         `json:"user"`
	Password       string         `json:"password,omitempty"`
	PasswordEnv    string         `json:"password_env,omitempty"`
	PasswordPrompt bool           `json:"password_prompt,omitempty"`
	Proxy          *ProxyConfig   `json:"proxy,omitempty"`
	SSH            *SSHConfig     `json:"ssh,omitempty"`
	Timeout        *TimeoutConfig `json:"timeout,omitempty"`
}

type SessionFile struct {
	CurrentConnection string `json:"-"`
	CurrentDatabase   string `json:"-"`
}

func (s *SessionFile) MarshalJSON() ([]byte, error) {
	type sessionAlias struct {
		Connection string `json:"connection,omitempty"`
		Database   string `json:"database,omitempty"`
	}
	if s == nil {
		s = &SessionFile{}
	}
	return json.Marshal(sessionAlias{
		Connection: s.CurrentConnection,
		Database:   s.CurrentDatabase,
	})
}

func (s *SessionFile) UnmarshalJSON(data []byte) error {
	type sessionAlias struct {
		Connection       string `json:"connection,omitempty"`
		Database         string `json:"database,omitempty"`
		LegacyConnection string `json:"current_connection,omitempty"`
		LegacyDatabase   string `json:"current_database,omitempty"`
	}
	var raw sessionAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.CurrentConnection = strings.TrimSpace(raw.Connection)
	if s.CurrentConnection == "" {
		s.CurrentConnection = strings.TrimSpace(raw.LegacyConnection)
	}
	s.CurrentDatabase = strings.TrimSpace(raw.Database)
	if s.CurrentDatabase == "" {
		s.CurrentDatabase = strings.TrimSpace(raw.LegacyDatabase)
	}
	return nil
}

func (c *ConnectionConfig) UnmarshalJSON(data []byte) error {
	type rawConnectionConfig ConnectionConfig
	var raw rawConnectionConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = ConnectionConfig(raw)
	c.ApplyDefaults()
	return nil
}

func (c *ConnectionConfig) ApplyDefaults() {
	if c.Version == 0 {
		c.Version = CurrentConnectionSchemaVersion
	}
	if c.Driver == "" {
		c.Driver = "mysql"
	}
	if c.Port == 0 {
		c.Port = 3306
	}
	if c.SSH != nil && c.SSH.Port == 0 {
		c.SSH.Port = 22
	}
	if c.Timeout == nil {
		c.Timeout = &TimeoutConfig{}
	}
	if c.Timeout.ConnectSeconds <= 0 {
		c.Timeout.ConnectSeconds = int(defaultConnectTimeout / time.Second)
	}
	if c.Timeout.QuerySeconds <= 0 {
		c.Timeout.QuerySeconds = int(defaultQueryTimeout / time.Second)
	}
}

func (c *ConnectionConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("connection config is required")
	}
	c.ApplyDefaults()
	if c.Version != CurrentConnectionSchemaVersion {
		return fmt.Errorf("unsupported version %d", c.Version)
	}
	return c.ValidateDetailed().Error()
}

func (c *ConnectionConfig) Address() string {
	c.ApplyDefaults()
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *ConnectionConfig) ConnectTimeout() time.Duration {
	c.ApplyDefaults()
	return time.Duration(c.Timeout.ConnectSeconds) * time.Second
}

func (c *ConnectionConfig) QueryTimeout() time.Duration {
	c.ApplyDefaults()
	return time.Duration(c.Timeout.QuerySeconds) * time.Second
}

func (c *ConnectionConfig) UsesSSH() bool {
	return c != nil && (c.Mode == "ssh" || c.Mode == "proxy-ssh")
}

func (c *ConnectionConfig) UsesProxy() bool {
	return c != nil && (c.Mode == "proxy" || c.Mode == "proxy-ssh")
}

func (c *ConnectionConfig) PasswordValue() (string, error) {
	if c.PasswordEnv == "" {
		return c.Password, nil
	}

	value := os.Getenv(c.PasswordEnv)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is empty", c.PasswordEnv)
	}
	return value, nil
}

func (s *SSHConfig) PrivateKeyPath() (string, error) {
	if s == nil {
		return "", fmt.Errorf("ssh settings are required")
	}
	return util.ExpandHome(s.PrivateKey)
}
