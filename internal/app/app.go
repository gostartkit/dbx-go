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
)

type Application struct {
	prompt    *ui.Prompt
	store     *config.Store
	connector *connect.Connector
	templates *tpl.Service
	session   *Session
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

	application := &Application{
		prompt:    ui.NewPrompt(in, out),
		store:     store,
		connector: connect.NewConnector(),
		templates: tpl.NewService(store),
		session:   &Session{},
	}

	if restoreErr := application.restoreSession(context.Background()); restoreErr != nil {
		application.prompt.Printf("Warning: %v\n", restoreErr)
	}

	return application, nil
}

func (a *Application) Run(ctx context.Context) error {
	a.printHelp()
	return repl.New(a.prompt, a.handleLine).Run(ctx)
}

func (a *Application) restoreSession(ctx context.Context) error {
	sessionFile, err := a.store.LoadSession()
	if err != nil {
		return err
	}
	if sessionFile.CurrentConnection == "" {
		return nil
	}

	cfg, err := a.store.LoadConnection(sessionFile.CurrentConnection)
	if err != nil {
		return err
	}

	a.session.Connection = cfg

	db, err := a.connector.Open(ctx, cfg)
	if err != nil {
		return fmt.Errorf("restore connection %q: %w", cfg.Name, err)
	}

	a.session.DB = db
	return nil
}
