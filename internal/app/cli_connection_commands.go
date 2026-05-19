package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/cmd"
	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

type connectionCreateFlags struct {
	driver         string
	mode           string
	host           string
	port           int
	user           string
	passwordEnv    string
	password       string
	proxyURL       string
	sshHost        string
	sshPort        int
	sshUser        string
	sshPasswordEnv string
	sshPassword    string
	sshPrivateKey  string
	connectTimeout int
	queryTimeout   int
	test           bool
	connectNow     bool
	force          bool
}

type connectionEditFlags struct {
	driver         optionalString
	mode           optionalString
	host           optionalString
	port           optionalInt
	user           optionalString
	passwordEnv    optionalString
	password       optionalString
	proxyURL       optionalString
	sshHost        optionalString
	sshPort        optionalInt
	sshUser        optionalString
	sshPasswordEnv optionalString
	sshPassword    optionalString
	sshPrivateKey  optionalString
	connectTimeout optionalInt
	queryTimeout   optionalInt
	test           bool
}

func (b *cliBuilder) connectCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "connect",
		UsageLine:   "dbx connect <name>",
		Short:       "Connect to a saved connection",
		Long:        helpEntries["connect"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name"}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) == 0 {
				return b.withApplication(ctx, func(application *Application) error {
					if strings.EqualFold(b.globals.Format, "json") {
						return util.WrapLayer("validation", "connect", fmt.Errorf("connect without a name is only supported in text mode"))
					}
					return application.handleConnect(ctx)
				})
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "connect"}, func(application *Application, meta *auditMetadata) error {
				if len(args) != 1 {
					return util.WrapLayer("validation", "connect", fmt.Errorf("usage: dbx connect <name>"))
				}
				if cfg, err := application.store.LoadConnection(args[0]); err == nil {
					meta.Connection = cfg.Name
					meta.Mode = cfg.Mode
				}

				result, err := application.connectNonInteractive(ctx, args[0])
				if err != nil {
					return err
				}
				return b.writeOutput(result, func() error {
					fmt.Fprintln(b.out, result.Message)
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) connectionsCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "connections",
		UsageLine: "dbx connections",
		Short:     "List saved connections",
		Long:      helpEntries["connections"].body,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			return b.withAuditedApplication(ctx, auditMetadata{Command: "connections"}, func(application *Application, meta *auditMetadata) error {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "connections", err)
				}
				result, err := application.listConnectionSummaries()
				if err != nil {
					return err
				}
				return b.writeOutput(result, func() error {
					if len(result.Connections) == 0 {
						fmt.Fprintln(b.out, "No configured connections found.")
						return nil
					}
					fmt.Fprintln(b.out, "Configured connections:")
					for _, connection := range result.Connections {
						if connection.ViaProxy != "" && connection.ViaSSH != "" {
							fmt.Fprintf(b.out, "  - %s (%s %s %s via %s -> %s)\n", connection.Name, connection.Driver, connection.Mode, connection.Address, connection.ViaProxy, connection.ViaSSH)
							continue
						}
						if connection.ViaProxy != "" {
							fmt.Fprintf(b.out, "  - %s (%s %s %s via %s)\n", connection.Name, connection.Driver, connection.Mode, connection.Address, connection.ViaProxy)
							continue
						}
						if connection.ViaSSH != "" {
							fmt.Fprintf(b.out, "  - %s (%s %s %s via %s)\n", connection.Name, connection.Driver, connection.Mode, connection.Address, connection.ViaSSH)
							continue
						}
						fmt.Fprintf(b.out, "  - %s (%s %s %s)\n", connection.Name, connection.Driver, connection.Mode, connection.Address)
					}
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) connectionGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "connection",
		UsageLine: "dbx connection <subcommand>",
		Short:     "Manage saved connections",
		Long:      helpEntries["connection"].body,
		SubCommands: []*cmd.Command{
			b.connectionCreateCommand(),
			b.connectionDoctorCommand(),
			b.connectionEditCommand(),
			b.connectionDeleteCommand(),
			b.connectionShowCommand(),
			b.connectionTestCommand(),
		},
	}
}

func (b *cliBuilder) connectionCreateCommand() *cmd.Command {
	flags := &connectionCreateFlags{driver: "mysql", mode: "direct", port: 3306, sshPort: 22, connectTimeout: 10, queryTimeout: 30}
	return &cmd.Command{
		Name:        "create",
		UsageLine:   "dbx connection create <name> [flags]",
		Short:       "Create a saved connection",
		Long:        helpEntries["connection create"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true}},
		SetFlags: func(f *cmd.FlagSet) {
			f.StringVar(&flags.driver, "driver", "mysql", "database driver", "")
			f.SetEnum("driver", "mysql")
			f.StringVar(&flags.mode, "mode", "direct", "connection mode", "")
			f.SetEnum("mode", "direct", "ssh", "proxy", "proxy-ssh")
			f.StringVar(&flags.host, "host", "", "database host", "")
			f.IntVar(&flags.port, "port", 3306, "database port", "")
			f.StringVar(&flags.user, "user", "", "database user", "")
			f.StringVar(&flags.passwordEnv, "password-env", "", "database password environment variable", "")
			f.StringVar(&flags.password, "password", "", "database password", "")
			f.StringVar(&flags.proxyURL, "proxy-url", "", "SOCKS5 proxy URL for proxy or proxy-ssh mode", "")
			f.StringVar(&flags.sshHost, "ssh-host", "", "SSH host", "")
			f.IntVar(&flags.sshPort, "ssh-port", 22, "SSH port", "")
			f.StringVar(&flags.sshUser, "ssh-user", "", "SSH user", "")
			f.StringVar(&flags.sshPasswordEnv, "ssh-password-env", "", "SSH password environment variable", "")
			f.StringVar(&flags.sshPassword, "ssh-password", "", "SSH password", "")
			f.StringVar(&flags.sshPrivateKey, "ssh-private-key", "", "SSH private key path", "")
			f.IntVar(&flags.connectTimeout, "connect-timeout", 10, "connect timeout in seconds", "")
			f.IntVar(&flags.queryTimeout, "query-timeout", 30, "query timeout in seconds", "")
			f.BoolVar(&flags.test, "test", false, "test the connection before saving", "")
			f.BoolVar(&flags.connectNow, "connect-now", false, "connect immediately after saving", "")
			f.BoolVar(&flags.force, "force", false, "overwrite an existing connection", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "connection create", fmt.Errorf("usage: dbx connection create <name> [flags]"))
			}

			return b.withAuditedApplication(ctx, auditMetadata{Command: "connection create"}, func(application *Application, meta *auditMetadata) error {
				cfg, err := buildCreateConnectionConfig(args[0], flags)
				if err != nil {
					return err
				}
				meta.Connection = cfg.Name
				meta.Mode = cfg.Mode
				if err := b.requireCLIConfirmation("connection create"); err != nil {
					return err
				}
				if application.store.ConnectionExists(cfg.Name) && !flags.force {
					return util.WrapLayer("config", "create connection", fmt.Errorf("connection %q already exists; use --force to overwrite", cfg.Name))
				}

				if err := application.store.SaveConnection(cfg); err != nil {
					return util.WrapLayer("config", "save connection "+cfg.Name, err)
				}
				var testErr error
				if flags.test {
					testErr = application.testConnection(ctx, cfg)
				}
				if flags.connectNow && testErr == nil {
					if err := application.activateConnection(ctx, cfg, true); err != nil {
						return err
					}
					defer application.session.Close()
				}

				result := &ConnectionCreateResult{
					OK:          true,
					Connection:  cfg.Name,
					Saved:       true,
					EditCommand: "connection edit " + cfg.Name,
					Path:        application.store.ConnectionConfigPath(cfg.Name),
				}
				if flags.test {
					ok := testErr == nil
					result.TestOK = &ok
				}
				if testErr != nil {
					result.Warning = "connection test failed"
					fmt.Fprintln(b.err, "Connection test failed:")
					fmt.Fprintf(b.err, "  %v\n", testErr)
				}
				return b.writeOutput(result, func() error {
					if testErr == nil && flags.test {
						fmt.Fprintln(b.out, "Connection successful.")
					}
					application.printSavedConnection(cfg.Name)
					if testErr != nil {
						application.printConnectionEditHint(cfg.Name)
					}
					if flags.connectNow && testErr == nil {
						fmt.Fprintf(b.out, "Connected to %s.\n", cfg.Name)
					}
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) connectionEditCommand() *cmd.Command {
	flags := &connectionEditFlags{}
	return &cmd.Command{
		Name:        "edit",
		UsageLine:   "dbx connection edit <name> [flags]",
		Short:       "Edit a saved connection",
		Long:        helpEntries["connection edit"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true}},
		SetFlags: func(f *cmd.FlagSet) {
			bindOptionalStringFlag(f, &flags.driver, "driver", "database driver")
			if flag, ok := f.Lookup("driver"); ok {
				flag.Enum = []string{"mysql"}
			}
			bindOptionalStringFlag(f, &flags.mode, "mode", "connection mode")
			if flag, ok := f.Lookup("mode"); ok {
				flag.Enum = []string{"direct", "ssh", "proxy", "proxy-ssh"}
			}
			bindOptionalStringFlag(f, &flags.host, "host", "database host")
			bindOptionalIntFlag(f, &flags.port, "port", "database port")
			bindOptionalStringFlag(f, &flags.user, "user", "database user")
			bindOptionalStringFlag(f, &flags.passwordEnv, "password-env", "database password environment variable")
			bindOptionalStringFlag(f, &flags.password, "password", "database password")
			bindOptionalStringFlag(f, &flags.proxyURL, "proxy-url", "SOCKS5 proxy URL for proxy or proxy-ssh mode")
			bindOptionalStringFlag(f, &flags.sshHost, "ssh-host", "SSH host")
			bindOptionalIntFlag(f, &flags.sshPort, "ssh-port", "SSH port")
			bindOptionalStringFlag(f, &flags.sshUser, "ssh-user", "SSH user")
			bindOptionalStringFlag(f, &flags.sshPasswordEnv, "ssh-password-env", "SSH password environment variable")
			bindOptionalStringFlag(f, &flags.sshPassword, "ssh-password", "SSH password")
			bindOptionalStringFlag(f, &flags.sshPrivateKey, "ssh-private-key", "SSH private key path")
			bindOptionalIntFlag(f, &flags.connectTimeout, "connect-timeout", "connect timeout in seconds")
			bindOptionalIntFlag(f, &flags.queryTimeout, "query-timeout", "query timeout in seconds")
			f.BoolVar(&flags.test, "test", false, "test the connection before saving", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "connection edit", fmt.Errorf("usage: dbx connection edit <name> [flags]"))
			}

			return b.withAuditedApplication(ctx, auditMetadata{Command: "connection edit", Connection: args[0]}, func(application *Application, meta *auditMetadata) error {
				cfg, err := application.store.LoadConnection(args[0])
				if err != nil {
					return util.WrapLayer("config", "load connection "+args[0], err)
				}
				meta.Mode = cfg.Mode
				if err := b.requireCLIConfirmation("connection edit"); err != nil {
					return err
				}

				if err := applyEditConnectionFlags(cfg, flags); err != nil {
					return err
				}
				meta.Mode = cfg.Mode
				if flags.test {
					if err := application.testConnection(ctx, cfg); err != nil {
						return err
					}
				}
				if err := application.store.SaveConnection(cfg); err != nil {
					return util.WrapLayer("config", "save connection "+cfg.Name, err)
				}

				result := &ConnectResult{
					OK:         true,
					Connection: cfg.Name,
					Message:    application.store.ConnectionConfigPath(cfg.Name),
				}
				return b.writeOutput(result, func() error {
					if flags.test {
						fmt.Fprintln(b.out, "Connection successful.")
					}
					fmt.Fprintf(b.out, "Saved: %s\n", application.store.ConnectionConfigPath(cfg.Name))
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) connectionDeleteCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "delete",
		UsageLine:   "dbx connection delete <name> [flags]",
		Short:       "Delete a saved connection",
		Long:        helpEntries["connection delete"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "connection delete", fmt.Errorf("usage: dbx connection delete <name>"))
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "connection delete", Connection: args[0]}, func(application *Application, meta *auditMetadata) error {
				if cfg, err := application.store.LoadConnection(args[0]); err == nil {
					meta.Mode = cfg.Mode
				}
				if err := b.requireCLIConfirmation("connection delete"); err != nil {
					return err
				}
				if err := application.deleteConnectionByName(args[0]); err != nil {
					return err
				}
				return b.writeOutput(&ConnectResult{OK: true, Connection: args[0], Message: "deleted"}, func() error {
					fmt.Fprintf(b.out, "Deleted connection %s.\n", args[0])
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) connectionShowCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "show",
		UsageLine:   "dbx connection show <name>",
		Short:       "Show a saved connection",
		Long:        helpEntries["connection show"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "connection show", fmt.Errorf("usage: dbx connection show <name>"))
			}

			return b.withAuditedApplication(ctx, auditMetadata{Command: "connection show", Connection: args[0]}, func(application *Application, meta *auditMetadata) error {
				result, err := application.showConnection(args[0])
				if err != nil {
					return err
				}
				meta.Mode = result.Mode
				return b.writeOutput(result, func() error {
					fmt.Fprintf(b.out, "Name: %s\n", result.Name)
					fmt.Fprintf(b.out, "Driver: %s\n", result.Driver)
					fmt.Fprintf(b.out, "Mode: %s\n\n", result.Mode)
					fmt.Fprintf(b.out, "Host: %s:%d\n", result.Host, result.Port)
					fmt.Fprintf(b.out, "User: %s\n", result.User)
					fmt.Fprintf(b.out, "Connect timeout: %d\n", result.ConnectTimeout)
					fmt.Fprintf(b.out, "Query timeout: %d\n\n", result.QueryTimeout)
					fmt.Fprintln(b.out, "Password:")
					switch result.Password.Mode {
					case "env":
						fmt.Fprintf(b.out, "  env: %s\n", result.Password.Env)
					case "saved":
						fmt.Fprintf(b.out, "  saved: %s\n", result.Password.Value)
					case "prompt":
						fmt.Fprintln(b.out, "  prompt every time")
					default:
						fmt.Fprintln(b.out, "  not configured")
					}
					if result.Proxy != nil {
						fmt.Fprintln(b.out)
						fmt.Fprintln(b.out, "Proxy:")
						fmt.Fprintf(b.out, "  url: %s\n", result.Proxy.URL)
					}
					if result.SSH != nil {
						fmt.Fprintln(b.out)
						fmt.Fprintln(b.out, "SSH:")
						fmt.Fprintf(b.out, "  host: %s:%d\n", result.SSH.Host, result.SSH.Port)
						fmt.Fprintf(b.out, "  user: %s\n", result.SSH.User)
						if result.SSH.PrivateKey != "" {
							fmt.Fprintf(b.out, "  private_key: %s\n", result.SSH.PrivateKey)
						}
						if result.SSH.PasswordEnv != "" {
							fmt.Fprintf(b.out, "  password_env: %s\n", result.SSH.PasswordEnv)
						} else if result.SSH.PasswordMode == "saved" {
							fmt.Fprintln(b.out, "  password: [redacted]")
						}
					}
					return nil
				})
			})
		},
	}
}

func (b *cliBuilder) connectionTestCommand() *cmd.Command {
	var verbose bool
	return &cmd.Command{
		Name:        "test",
		UsageLine:   "dbx connection test <name>",
		Short:       "Test a saved connection",
		Long:        helpEntries["connection test"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true}},
		SetFlags: func(f *cmd.FlagSet) {
			f.BoolVar(&verbose, "verbose", false, "include per-layer diagnostic details", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "connection test", fmt.Errorf("usage: dbx connection test <name>"))
			}

			return b.withAuditedApplication(ctx, auditMetadata{Command: "connection test", Connection: args[0]}, func(application *Application, meta *auditMetadata) error {
				cfg, err := application.store.LoadConnection(args[0])
				if err != nil {
					return util.WrapLayer("config", "load connection "+args[0], err)
				}
				meta.Mode = cfg.Mode

				result, diagErr := application.diagnoseConnection(ctx, cfg, diagnosticOptions{
					Verbose:    verbose,
					ConfigPath: application.store.ConnectionConfigPath(args[0]),
				})
				if diagErr != nil {
					result.Error = errorResult(diagErr)
				}
				if writeErr := b.writeOutput(result, func() error {
					application.printDiagnosticResult(result, verbose)
					if diagErr == nil {
						fmt.Fprintln(b.out, "Connection successful.")
						return nil
					}
					fmt.Fprintln(b.err, diagErr.Error())
					return nil
				}); writeErr != nil {
					return writeErr
				}
				if diagErr != nil {
					failed := false
					meta.Success = &failed
					return util.MarkOutputHandled(diagErr)
				}
				succeeded := true
				meta.Success = &succeeded
				return nil
			})
		},
	}
}

func (b *cliBuilder) connectionDoctorCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "doctor",
		UsageLine:   "dbx connection doctor <name>",
		Short:       "Inspect a saved connection statically",
		Long:        helpEntries["connection doctor"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if len(args) != 1 {
				return util.WrapLayer("validation", "connection doctor", fmt.Errorf("usage: dbx connection doctor <name>"))
			}

			return b.withAuditedApplication(ctx, auditMetadata{Command: "connection doctor", Connection: args[0]}, func(application *Application, meta *auditMetadata) error {
				if cfg, err := application.store.LoadConnection(args[0]); err == nil {
					meta.Mode = cfg.Mode
				}
				result, doctorErr := application.doctorConnection(args[0])
				if doctorErr != nil {
					result.Error = errorResult(doctorErr)
				}
				if writeErr := b.writeOutput(result, func() error {
					application.printDoctorResult(result)
					return nil
				}); writeErr != nil {
					return writeErr
				}
				if doctorErr != nil {
					failed := false
					meta.Success = &failed
					return util.MarkOutputHandled(doctorErr)
				}
				succeeded := true
				meta.Success = &succeeded
				return nil
			})
		},
	}
}

func buildCreateConnectionConfig(name string, flags *connectionCreateFlags) (*config.ConnectionConfig, error) {
	if err := util.ValidateIdentifier(name); err != nil {
		return nil, util.WrapLayer("validation", "validate connection name", err)
	}

	cfg := &config.ConnectionConfig{
		Name:        name,
		Driver:      strings.TrimSpace(flags.driver),
		Mode:        strings.TrimSpace(flags.mode),
		Host:        strings.TrimSpace(flags.host),
		Port:        flags.port,
		User:        strings.TrimSpace(flags.user),
		PasswordEnv: strings.TrimSpace(flags.passwordEnv),
		Password:    flags.password,
		Timeout: &config.TimeoutConfig{
			ConnectSeconds: flags.connectTimeout,
			QuerySeconds:   flags.queryTimeout,
		},
	}
	if strings.TrimSpace(cfg.Password) != "" {
		cfg.PasswordEnv = ""
	}
	if strings.TrimSpace(flags.proxyURL) != "" {
		cfg.Proxy = &config.ProxyConfig{
			URL: strings.TrimSpace(flags.proxyURL),
		}
	}
	if cfg.UsesSSH() || createFlagsIncludeSSH(flags) {
		cfg.SSH = &config.SSHConfig{
			Host:        strings.TrimSpace(flags.sshHost),
			Port:        flags.sshPort,
			User:        strings.TrimSpace(flags.sshUser),
			PrivateKey:  strings.TrimSpace(flags.sshPrivateKey),
			PasswordEnv: strings.TrimSpace(flags.sshPasswordEnv),
			Password:    flags.sshPassword,
		}
		if strings.TrimSpace(cfg.SSH.Password) != "" {
			cfg.SSH.PasswordEnv = ""
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, util.WrapLayer("validation", "validate connection config", err)
	}
	return cfg, nil
}

func applyEditConnectionFlags(cfg *config.ConnectionConfig, flags *connectionEditFlags) error {
	if flags.driver.Set {
		cfg.Driver = flags.driver.Value
	}
	if flags.mode.Set {
		cfg.Mode = flags.mode.Value
		if cfg.Mode == "direct" {
			cfg.Proxy = nil
			cfg.SSH = nil
		}
		if cfg.Mode == "ssh" {
			cfg.Proxy = nil
		}
		if cfg.Mode == "proxy" {
			cfg.SSH = nil
		}
		if cfg.Mode == "proxy" && cfg.Proxy == nil {
			cfg.Proxy = &config.ProxyConfig{}
		}
		if cfg.Mode == "proxy-ssh" && cfg.Proxy == nil {
			cfg.Proxy = &config.ProxyConfig{}
		}
		if cfg.UsesSSH() && cfg.SSH == nil {
			cfg.SSH = &config.SSHConfig{}
		}
	}
	if flags.host.Set {
		cfg.Host = flags.host.Value
	}
	if flags.port.Set {
		cfg.Port = flags.port.Value
	}
	if flags.user.Set {
		cfg.User = flags.user.Value
	}
	if flags.passwordEnv.Set {
		cfg.PasswordEnv = flags.passwordEnv.Value
		cfg.Password = ""
		cfg.PasswordPrompt = false
	}
	if flags.password.Set {
		cfg.Password = flags.password.Value
		cfg.PasswordEnv = ""
		cfg.PasswordPrompt = false
	}
	if flags.proxyURL.Set {
		if cfg.Proxy == nil {
			cfg.Proxy = &config.ProxyConfig{}
		}
		cfg.Proxy.URL = flags.proxyURL.Value
	}
	if flags.connectTimeout.Set || flags.queryTimeout.Set {
		cfg.ApplyDefaults()
	}
	if flags.connectTimeout.Set {
		cfg.Timeout.ConnectSeconds = flags.connectTimeout.Value
	}
	if flags.queryTimeout.Set {
		cfg.Timeout.QuerySeconds = flags.queryTimeout.Value
	}

	if cfg.UsesSSH() {
		if cfg.SSH == nil {
			cfg.SSH = &config.SSHConfig{}
		}
		if flags.sshHost.Set {
			cfg.SSH.Host = flags.sshHost.Value
		}
		if flags.sshPort.Set {
			cfg.SSH.Port = flags.sshPort.Value
		}
		if flags.sshUser.Set {
			cfg.SSH.User = flags.sshUser.Value
		}
		if flags.sshPrivateKey.Set {
			cfg.SSH.PrivateKey = flags.sshPrivateKey.Value
			if cfg.SSH.PrivateKey != "" {
				cfg.SSH.PasswordEnv = ""
				cfg.SSH.Password = ""
			}
		}
		if flags.sshPasswordEnv.Set {
			cfg.SSH.PasswordEnv = flags.sshPasswordEnv.Value
			cfg.SSH.PrivateKey = ""
			cfg.SSH.Password = ""
		}
		if flags.sshPassword.Set {
			cfg.SSH.Password = flags.sshPassword.Value
			cfg.SSH.PrivateKey = ""
			cfg.SSH.PasswordEnv = ""
		}
	}
	if cfg.Mode == "proxy" && editFlagsIncludeSSH(flags) {
		return util.WrapLayer("validation", "validate connection config", fmt.Errorf("ssh settings are not supported for proxy mode"))
	}

	if err := cfg.Validate(); err != nil {
		return util.WrapLayer("validation", "validate connection config", err)
	}
	return nil
}

func createFlagsIncludeSSH(flags *connectionCreateFlags) bool {
	return strings.TrimSpace(flags.sshHost) != "" ||
		strings.TrimSpace(flags.sshUser) != "" ||
		strings.TrimSpace(flags.sshPasswordEnv) != "" ||
		strings.TrimSpace(flags.sshPassword) != "" ||
		strings.TrimSpace(flags.sshPrivateKey) != ""
}

func editFlagsIncludeSSH(flags *connectionEditFlags) bool {
	return flags.sshHost.Set ||
		flags.sshPort.Set ||
		flags.sshUser.Set ||
		flags.sshPasswordEnv.Set ||
		flags.sshPassword.Set ||
		flags.sshPrivateKey.Set
}
