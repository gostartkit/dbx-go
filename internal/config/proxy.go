package config

import (
	"fmt"
	"net/url"
	"strings"
)

type ParsedProxyURL struct {
	Scheme   string
	Address  string
	Username string
	Password string
}

func ParseProxyURL(raw string) (*ParsedProxyURL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, fmt.Errorf("proxy URL is required")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL")
	}
	if parsed.Scheme == "" {
		return nil, fmt.Errorf("proxy URL must include a scheme")
	}
	if parsed.Scheme != "socks5" {
		return nil, fmt.Errorf("unsupported proxy scheme: %s; supported schemes: socks5", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("proxy URL must include host:port")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return nil, fmt.Errorf("proxy URL must not include a path")
	}

	result := &ParsedProxyURL{
		Scheme:  parsed.Scheme,
		Address: parsed.Host,
	}
	if parsed.User != nil {
		result.Username = parsed.User.Username()
		result.Password, _ = parsed.User.Password()
	}
	return result, nil
}

func RedactProxyURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.User == nil {
		return value
	}

	username := parsed.User.Username()
	if _, ok := parsed.User.Password(); !ok {
		return value
	}

	return fmt.Sprintf("%s://%s:***@%s", parsed.Scheme, username, parsed.Host)
}
