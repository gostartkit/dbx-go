package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pkg.gostartkit.com/dbx/internal/config"
)

func TestShowTemplatesListsResolvedScopes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "global.json"), `{
  "name": "readonly_user",
  "description": "global readonly user",
  "match": {"command": "create user", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "global", "sql": "GLOBAL"}]
}`)
	if err := os.MkdirAll(store.ConnectionTemplatesDir("prod"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.ConnectionTemplatesDir("prod"), "connection.json"), `{
  "name": "prod_app_database",
  "description": "connection template",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "connection", "sql": "CONNECTION"}]
}`)

	app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{})
	result, err := app.showTemplatesResult(app.session.Connection, templateListFilters{})
	if err != nil {
		t.Fatalf("showTemplatesResult returned error: %v", err)
	}
	if len(result.Templates) == 0 {
		t.Fatalf("expected resolved templates")
	}

	foundGlobal := false
	foundConnection := false
	for _, candidate := range result.Templates {
		switch candidate.Name {
		case "readonly_user":
			foundGlobal = candidate.Scope == "global"
		case "prod_app_database":
			foundConnection = candidate.Scope == "connection"
		}
	}
	if !foundGlobal {
		t.Fatalf("global template missing from result: %+v", result.Templates)
	}
	if !foundConnection {
		t.Fatalf("connection template missing from result: %+v", result.Templates)
	}

	app.printTemplatesCatalog(result)
	if !strings.Contains(out.String(), "Templates:") || !strings.Contains(out.String(), "prod_app_database") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestConnectionTemplateOverridesGlobalTemplateByName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.ConnectionTemplatesDir("prod"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "global.json"), `{
  "name": "shared_workflow",
  "description": "global version",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "global", "sql": "GLOBAL"}]
}`)
	writeTemplate(t, filepath.Join(store.ConnectionTemplatesDir("prod"), "connection.json"), `{
  "name": "shared_workflow",
  "description": "connection version",
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "connection", "sql": "CONNECTION"}]
}`)

	app, _ := newReadOnlyTestApp(t, root, &readOnlyConnector{})
	result, err := app.showTemplatesResult(app.session.Connection, templateListFilters{})
	if err != nil {
		t.Fatalf("showTemplatesResult returned error: %v", err)
	}

	found := 0
	for _, candidate := range result.Templates {
		if candidate.Name != "shared_workflow" {
			continue
		}
		found++
		if candidate.Scope != "connection" {
			t.Fatalf("Scope = %q, want connection", candidate.Scope)
		}
	}
	if found != 1 {
		t.Fatalf("shared_workflow count = %d, want 1", found)
	}
}

func TestDescribeTemplateTextAndVerboseRedaction(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "create_database_with_user.json"), `{
  "name": "create_database_with_user",
  "description": "Create database, user, and grants",
  "match": {"command": "create database", "driver": "mysql"},
  "inputs": [
    {"name": "database", "type": "identifier", "prompt": "Database"},
    {"name": "password", "type": "secret", "description": "Database password", "prompt": "Password"}
  ],
  "actions": [
    {"type": "sql", "description": "Create user", "sql": "CREATE USER '{{database}}'@'%' IDENTIFIED BY '{{password}}'"}
  ]
}`)

	app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{})
	result, err := app.describeTemplateResult(&config.ConnectionConfig{Driver: "mysql"}, "create_database_with_user", true)
	if err != nil {
		t.Fatalf("describeTemplateResult returned error: %v", err)
	}
	if len(result.Inputs) != 2 || len(result.Actions) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.Contains(result.Actions[0].SQL, "***") {
		t.Fatalf("expected redacted SQL, got %q", result.Actions[0].SQL)
	}

	app.printTemplateDescription(result, true)
	if !strings.Contains(out.String(), "Template: create_database_with_user") {
		t.Fatalf("missing template heading: %q", out.String())
	}
	if !strings.Contains(out.String(), "Create user") || !strings.Contains(out.String(), "***") {
		t.Fatalf("missing verbose redacted SQL: %q", out.String())
	}
}

func TestCLIShowTemplateCommandRemoved(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "readonly_user.json"), `{
  "name": "readonly_user",
  "description": "Create readonly user",
  "match": {"command": "create user", "driver": "mysql"},
  "inputs": [{"name": "password", "type": "secret", "prompt": "Password"}],
  "actions": [{"type": "sql", "description": "Create user", "sql": "CREATE USER 'ro' IDENTIFIED BY '{{password}}'"}]
}`)

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"show", "template", "readonly_user", "--format", "json", "--config-dir", root})
	if err == nil {
		t.Fatalf("expected removed command failure")
	}
	_ = stdout
	_ = stderr
}

func TestTemplateRunPreviewParsesInputsAndRedactsSecrets(t *testing.T) {
	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "create_database_with_user.json"), `{
  "name": "create_database_with_user",
  "description": "Create database, same-name user, and grants",
  "match": {"command": "create database", "driver": "mysql"},
  "inputs": [
    {"name": "database", "type": "identifier", "prompt": "Database"},
    {"name": "password", "type": "secret", "prompt": "Password"}
  ],
  "actions": [
    {"type": "sql", "description": "Create database", "sql": "CREATE DATABASE `+"`{{database}}`"+`"},
    {"type": "sql", "description": "Create user", "sql": "CREATE USER '{{database}}'@'%' IDENTIFIED BY '{{password}}'"}
  ]
}`)
	t.Setenv("APP_PASSWORD", "super-secret")

	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: failingConnector{openErr: context.DeadlineExceeded},
	})
	err := app.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"--format", "json",
		"run", "create_database_with_user",
		"--input", "database=app_prod",
		"--input", "password-env=APP_PASSWORD",
		"--preview",
		"--verbose",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "super-secret") {
		t.Fatalf("stdout leaked secret: %q", stdout.String())
	}

	var result TemplateRunResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if !result.Preview || result.Template != "create_database_with_user" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Inputs["password"] != "***" {
		t.Fatalf("password input = %q, want ***", result.Inputs["password"])
	}
	if len(result.Actions) != 2 || !strings.Contains(result.Actions[1].SQL, "***") {
		t.Fatalf("unexpected actions: %+v", result.Actions)
	}
}

func TestTemplateRunPreviewAndDryRunDoNotExecute(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "readonly_user.json"), `{
  "name": "readonly_user",
  "match": {"command": "create user", "driver": "mysql"},
  "inputs": [{"name": "username", "type": "string", "prompt": "Username"}],
  "actions": [{"type": "sql", "description": "Create user", "sql": "CREATE USER '{{username}}'"}]
}`)

	previewApp, previewOut, previewErr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: failingConnector{openErr: context.DeadlineExceeded},
	})
	err := previewApp.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"run", "readonly_user",
		"--input", "username=analytics-ro",
		"--preview",
	})
	if err != nil {
		t.Fatalf("preview Run returned error: %v\nstderr=%s", err, previewErr.String())
	}
	if !strings.Contains(previewOut.String(), "Preview only. No actions executed.") {
		t.Fatalf("unexpected preview output: %q", previewOut.String())
	}

	dryRunApp, dryRunOut, dryRunErr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: failingConnector{openErr: context.DeadlineExceeded},
	})
	err = dryRunApp.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"--dry-run",
		"run", "readonly_user",
		"--input", "username=analytics-ro",
	})
	if err != nil {
		t.Fatalf("dry-run Run returned error: %v\nstderr=%s", err, dryRunErr.String())
	}
	if !strings.Contains(dryRunOut.String(), "[DRY-RUN] Create user") {
		t.Fatalf("unexpected dry-run output: %q", dryRunOut.String())
	}
}

func TestTemplateRunRequiresConfirmationInREPLAndCLI(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "create_database_with_user.json"), `{
  "name": "create_database_with_user",
  "match": {"command": "create database", "driver": "mysql"},
  "inputs": [
    {"name": "database", "type": "identifier", "prompt": "Database"},
    {"name": "password", "type": "secret", "prompt": "Password"}
  ],
  "actions": [{"type": "sql", "description": "Create database", "sql": "CREATE DATABASE `+"`{{database}}`"+`"}]
}`)

	var replOut bytes.Buffer
	replApp, err := NewWithOptions(strings.NewReader("app_demo\nsecret123\nn\n"), &replOut, &replOut, Options{
		ConfigDir: root,
		Connector: &readOnlyConnector{},
	})
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	replApp.session.Connection = sampleConnection("prod")
	replApp.session.DB = &sql.DB{}
	if err := replApp.handleTemplateRun(context.Background(), "create_database_with_user", false, false, false); err != nil {
		t.Fatalf("handleTemplateRun returned error: %v", err)
	}
	if !strings.Contains(replOut.String(), "Confirm execution?") || !strings.Contains(replOut.String(), "Cancelled.") {
		t.Fatalf("unexpected REPL output: %q", replOut.String())
	}

	cliApp, _, cliErr := newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	err = cliApp.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"run", "create_database_with_user",
		"--input", "database=app_demo",
		"--input", "password=secret123",
	})
	if err == nil || !strings.Contains(err.Error(), "confirmation required") {
		t.Fatalf("expected confirmation error, got %v\nstderr=%s", err, cliErr.String())
	}
}

func TestTemplateInspectionCommandsDoNotAskConfirmation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "readonly_user.json"), `{
  "name": "readonly_user",
  "match": {"command": "create user", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "Create user", "sql": "CREATE USER 'ro'"}]
}`)

	app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{})
	if err := app.handleShowTemplates(context.Background(), templateListFilters{}); err != nil {
		t.Fatalf("handleShowTemplates returned error: %v", err)
	}
	if strings.Contains(out.String(), "Confirm execution?") {
		t.Fatalf("unexpected confirmation prompt: %q", out.String())
	}

	out.Reset()
	if err := app.handleDescribeTemplate(context.Background(), "readonly_user", false); err != nil {
		t.Fatalf("handleDescribeTemplate returned error: %v", err)
	}
	if strings.Contains(out.String(), "Confirm execution?") {
		t.Fatalf("unexpected confirmation prompt: %q", out.String())
	}
}

func TestTemplateRunRejectsMissingRequiredInputAndExtraArgs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "readonly_user.json"), `{
  "name": "readonly_user",
  "match": {"command": "create user", "driver": "mysql"},
  "inputs": [{"name": "username", "type": "string", "prompt": "Username"}],
  "actions": [{"type": "sql", "description": "Create user", "sql": "CREATE USER '{{username}}'"}]
}`)

	app, _, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	err := app.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"--format", "json",
		"run", "readonly_user",
		"--preview",
	})
	if err == nil || !strings.Contains(err.Error(), "missing required template input") {
		t.Fatalf("expected missing input error, got %v\nstderr=%s", err, stderr.String())
	}

	aliasApp, _, aliasErr := newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	err = aliasApp.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"run", "template", "readonly_user",
	})
	if err == nil || !strings.Contains(err.Error(), `unknown command "template"`) {
		t.Fatalf("expected usage error, got %v\nstderr=%s", err, aliasErr.String())
	}
}

func TestTemplateCommandsUnsupportedVersionFailsClearly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "bad.json"), `{
  "version": 2,
  "name": "bad",
  "match": {"command": "create user", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "bad", "sql": "SELECT 1"}]
}`)

	app, _, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	err := app.Run(context.Background(), []string{"show", "templates", "--config-dir", root})
	if err == nil || !strings.Contains(err.Error(), "unsupported version 2") {
		t.Fatalf("expected unsupported version error, got %v\nstderr=%s", err, stderr.String())
	}
}

func TestShowTemplatesCategoryTagsAndFiltering(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "readonly_user.json"), `{
  "name": "readonly_user",
  "description": "Create readonly user",
  "category": "user",
  "tags": ["readonly", "grant"],
  "match": {"command": "create user", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "Create user", "sql": "CREATE USER 'ro'"}]
}`)
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "create_database_with_user.json"), `{
  "name": "create_database_with_user",
  "category": "database",
  "tags": ["grant", "tenant"],
  "match": {"command": "create database", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "Create database", "sql": "CREATE DATABASE demo"}]
}`)

	app, out := newReadOnlyTestApp(t, root, &readOnlyConnector{})

	all, err := app.showTemplatesResult(&config.ConnectionConfig{Driver: "mysql"}, templateListFilters{})
	if err != nil {
		t.Fatalf("showTemplatesResult returned error: %v", err)
	}
	if len(all.Templates) < 2 {
		t.Fatalf("template count = %d, want at least 2", len(all.Templates))
	}
	foundReadonly := false
	foundDatabase := false
	for _, candidate := range all.Templates {
		if candidate.Name == "readonly_user" && candidate.Category == "user" {
			foundReadonly = true
		}
		if candidate.Name == "create_database_with_user" && candidate.Category == "database" {
			foundDatabase = true
		}
	}
	if !foundReadonly || !foundDatabase {
		t.Fatalf("missing categorized templates: %+v", all.Templates)
	}

	app.printTemplatesCatalog(all)
	if !strings.Contains(out.String(), "database") || !strings.Contains(out.String(), "[grant,tenant]") {
		t.Fatalf("unexpected catalog output: %q", out.String())
	}

	tagged, err := app.showTemplatesResult(&config.ConnectionConfig{Driver: "mysql"}, templateListFilters{Tag: "readonly"})
	if err != nil {
		t.Fatalf("tagged showTemplatesResult returned error: %v", err)
	}
	if len(tagged.Templates) != 1 || tagged.Templates[0].Name != "readonly_user" {
		t.Fatalf("unexpected tag filter result: %+v", tagged.Templates)
	}

	searched, err := app.showTemplatesResult(&config.ConnectionConfig{Driver: "mysql"}, templateListFilters{Query: "database"})
	if err != nil {
		t.Fatalf("searched showTemplatesResult returned error: %v", err)
	}
	foundSearched := false
	for _, candidate := range searched.Templates {
		if candidate.Name == "create_database_with_user" {
			foundSearched = true
		}
	}
	if !foundSearched {
		t.Fatalf("unexpected search result: %+v", searched.Templates)
	}
}

func TestTemplateCategoryDefaultsToCustom(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "readonly_user.json"), `{
  "name": "readonly_user",
  "match": {"command": "create user", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "Create user", "sql": "CREATE USER 'ro'"}]
}`)

	app, _ := newReadOnlyTestApp(t, root, &readOnlyConnector{})
	result, err := app.describeTemplateResult(&config.ConnectionConfig{Driver: "mysql"}, "readonly_user", false)
	if err != nil {
		t.Fatalf("describeTemplateResult returned error: %v", err)
	}
	if result.Category != "custom" {
		t.Fatalf("Category = %q, want custom", result.Category)
	}
}

func TestTemplateRunPreviewShowsRedactedInputSummaryAndCategory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConnection(sampleConnection("prod")); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "create_database_with_user.json"), `{
  "name": "create_database_with_user",
  "category": "database",
  "match": {"command": "create database", "driver": "mysql"},
  "inputs": [
    {"name": "database", "type": "identifier", "prompt": "Database"},
    {"name": "environment", "type": "select", "default": "prod", "options": ["dev", "prod"]},
    {"name": "password", "type": "secret", "prompt": "Password"}
  ],
  "actions": [{"type": "sql", "description": "Create database", "sql": "CREATE DATABASE `+"`{{database}}`"+`"}]
}`)

	app, stdout, stderr := newCLIAppWithOptions(t, "", Options{
		ConfigDir: root,
		Connector: failingConnector{openErr: context.DeadlineExceeded},
	})
	err := app.Run(context.Background(), []string{
		"--config-dir", root,
		"--connection", "prod",
		"run", "create_database_with_user",
		"--input", "database=greenhn_prod",
		"--input", "password=super-secret",
		"--preview",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	output := stdout.String()
	if strings.Contains(output, "super-secret") {
		t.Fatalf("preview leaked secret: %q", output)
	}
	if !strings.Contains(output, "Scope: global") || !strings.Contains(output, "Category: database") {
		t.Fatalf("missing plan heading: %q", output)
	}
	if !strings.Contains(output, "password: [REDACTED]") {
		t.Fatalf("missing redacted password summary: %q", output)
	}
	if !strings.Contains(output, "environment: prod (default)") {
		t.Fatalf("missing default marker: %q", output)
	}
}

func TestTemplateValidateSuccessAndJSONOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "create_database_with_user.json"), `{
  "name": "create_database_with_user",
  "category": "database",
  "tags": ["readonly"],
  "match": {"command": "create database", "driver": "mysql"},
  "inputs": [{"name": "database", "type": "identifier", "prompt": "Database"}],
  "actions": [{"type": "sql", "description": "Create database", "sql": "CREATE DATABASE demo"}]
}`)

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"run", "create_database_with_user", "--validate", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	var result TemplateValidationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if !result.Valid || result.Category != "database" || result.Command != "create database" {
		t.Fatalf("unexpected validation result: %+v", result)
	}
}

func TestTemplateValidateFailures(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "bad.json"), `{
  "name": "bad",
  "match": {"command": "not real", "driver": "mysql"},
  "inputs": [{"name": "role", "type": "select", "prompt": "Role"}],
  "actions": [{"type": "sql", "description": "Bad", "sql": "SELECT 1"}]
}`)

	app, _, stderr := newCLIAppWithOptions(t, "", Options{ConfigDir: root})
	err := app.Run(context.Background(), []string{"run", "bad", "--validate", "--config-dir", root})
	if err == nil || (!strings.Contains(err.Error(), "unsupported match command") && !strings.Contains(err.Error(), "select input")) {
		t.Fatalf("expected validation failure, got %v\nstderr=%s", err, stderr.String())
	}
}

func TestShowTemplatesJSONIncludesCategoryAndTags(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := config.NewStore(root)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(store.GlobalTemplatesDir(), "readonly_user.json"), `{
  "name": "readonly_user",
  "category": "user",
  "tags": ["readonly", "grant"],
  "match": {"command": "create user", "driver": "mysql"},
  "actions": [{"type": "sql", "description": "Create user", "sql": "CREATE USER ro"}]
}`)

	app, stdout, stderr := newCLIApp(t, "", root)
	err := app.Run(context.Background(), []string{"show", "templates", "--tag", "readonly", "--format", "json", "--config-dir", root})
	if err != nil {
		t.Fatalf("Run returned error: %v\nstderr=%s", err, stderr.String())
	}
	var result TemplatesCatalogResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", err, stdout.String())
	}
	if len(result.Templates) != 1 {
		t.Fatalf("template count = %d, want 1", len(result.Templates))
	}
	if result.Templates[0].Category != "user" || len(result.Templates[0].Tags) != 2 {
		t.Fatalf("unexpected template JSON: %+v", result.Templates[0])
	}
}
