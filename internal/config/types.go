package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"pkg.gostartkit.com/dbx/internal/util"
)

const (
	defaultConnectTimeout = 10 * time.Second
	defaultQueryTimeout   = 30 * time.Second
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
	CurrentConnection string `json:"current_connection,omitempty"`
}

func (c *ConnectionConfig) ApplyDefaults() {
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

	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("connection name is required")
	}
	if c.Driver != "mysql" {
		return fmt.Errorf("unsupported driver %q", c.Driver)
	}
	if c.Mode != "direct" && c.Mode != "ssh" && c.Mode != "proxy-ssh" {
		return fmt.Errorf("unsupported connection mode %q", c.Mode)
	}
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be greater than zero")
	}
	if strings.TrimSpace(c.User) == "" {
		return fmt.Errorf("user is required")
	}
	if c.Timeout.ConnectSeconds <= 0 {
		return fmt.Errorf("timeout.connect_seconds must be greater than zero")
	}
	if c.Timeout.QuerySeconds <= 0 {
		return fmt.Errorf("timeout.query_seconds must be greater than zero")
	}
	if c.Mode == "direct" && c.Proxy != nil && strings.TrimSpace(c.Proxy.URL) != "" {
		return fmt.Errorf("proxy settings are not supported for direct mode")
	}
	if c.Mode == "ssh" && c.Proxy != nil && strings.TrimSpace(c.Proxy.URL) != "" {
		return fmt.Errorf("proxy settings are only supported for proxy-ssh mode")
	}
	if c.Mode == "proxy-ssh" {
		if c.Proxy == nil || strings.TrimSpace(c.Proxy.URL) == "" {
			return fmt.Errorf("proxy.url is required for proxy-ssh mode")
		}
		if _, err := ParseProxyURL(c.Proxy.URL); err != nil {
			return fmt.Errorf("proxy.url is invalid: %w", err)
		}
	}
	if c.Mode == "ssh" || c.Mode == "proxy-ssh" {
		if c.SSH == nil {
			return fmt.Errorf("ssh settings are required for %s mode", c.Mode)
		}
		if strings.TrimSpace(c.SSH.Host) == "" {
			return fmt.Errorf("ssh.host is required")
		}
		if c.SSH.Port <= 0 {
			return fmt.Errorf("ssh.port must be greater than zero")
		}
		if strings.TrimSpace(c.SSH.User) == "" {
			return fmt.Errorf("ssh.user is required")
		}
		if strings.TrimSpace(c.SSH.PrivateKey) == "" && strings.TrimSpace(c.SSH.PasswordEnv) == "" && strings.TrimSpace(c.SSH.Password) == "" {
			return fmt.Errorf("ssh.private_key or ssh.password_env or ssh.password is required")
		}
	}
	return nil
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
