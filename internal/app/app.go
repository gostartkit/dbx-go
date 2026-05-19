package app

import (
	"context"
	"fmt"
	"io"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/connect"
	"pkg.gostartkit.com/dbx/internal/repl"
	tpl "pkg.gostartkit.com/dbx/internal/template"
	"pkg.gostartkit.com/dbx/internal/ui"
	"pkg.gostartkit.com/dbx/internal/util"
)

type Application struct {
	prompt             *ui.Prompt
	store              *config.Store
	connector          *connect.Connector
	templates          *tpl.Service
	session            *Session
	history            []string
	dryRun             bool
	reconnectCandidate *config.ConnectionConfig
}

func New(in io.Reader, out io.Writer, _ io.Writer) (*Application, error) {
	rootDir, err := config.DefaultRootDir()
	if err != nil {
		return nil, err
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
		connector: connect.NewConnector(),
		templates: tpl.NewService(store),
		session:   &Session{},
		history:   history,
	}

	if loadErr := application.loadReconnectCandidate(); loadErr != nil {
		application.prompt.Printf("Warning: %v\n", loadErr)
	}

	return application, nil
}

func (a *Application) Run(ctx context.Context) error {
	if err := a.maybeReconnect(ctx); err != nil {
		return err
	}
	a.printHelp()
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

	db, err := a.connector.Open(ctx, a.reconnectCandidate)
	if err != nil {
		a.prompt.Printf("Warning: %v\n", util.WrapLayer("mysql", "reconnect previous session "+a.reconnectCandidate.Name, err))
		a.reconnectCandidate = nil
		return nil
	}

	a.session.Connection = a.reconnectCandidate
	a.session.DB = db
	a.prompt.Printf("Reconnected to %s.\n", a.reconnectCandidate.Name)
	a.reconnectCandidate = nil
	return nil
}

func (a *Application) recordHistory(line string) error {
	if err := a.store.AppendHistory(line); err != nil {
		return util.WrapLayer("config", "persist history", err)
	}
	a.history = append(a.history, line)
	if len(a.history) > 1000 {
		a.history = append([]string(nil), a.history[len(a.history)-1000:]...)
	}
	return nil
}
