package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"pkg.gostartkit.com/cmd"
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
	prompt                      *ui.Prompt
	out                         io.Writer
	store                       *config.Store
	connector                   connectorClient
	templates                   *tpl.Service
	replApp                     *cmd.App
	session                     *Session
	history                     []string
	dryRun                      bool
	reconnectCandidate          *config.ConnectionConfig
	reconnectDatabase           string
	completionDBs               []string
	completionDBsConn           string
	completionTables            []string
	completionTablesConn        string
	completionTablesDB          string
	completionUsers             []string
	completionUsersConn         string
	completionMu                sync.Mutex
	completionDBsReady          bool
	completionUsersReady        bool
	completionTablesReady       bool
	completionDBsLoadingConn    string
	completionUsersLoadingConn  string
	completionTablesLoadingConn string
	completionTablesLoadingDB   string
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
		out:       out,
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

func (a *Application) Run(ctx context.Context) error {
	if err := a.maybeReconnect(ctx); err != nil {
		return err
	}
	return repl.New(a.prompt, a.promptLabel, a.handleLine).Run(ctx)
}

func (a *Application) Close() error {
	if a == nil {
		return nil
	}

	var errs []error
	if a.session != nil {
		errs = append(errs, a.session.Close())
	}
	if a.store != nil {
		errs = append(errs, a.store.Close())
	}
	return errors.Join(errs...)
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
	a.reconnectDatabase = sessionFile.CurrentDatabase
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
	a.session.Database = ""
	a.clearTableCompletion()
	a.clearUserCompletion()
	if err := a.restoreSessionDatabase(ctx); err != nil {
		a.prompt.Printf("Warning: %v\n", err)
	}
	if err := a.store.SaveSession(&config.SessionFile{CurrentConnection: a.reconnectCandidate.Name, CurrentDatabase: a.session.Database}); err != nil {
		return util.WrapLayer("config", "save session", err)
	}

	a.prompt.Printf("Reconnected to %s.\n", a.reconnectCandidate.Name)
	a.reconnectCandidate = nil
	return nil
}

func (a *Application) promptLabel() string {
	label := "dbx"
	if a.session != nil && a.session.Connection != nil {
		scope := a.session.Connection.Name
		if strings.TrimSpace(a.session.Database) != "" {
			scope += "/" + a.session.Database
		}
		label += "[" + scope + "]"
		if a.session.DB == nil {
			label += "[disconnected]"
		}
	}
	if a.dryRun {
		label += "[dry-run]"
	}
	return label + "> "
}

func (a *Application) restoreSessionDatabase(ctx context.Context) error {
	if a.session == nil || a.session.Connection == nil || a.session.DB == nil {
		return nil
	}
	if strings.TrimSpace(a.reconnectDatabase) == "" {
		return nil
	}
	databases, err := a.connector.ListDatabases(ctx, a.session.Connection, a.session.DB)
	if err != nil {
		a.reconnectDatabase = ""
		return util.WrapLayer("mysql", "validate restored database selection", err)
	}
	for _, database := range databases {
		if database == a.reconnectDatabase {
			a.session.Database = database
			a.reconnectDatabase = ""
			return nil
		}
	}
	stale := a.reconnectDatabase
	a.reconnectDatabase = ""
	return fmt.Errorf("database %q no longer exists; cleared previous database selection", stale)
}

func (a *Application) currentCompletionDatabases() []string {
	cfg, db, _ := a.completionSnapshot()
	if cfg == nil || db == nil {
		return nil
	}
	return a.currentDatabaseCompletionValues(cfg, db)
}

func (a *Application) currentCompletionUsers() []string {
	cfg, db, _ := a.completionSnapshot()
	if cfg == nil || db == nil {
		return nil
	}
	return a.currentUserCompletionValues(cfg, db)
}

func (a *Application) currentCompletionTemplates() []string {
	cfg, err := a.commandContext().resolveTemplateScope("")
	if err != nil {
		return nil
	}
	templates, err := a.templates.ListResolved(cfg)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(templates))
	for _, candidate := range templates {
		names = append(names, candidate.Name)
	}
	return names
}

func (a *Application) currentCompletionTemplateTags() []string {
	cfg, err := a.commandContext().resolveTemplateScope("")
	if err != nil {
		return nil
	}
	templates, err := a.templates.ListResolved(cfg)
	if err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	tags := make([]string, 0)
	for _, candidate := range templates {
		for _, tag := range candidate.EffectiveTags() {
			if _, exists := seen[tag]; exists {
				continue
			}
			seen[tag] = struct{}{}
			tags = append(tags, tag)
		}
	}
	slices.Sort(tags)
	return tags
}

func (a *Application) currentCompletionTables() []string {
	cfg, db, database := a.completionSnapshot()
	if cfg == nil || db == nil || strings.TrimSpace(database) == "" {
		return nil
	}
	return a.currentTableCompletionValues(cfg, db, database)
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

func (a *Application) currentConnectionName() string {
	if a.session == nil || a.session.Connection == nil {
		return ""
	}
	return a.session.Connection.Name
}

func (a *Application) currentDatabaseName() string {
	if a.session == nil {
		return ""
	}
	return a.session.Database
}
