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
	overwrite      bool
}

func (b *cliBuilder) connectCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "connect",
		UsageLine:   "dbx connect <name>",
		Short:       "Connect to a saved connection",
		Long:        helpEntries["connect"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Completion: b.completeConnections}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				switch len(args) {
				case 0:
					return b.application.handleConnect(ctx)
				case 1:
					return b.application.handleConnectByName(ctx, args[0])
				default:
					return util.WrapLayer("validation", "connect", fmt.Errorf("usage: connect [name]"))
				}
			}
			if len(args) == 0 {
				return b.withApplication(ctx, func(application *Application) error {
					if strings.EqualFold(b.globals.Format, "json") {
						return util.WrapLayer("validation", "connect", fmt.Errorf("connect without a name is only supported in text mode"))
					}
					return application.handleConnect(ctx)
				})
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "connect"}, func(application *Application, meta *auditMetadata) error {
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
			if b.mode == ModeREPL {
				if err := b.requireNoArgs(args); err != nil {
					return util.WrapLayer("validation", "connections", err)
				}
				return b.application.handleConnections(ctx)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show connections"}, func(application *Application, meta *auditMetadata) error {
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

func (b *cliBuilder) showConnectionsCommand() *cmd.Command {
	command := b.connectionsCommand()
	command.Name = "connections"
	command.UsageLine = "dbx show connections"
	command.Short = "Show saved connections"
	return command
}

func (b *cliBuilder) showConnectionCommand() *cmd.Command {
	command := b.connectionShowCommand()
	command.Name = "connection"
	command.UsageLine = "dbx show connection <name>"
	command.Short = "Show a saved connection"
	return command
}

func (b *cliBuilder) createConnectionCommand() *cmd.Command {
	command := b.connectionCreateCommand()
	command.Name = "connection"
	command.UsageLine = "dbx create connection <name> [flags]"
	command.Short = "Create a saved connection"
	return command
}

func (b *cliBuilder) dropConnectionCommand() *cmd.Command {
	command := b.connectionDeleteCommand()
	command.Name = "connection"
	command.UsageLine = "dbx drop connection <name> [flags]"
	command.Short = "Drop a saved connection"
	return command
}

func (b *cliBuilder) doctorGroupCommand() *cmd.Command {
	return &cmd.Command{
		Name:      "doctor",
		UsageLine: "dbx doctor",
		Short:     "Inspect the selected connection statically",
		Long:      helpEntries["doctor"].body,
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if err := b.requireNoArgs(args); err != nil {
				return util.WrapLayer("validation", "doctor", err)
			}
			if b.mode == ModeREPL {
				return b.application.handleConnectionDoctor(ctx, "")
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "doctor"}, func(application *Application, meta *auditMetadata) error {
				name := strings.TrimSpace(b.globals.Connection)
				if name == "" {
					sessionFile, err := application.store.LoadSession()
					if err != nil {
						return util.WrapLayer("config", "load session", err)
					}
					name = strings.TrimSpace(sessionFile.CurrentConnection)
				}
				if name == "" {
					return util.WrapLayer("validation", "doctor", fmt.Errorf("no connection selected; use --connection or connect <name> first"))
				}
				if cfg, err := application.store.LoadConnection(name); err == nil {
					meta.Connection = cfg.Name
					meta.Mode = cfg.Mode
				}
				result, doctorErr := application.doctorConnection(name)
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

func (b *cliBuilder) connectionCreateCommand() *cmd.Command {
	flags := &connectionCreateFlags{driver: "mysql", mode: "direct", port: 3306, sshPort: 22, connectTimeout: 10, queryTimeout: 30}
	return &cmd.Command{
		Name:      "create",
		UsageLine: "dbx connection create <name> [flags]",
		Short:     "Create a saved connection",
		Long:      helpEntries["connection create"].body,
		Positionals: b.positionalsForMode(
			[]cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true, Completion: b.completeConnections}},
			nil,
		),
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
			f.BoolVar(&flags.overwrite, "overwrite", false, "overwrite an existing connection", "")
		},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				return b.application.handleConnectionCreate(ctx)
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "create connection"}, func(application *Application, meta *auditMetadata) error {
				cfg, err := buildCreateConnectionConfig(args[0], flags)
				if err != nil {
					return err
				}
				meta.Connection = cfg.Name
				meta.Mode = cfg.Mode
				if err := b.requireCLIConfirmation("create connection"); err != nil {
					return err
				}
				if application.store.ConnectionExists(cfg.Name) && !flags.overwrite {
					return util.WrapLayer("config", "create connection", fmt.Errorf("connection %q already exists; use --overwrite to replace it", cfg.Name))
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
					OK:               true,
					Connection:       cfg.Name,
					Saved:            true,
					OverwriteCommand: "create connection " + cfg.Name + " --overwrite",
					Path:             application.store.ConnectionConfigPath(cfg.Name),
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
						application.printConnectionOverwriteHint(cfg.Name)
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

func (b *cliBuilder) connectionDeleteCommand() *cmd.Command {
	return &cmd.Command{
		Name:        "delete",
		UsageLine:   "dbx connection delete <name> [flags]",
		Short:       "Delete a saved connection",
		Long:        helpEntries["connection delete"].body,
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true, Completion: b.completeConnections}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				return b.application.handleConnectionDelete(ctx, args[0])
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "drop connection", Connection: args[0]}, func(application *Application, meta *auditMetadata) error {
				if cfg, err := application.store.LoadConnection(args[0]); err == nil {
					meta.Mode = cfg.Mode
				}
				if err := b.requireCLIConfirmation("drop connection"); err != nil {
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
		Positionals: []cmd.PositionalArg{{Name: "name", Usage: "saved connection name", Required: true, Completion: b.completeConnections}},
		Run: func(ctx context.Context, _ *cmd.Command, args []string) error {
			if b.mode == ModeREPL {
				return b.application.handleConnectionShow(ctx, args[0])
			}
			return b.withAuditedApplication(ctx, auditMetadata{Command: "show connection", Connection: args[0]}, func(application *Application, meta *auditMetadata) error {
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

func createFlagsIncludeSSH(flags *connectionCreateFlags) bool {
	return strings.TrimSpace(flags.sshHost) != "" ||
		strings.TrimSpace(flags.sshUser) != "" ||
		strings.TrimSpace(flags.sshPasswordEnv) != "" ||
		strings.TrimSpace(flags.sshPassword) != "" ||
		strings.TrimSpace(flags.sshPrivateKey) != ""
}
