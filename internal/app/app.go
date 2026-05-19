package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/repl"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/ui"
	"pkg.gostartkit.com/dbx/internal/util"
)

type Options struct {
	ConfigDir string
	Connector connectorClient
}

type Application struct {
	prompt             *ui.Prompt
	store              *config.Store
	connector          connectorClient
	templates          *tpl.Service
	session            *Session
	history            []string
	dryRun             bool
	reconnectCandidate *config.ConnectionConfig
}

func New(in io.Reader, out io.Writer, _ io.Writer) (*Application, error) {
	return NewWithOptions(in, out, nil, Options{})
}

func NewWithOptions(in io.Reader, out io.Writer, _ io.Writer, opts Options) (*Application, error) {
	rootDir := opts.ConfigDir
	if rootDir == "" {
		var err error
		rootDir, err = config.DefaultRootDir()
		if err != nil {
			return nil, err
		}
	}

	store := config.NewStore(rootDir)
	if err := store.EnsureLayout(); err != nil {
		return nil, err
	}

	history, err := store.LoadHistory()
	if err != nil {
		return nil, util.WrapLayer("config", "load history", err)
	}

	application := &Application{
		prompt:    ui.NewPrompt(in, out),
		store:     store,
		connector: defaultConnector(),
		templates: tpl.NewService(store),
		session:   &Session{},
		history:   history,
	}
	if opts.Connector != nil {
		application.connector = opts.Connector
	}
	application.prompt.SetCompleter(application.completeInput)
	application.prompt.SetHistory(history)

	if loadErr := application.loadReconnectCandidate(); loadErr != nil {
		application.prompt.Printf("Warning: %v\n", loadErr)
	}

	return application, nil
}

func (a *Application) completeInput(line string) ui.Completion {
	connections, err := a.store.ListConnections()
	if err != nil {
		return calculateCompletion(line, nil)
	}

	names := make([]string, 0, len(connections))
	for _, connection := range connections {
		names = append(names, connection.Name)
	}

	return calculateCompletion(line, names)
}

func (a *Application) Run(ctx context.Context) error {
	if err := a.maybeReconnect(ctx); err != nil {
		return err
	}
	return repl.New(a.prompt, a.handleLine).Run(ctx)
}

func (a *Application) Close() error {
	return a.session.Close()
}

func (a *Application) loadReconnectCandidate() error {
	sessionFile, err := a.store.LoadSession()
	if err != nil {
		return util.WrapLayer("config", "load session", err)
	}
	if sessionFile.CurrentConnection == "" {
		return nil
	}
	if !a.store.ConnectionExists(sessionFile.CurrentConnection) {
		if err := a.store.SaveSession(&config.SessionFile{}); err != nil {
			return util.WrapLayer("config", "clear stale session", err)
		}
		return fmt.Errorf("config error: validate previous session: connection %q no longer exists", sessionFile.CurrentConnection)
	}

	cfg, err := a.store.LoadConnection(sessionFile.CurrentConnection)
	if err != nil {
		return util.WrapLayer("config", "load previous session connection "+sessionFile.CurrentConnection, err)
	}

	a.reconnectCandidate = cfg
	return nil
}

func (a *Application) maybeReconnect(ctx context.Context) error {
	if a.reconnectCandidate == nil {
		return nil
	}

	confirmed, err := a.confirm(ctx, fmt.Sprintf("Reconnect previous session %q?", a.reconnectCandidate.Name), true)
	if err != nil {
		return err
	}
	if !confirmed {
		a.reconnectCandidate = nil
		return nil
	}

	runtimeCfg, err := a.prepareConnectionForOpen(ctx, a.reconnectCandidate)
	if err != nil {
		return err
	}

	db, err := a.connector.Open(ctx, runtimeCfg)
	if err != nil {
		a.prompt.Printf("Warning: %v\n", util.WrapLayer("mysql", "reconnect previous session "+a.reconnectCandidate.Name, err))
		a.reconnectCandidate = nil
		return nil
	}

	if err := a.session.Close(); err != nil {
		db.Close()
		return err
	}

	a.session.Connection = cloneConnectionConfig(a.reconnectCandidate)
	a.session.DB = db
	if err := a.store.SaveSession(&config.SessionFile{CurrentConnection: a.reconnectCandidate.Name}); err != nil {
		return util.WrapLayer("config", "save session", err)
	}

	a.prompt.Printf("Reconnected to %s.\n", a.reconnectCandidate.Name)
	a.reconnectCandidate = nil
	return nil
}

func (a *Application) recordHistory(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if len(a.history) > 0 && a.history[len(a.history)-1] == line {
		a.prompt.AppendHistory(line)
		return nil
	}
	if err := a.store.AppendHistory(line); err != nil {
		return util.WrapLayer("config", "persist history", err)
	}
	a.history = append(a.history, line)
	a.prompt.AppendHistory(line)
	if len(a.history) > 1000 {
		a.history = append([]string(nil), a.history[len(a.history)-1000:]...)
	}
	return nil
}
