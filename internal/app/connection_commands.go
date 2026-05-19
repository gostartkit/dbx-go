package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func (a *Application) handleConnectByName(ctx context.Context, name string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "connect"}, func(meta *auditMetadata) error {
		return a.connectByName(ctx, name, meta)
	})
}

func (a *Application) connectByName(ctx context.Context, name string, meta *auditMetadata) error {
	cfg, err := a.store.LoadConnection(name)
	if err != nil {
		return util.WrapLayer("config", "load connection "+name, err)
	}
	if meta != nil {
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
	}

	a.prompt.Println("Execution Plan")
	a.prompt.Printf("  1. Open %s MySQL connection %q to %s\n", cfg.Mode, cfg.Name, cfg.Address())
	if cfg.UsesProxy() && cfg.Proxy != nil {
		if cfg.Mode == "proxy" {
			a.prompt.Printf("  2. Reach MySQL through SOCKS5 proxy %s\n", config.RedactProxyURL(cfg.Proxy.URL))
		} else {
			a.prompt.Printf("  2. Reach SSH through SOCKS5 proxy %s\n", config.RedactProxyURL(cfg.Proxy.URL))
		}
	}
	if cfg.UsesSSH() && cfg.SSH != nil {
		step := 2
		if cfg.UsesProxy() {
			step = 3
		}
		a.prompt.Printf("  %d. Tunnel through SSH bastion %s:%d as %s\n", step, cfg.SSH.Host, cfg.SSH.Port, cfg.SSH.User)
	}

	confirmed, err := a.confirm(ctx, "Confirm execution?", true)
	if err != nil {
		return err
	}
	if !confirmed {
		a.prompt.Println("Cancelled.")
		return nil
	}

	return a.connectWithConfig(ctx, cfg, true)
}

func (a *Application) handleConnectionCreate(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "connection create"}, func(meta *auditMetadata) error {
		a.prompt.Println("Create Connection")

		name, err := a.ask(ctx, "Connection name", "")
		if err != nil {
			return err
		}
		if err := util.ValidateIdentifier(name); err != nil {
			return util.WrapLayer("validation", "validate connection name", err)
		}
		if a.store.ConnectionExists(name) {
			return util.WrapLayer("config", "create connection", fmt.Errorf("connection %q already exists", name))
		}

		cfg := &config.ConnectionConfig{
			Name:   name,
			Driver: "mysql",
			Mode:   "direct",
		}
		if err := a.promptConnectionConfig(ctx, cfg, true); err != nil {
			return err
		}
		meta.Connection = cfg.Name
		meta.Mode = cfg.Mode
		return a.testSaveAndMaybeConnect(ctx, cfg, false)
	})
}

func (a *Application) handleConnectionEdit(ctx context.Context, name string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "connection edit", Connection: name}, func(meta *auditMetadata) error {
		cfg, err := a.store.LoadConnection(name)
		if err != nil {
			return util.WrapLayer("config", "load connection "+name, err)
		}
		meta.Mode = cfg.Mode

		a.prompt.Printf("Edit Connection: %s\n", name)
		if err := a.promptConnectionConfig(ctx, cfg, false); err != nil {
			return err
		}
		meta.Mode = cfg.Mode

		testConnection, err := a.confirm(ctx, "Test connection?", true)
		if err != nil {
			return err
		}
		if testConnection {
			if err := a.testConnection(ctx, cfg); err != nil {
				return err
			}
			a.prompt.Println("Connection successful.")
		}

		saveChanges, err := a.confirm(ctx, "Save changes?", true)
		if err != nil {
			return err
		}
		if !saveChanges {
			a.prompt.Println("Cancelled.")
			return nil
		}

		if err := a.store.SaveConnection(cfg); err != nil {
			return util.WrapLayer("config", "save connection "+cfg.Name, err)
		}
		a.prompt.Printf("Saved: %s\n", a.store.ConnectionConfigPath(cfg.Name))

		connectNow, err := a.confirm(ctx, "Connect now?", true)
		if err != nil {
			return err
		}
		if !connectNow {
			return nil
		}
		return a.connectWithConfig(ctx, cfg, true)
	})
}

func (a *Application) handleConnectionDelete(ctx context.Context, name string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "connection delete", Connection: name}, func(meta *auditMetadata) error {
		if !a.store.ConnectionExists(name) {
			return util.WrapLayer("config", "delete connection "+name, os.ErrNotExist)
		}
		if cfg, err := a.store.LoadConnection(name); err == nil {
			meta.Mode = cfg.Mode
		}

		confirmed, err := a.confirm(ctx, fmt.Sprintf("Delete connection %q?", name), false)
		if err != nil {
			return err
		}
		if !confirmed {
			a.prompt.Println("Cancelled.")
			return nil
		}

		if err := a.deleteConnectionByName(name); err != nil {
			return err
		}
		a.prompt.Printf("Deleted connection %s.\n", name)
		return nil
	})
}

func (a *Application) deleteConnectionByName(name string) error {
	if err := a.store.DeleteConnection(name); err != nil {
		return util.WrapLayer("config", "delete connection "+name, err)
	}

	if a.session.Connection != nil && a.session.Connection.Name == name {
		if err := a.session.Reset(); err != nil {
			return util.WrapLayer("mysql", "close active connection before delete", err)
		}
	}
	if a.reconnectCandidate != nil && a.reconnectCandidate.Name == name {
		a.reconnectCandidate = nil
	}

	sessionFile, err := a.store.LoadSession()
	if err == nil && sessionFile.CurrentConnection == name {
		if err := a.store.SaveSession(&config.SessionFile{}); err != nil {
			return util.WrapLayer("config", "clear session after delete", err)
		}
	}
	return nil
}

func (a *Application) handleConnectionShow(ctx context.Context, name string) error {
	return a.auditCommand(ctx, auditMetadata{Command: "connection show", Connection: name}, func(meta *auditMetadata) error {
		cfg, err := a.store.LoadConnection(name)
		if err != nil {
			return util.WrapLayer("config", "load connection "+name, err)
		}
		meta.Mode = cfg.Mode

		a.prompt.Printf("Name: %s\n", cfg.Name)
		a.prompt.Printf("Driver: %s\n", cfg.Driver)
		a.prompt.Printf("Mode: %s\n", cfg.Mode)
		a.prompt.Println()
		a.prompt.Printf("Host: %s\n", cfg.Address())
		a.prompt.Printf("User: %s\n", cfg.User)
		a.prompt.Printf("Connect timeout: %s\n", cfg.ConnectTimeout())
		a.prompt.Printf("Query timeout: %s\n", cfg.QueryTimeout())
		a.prompt.Println()
		a.prompt.Println("Password:")
		switch {
		case cfg.PasswordPrompt:
			a.prompt.Println("  prompt every time")
		case strings.TrimSpace(cfg.PasswordEnv) != "":
			a.prompt.Printf("  env: %s\n", cfg.PasswordEnv)
		case strings.TrimSpace(cfg.Password) != "":
			a.prompt.Println("  saved: [redacted]")
		default:
			a.prompt.Println("  not configured")
		}

		if cfg.Proxy != nil && strings.TrimSpace(cfg.Proxy.URL) != "" {
			a.prompt.Println()
			a.prompt.Println("Proxy:")
			a.prompt.Printf("  url: %s\n", config.RedactProxyURL(cfg.Proxy.URL))
		}

		if cfg.UsesSSH() && cfg.SSH != nil {
			a.prompt.Println()
			a.prompt.Println("SSH:")
			a.prompt.Printf("  host: %s:%d\n", cfg.SSH.Host, cfg.SSH.Port)
			a.prompt.Printf("  user: %s\n", cfg.SSH.User)
			if strings.TrimSpace(cfg.SSH.PrivateKey) != "" {
				a.prompt.Printf("  private_key: %s\n", cfg.SSH.PrivateKey)
			}
			if strings.TrimSpace(cfg.SSH.PasswordEnv) != "" {
				a.prompt.Printf("  password_env: %s\n", cfg.SSH.PasswordEnv)
			} else if strings.TrimSpace(cfg.SSH.Password) != "" {
				a.prompt.Println("  password: [redacted]")
			}
		}

		return nil
	})
}

func (a *Application) promptConnectionConfig(ctx context.Context, cfg *config.ConnectionConfig, isCreate bool) error {
	cfg.Driver = "mysql"
	cfg.ApplyDefaults()

	mode, err := a.choose(ctx, "Connection mode", []string{"direct", "ssh", "proxy", "proxy-ssh"}, cfg.Mode)
	if err != nil {
		return err
	}
	cfg.Mode = mode

	host, err := a.ask(ctx, "Database host", cfg.Host)
	if err != nil {
		return err
	}
	cfg.Host = strings.TrimSpace(host)

	port, err := a.askInt(ctx, "Database port", cfg.Port)
	if err != nil {
		return err
	}
	cfg.Port = port

	user, err := a.ask(ctx, "Database user", cfg.User)
	if err != nil {
		return err
	}
	cfg.User = strings.TrimSpace(user)

	if err := a.promptPasswordHandling(ctx, cfg); err != nil {
		return err
	}

	connectTimeout, err := a.askInt(ctx, "Connect timeout seconds", cfg.Timeout.ConnectSeconds)
	if err != nil {
		return err
	}
	queryTimeout, err := a.askInt(ctx, "Query timeout seconds", cfg.Timeout.QuerySeconds)
	if err != nil {
		return err
	}
	cfg.Timeout.ConnectSeconds = connectTimeout
	cfg.Timeout.QuerySeconds = queryTimeout

	if cfg.UsesProxy() {
		proxyURL, err := a.askProxyURL(ctx, cfg.Proxy)
		if err != nil {
			return err
		}
		cfg.Proxy = &config.ProxyConfig{URL: proxyURL}
	} else {
		cfg.Proxy = nil
	}

	if cfg.UsesSSH() {
		if cfg.SSH == nil {
			cfg.SSH = &config.SSHConfig{}
		}

		sshHost, err := a.ask(ctx, "SSH host", cfg.SSH.Host)
		if err != nil {
			return err
		}
		cfg.SSH.Host = strings.TrimSpace(sshHost)

		sshPort, err := a.askInt(ctx, "SSH port", cfg.SSH.Port)
		if err != nil {
			return err
		}
		cfg.SSH.Port = sshPort

		sshUser, err := a.ask(ctx, "SSH user", cfg.SSH.User)
		if err != nil {
			return err
		}
		cfg.SSH.User = strings.TrimSpace(sshUser)

		if err := a.promptSSHAuthHandling(ctx, cfg.SSH, isCreate); err != nil {
			return err
		}
	} else {
		cfg.SSH = nil
	}

	if err := cfg.Validate(); err != nil {
		return util.WrapLayer("validation", "validate connection config", err)
	}
	return nil
}

func (a *Application) askProxyURL(ctx context.Context, proxyCfg *config.ProxyConfig) (string, error) {
	if proxyCfg != nil && strings.TrimSpace(proxyCfg.URL) != "" {
		redacted := config.RedactProxyURL(proxyCfg.URL)
		if redacted != proxyCfg.URL {
			value, err := a.ask(ctx, "Proxy URL (leave blank to keep current)", "")
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(value) == "" {
				return proxyCfg.URL, nil
			}
			return strings.TrimSpace(value), nil
		}
		return a.ask(ctx, "Proxy URL", redacted)
	}
	return a.ask(ctx, "Proxy URL", "")
}

func (a *Application) promptPasswordHandling(ctx context.Context, cfg *config.ConnectionConfig) error {
	defaultOption := "prompt every time"
	switch {
	case strings.TrimSpace(cfg.PasswordEnv) != "":
		defaultOption = "env variable"
	case strings.TrimSpace(cfg.Password) != "":
		defaultOption = "save password"
	case cfg.PasswordPrompt:
		defaultOption = "prompt every time"
	}

	choice, err := a.choose(ctx, "Password handling", []string{"prompt every time", "env variable", "save password"}, defaultOption)
	if err != nil {
		return err
	}

	cfg.PasswordPrompt = false
	cfg.Password = ""
	cfg.PasswordEnv = ""

	switch choice {
	case "prompt every time":
		cfg.PasswordPrompt = true
	case "env variable":
		defaultEnv := cfg.PasswordEnv
		if defaultEnv == "" {
			defaultEnv = "MYSQL_" + strings.ToUpper(cfg.Name) + "_PASSWORD"
		}
		envName, err := a.ask(ctx, "Environment variable name", defaultEnv)
		if err != nil {
			return err
		}
		cfg.PasswordEnv = strings.TrimSpace(envName)
	case "save password":
		password, err := a.askPassword(ctx, "Database password")
		if err != nil {
			return err
		}
		cfg.Password = password
	}

	return nil
}

func (a *Application) promptSSHAuthHandling(ctx context.Context, sshCfg *config.SSHConfig, _ bool) error {
	defaultOption := "private key"
	switch {
	case strings.TrimSpace(sshCfg.PasswordEnv) != "":
		defaultOption = "env variable"
	case strings.TrimSpace(sshCfg.PrivateKey) != "":
		defaultOption = "private key"
	}

	choice, err := a.choose(ctx, "SSH auth", []string{"private key", "env variable"}, defaultOption)
	if err != nil {
		return err
	}

	sshCfg.PrivateKey = ""
	sshCfg.PasswordEnv = ""

	switch choice {
	case "private key":
		defaultKey := sshCfg.PrivateKey
		if defaultKey == "" {
			defaultKey = "~/.ssh/id_rsa"
		}
		keyPath, err := a.ask(ctx, "SSH private key", defaultKey)
		if err != nil {
			return err
		}
		sshCfg.PrivateKey = strings.TrimSpace(keyPath)
	case "env variable":
		defaultEnv := sshCfg.PasswordEnv
		if defaultEnv == "" {
			defaultEnv = "SSH_" + strings.ToUpper(strings.ReplaceAll(sshCfg.User, "-", "_")) + "_PASSWORD"
		}
		envName, err := a.ask(ctx, "SSH password env variable", defaultEnv)
		if err != nil {
			return err
		}
		sshCfg.PasswordEnv = strings.TrimSpace(envName)
	}

	return nil
}

func (a *Application) testSaveAndMaybeConnect(ctx context.Context, cfg *config.ConnectionConfig, alreadySaved bool) error {
	testConnection, err := a.confirm(ctx, "Test connection?", true)
	if err != nil {
		return err
	}
	var testErr error
	if testConnection {
		if err := a.testConnection(ctx, cfg); err != nil {
			testErr = err
			a.printConnectionTestFailure(cfg.Name, err)
		} else {
			a.prompt.Println("Connection successful.")
		}
	}

	if !alreadySaved {
		saveConnection, err := a.confirm(ctx, "Save connection?", true)
		if err != nil {
			return err
		}
		if !saveConnection {
			a.prompt.Println("Cancelled.")
			return nil
		}

		if err := a.store.SaveConnection(cfg); err != nil {
			return util.WrapLayer("config", "save connection "+cfg.Name, err)
		}
		a.printSavedConnection(cfg.Name)
	}

	if testErr != nil {
		a.printConnectionEditHint(cfg.Name)
		return nil
	}

	connectNow, err := a.confirm(ctx, "Connect now?", true)
	if err != nil {
		return err
	}
	if !connectNow {
		return nil
	}
	return a.connectWithConfig(ctx, cfg, true)
}

func (a *Application) printConnectionTestFailure(name string, err error) {
	a.prompt.Println("Connection test failed:")
	a.prompt.Printf("  %v\n", err)
}

func (a *Application) printSavedConnection(name string) {
	a.prompt.Println("Saved connection:")
	a.prompt.Printf("  %s\n", a.store.ConnectionConfigPath(name))
}

func (a *Application) printConnectionEditHint(name string) {
	a.prompt.Println()
	a.prompt.Println("You can update it with:")
	a.prompt.Printf("  connection edit %s\n", name)
}

func (a *Application) testConnection(ctx context.Context, cfg *config.ConnectionConfig) error {
	runtimeCfg, err := a.prepareConnectionForOpen(ctx, cfg)
	if err != nil {
		return err
	}
	db, err := a.connector.Open(ctx, runtimeCfg)
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}

func (a *Application) connectWithConfig(ctx context.Context, cfg *config.ConnectionConfig, persistSession bool) error {
	if err := a.activateConnection(ctx, cfg, persistSession); err != nil {
		return err
	}
	a.prompt.Printf("Connected to %s.\n", cfg.Name)
	return nil
}

func (a *Application) activateConnection(ctx context.Context, cfg *config.ConnectionConfig, persistSession bool) error {
	db, err := a.openConnection(ctx, cfg)
	if err != nil {
		return err
	}

	if err := a.session.Close(); err != nil {
		db.Close()
		return err
	}

	a.session.Connection = cloneConnectionConfig(cfg)
	a.session.DB = db
	a.clearDatabaseSelection()
	a.clearTableCompletion()
	a.clearUserCompletion()

	if persistSession {
		if err := a.saveCurrentSession(); err != nil {
			return util.WrapLayer("config", "save session", err)
		}
	}
	return nil
}

func (a *Application) openConnection(ctx context.Context, cfg *config.ConnectionConfig) (*sql.DB, error) {
	runtimeCfg, err := a.prepareConnectionForOpen(ctx, cfg)
	if err != nil {
		return nil, err
	}

	db, err := a.connector.Open(ctx, runtimeCfg)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (a *Application) prepareConnectionForOpen(ctx context.Context, cfg *config.ConnectionConfig) (*config.ConnectionConfig, error) {
	cloned := cloneConnectionConfig(cfg)
	if cloned.PasswordPrompt {
		password, err := a.askPassword(ctx, "Database password")
		if err != nil {
			return nil, err
		}
		cloned.Password = password
	}
	return cloned, nil
}

func cloneConnectionConfig(cfg *config.ConnectionConfig) *config.ConnectionConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	if cfg.Proxy != nil {
		proxyCopy := *cfg.Proxy
		cloned.Proxy = &proxyCopy
	}
	if cfg.SSH != nil {
		sshCopy := *cfg.SSH
		cloned.SSH = &sshCopy
	}
	if cfg.Timeout != nil {
		timeoutCopy := *cfg.Timeout
		cloned.Timeout = &timeoutCopy
	}
	return &cloned
}

func (a *Application) askInt(ctx context.Context, label string, defaultValue int) (int, error) {
	defaultText := ""
	if defaultValue > 0 {
		defaultText = strconv.Itoa(defaultValue)
	}

	value, err := a.ask(ctx, label, defaultText)
	if err != nil {
		return 0, err
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, util.WrapLayer("validation", "parse integer for "+label, err)
	}
	if parsed <= 0 {
		return 0, util.WrapLayer("validation", "parse integer for "+label, fmt.Errorf("value must be greater than zero"))
	}
	return parsed, nil
}
