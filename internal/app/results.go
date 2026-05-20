package app

import (
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/driver"
	"pkg.gostartkit.com/dbx/internal/util"
)

type ErrorResult struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Layer   string `json:"layer,omitempty"`
}

type ActionStatus string

const (
	ActionStatusOK     ActionStatus = "ok"
	ActionStatusFailed ActionStatus = "failed"
	ActionStatusDryRun ActionStatus = "dry-run"
)

type ActionResult struct {
	Description string       `json:"description"`
	SQL         string       `json:"sql,omitempty"`
	Status      ActionStatus `json:"status"`
	DurationMS  int64        `json:"duration_ms,omitempty"`
}

type PlanExecutionResult struct {
	OK          bool           `json:"ok"`
	Error       *ErrorResult   `json:"error,omitempty"`
	Connection  string         `json:"connection,omitempty"`
	Command     string         `json:"command,omitempty"`
	Template    string         `json:"template,omitempty"`
	Layer       string         `json:"layer,omitempty"`
	Source      string         `json:"source,omitempty"`
	DryRun      bool           `json:"dry_run,omitempty"`
	Transaction bool           `json:"transaction,omitempty"`
	Committed   bool           `json:"committed,omitempty"`
	RolledBack  bool           `json:"rolled_back,omitempty"`
	Actions     []ActionResult `json:"actions,omitempty"`
}

type ConnectionSummary struct {
	Name     string `json:"name"`
	Driver   string `json:"driver"`
	Mode     string `json:"mode"`
	Address  string `json:"address"`
	ViaProxy string `json:"via_proxy,omitempty"`
	ViaSSH   string `json:"via_ssh,omitempty"`
}

type ConnectionsResult struct {
	OK          bool                `json:"ok"`
	Connections []ConnectionSummary `json:"connections"`
}

type DatabasesResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	Database   string   `json:"database,omitempty"`
	Databases  []string `json:"databases,omitempty"`
}

type TablesResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	Database   string   `json:"database,omitempty"`
	Tables     []string `json:"tables,omitempty"`
}

type TableColumnResult struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Null    string `json:"null,omitempty"`
	Key     string `json:"key,omitempty"`
	Default string `json:"default,omitempty"`
	Extra   string `json:"extra,omitempty"`
}

type TableDescriptionResult struct {
	OK         bool                `json:"ok"`
	Connection string              `json:"connection,omitempty"`
	Database   string              `json:"database,omitempty"`
	Table      string              `json:"table,omitempty"`
	Columns    []TableColumnResult `json:"columns,omitempty"`
}

type SchemaColumnResult struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Key      string `json:"key,omitempty"`
	Extra    string `json:"extra,omitempty"`
}

type ColumnsResult struct {
	OK         bool                 `json:"ok"`
	Connection string               `json:"connection,omitempty"`
	Database   string               `json:"database,omitempty"`
	Table      string               `json:"table,omitempty"`
	Columns    []SchemaColumnResult `json:"columns,omitempty"`
}

type RowCountResult struct {
	OK         bool   `json:"ok"`
	Connection string `json:"connection,omitempty"`
	Database   string `json:"database,omitempty"`
	Table      string `json:"table,omitempty"`
	Rows       int64  `json:"rows"`
}

type RowPreviewResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	Database   string   `json:"database,omitempty"`
	Table      string   `json:"table,omitempty"`
	Columns    []string `json:"columns,omitempty"`
	Rows       [][]any  `json:"rows,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

type TableIndexResult struct {
	Name   string `json:"name"`
	Column string `json:"column"`
	Type   string `json:"type"`
}

type TableIndexesResult struct {
	OK         bool               `json:"ok"`
	Connection string             `json:"connection,omitempty"`
	Database   string             `json:"database,omitempty"`
	Table      string             `json:"table,omitempty"`
	Indexes    []TableIndexResult `json:"indexes,omitempty"`
}

type CreateTableResult struct {
	OK          bool   `json:"ok"`
	Connection  string `json:"connection,omitempty"`
	Database    string `json:"database,omitempty"`
	Table       string `json:"table,omitempty"`
	CreateTable string `json:"create_table,omitempty"`
}

type TableStatusEntryResult struct {
	Name        string `json:"name"`
	Engine      string `json:"engine"`
	Rows        int64  `json:"rows"`
	DataLength  int64  `json:"data_length"`
	IndexLength int64  `json:"index_length"`
	Collation   string `json:"collation,omitempty"`
}

type TableStatusResult struct {
	OK         bool                     `json:"ok"`
	Connection string                   `json:"connection,omitempty"`
	Database   string                   `json:"database,omitempty"`
	Table      string                   `json:"table,omitempty"`
	Tables     []TableStatusEntryResult `json:"tables,omitempty"`
}

type ForeignKeyResult struct {
	Constraint       string `json:"constraint"`
	Column           string `json:"column"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
}

type ForeignKeysResult struct {
	OK          bool               `json:"ok"`
	Connection  string             `json:"connection,omitempty"`
	Database    string             `json:"database,omitempty"`
	Table       string             `json:"table,omitempty"`
	ForeignKeys []ForeignKeyResult `json:"foreign_keys,omitempty"`
}

type GrantsResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	User       string   `json:"user,omitempty"`
	Host       string   `json:"host,omitempty"`
	Grants     []string `json:"grants,omitempty"`
}

type UsersResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	Users      []string `json:"users,omitempty"`
}

type ProcessResult struct {
	ID          int64  `json:"id"`
	User        string `json:"user"`
	Host        string `json:"host"`
	Database    string `json:"database,omitempty"`
	Command     string `json:"command"`
	TimeSeconds int64  `json:"time_seconds"`
	State       string `json:"state,omitempty"`
	Info        string `json:"info,omitempty"`
}

type ProcesslistResult struct {
	OK         bool            `json:"ok"`
	Connection string          `json:"connection,omitempty"`
	Processes  []ProcessResult `json:"processes,omitempty"`
}

type TriggerResult struct {
	Name   string `json:"name"`
	Timing string `json:"timing"`
	Event  string `json:"event"`
	Table  string `json:"table"`
}

type TriggersResult struct {
	OK         bool            `json:"ok"`
	Connection string          `json:"connection,omitempty"`
	Database   string          `json:"database,omitempty"`
	Triggers   []TriggerResult `json:"triggers,omitempty"`
}

type VariableResult struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type VariablesResult struct {
	OK         bool             `json:"ok"`
	Connection string           `json:"connection,omitempty"`
	Pattern    string           `json:"pattern,omitempty"`
	Variables  []VariableResult `json:"variables,omitempty"`
}

type ViewsResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	Database   string   `json:"database,omitempty"`
	Views      []string `json:"views,omitempty"`
}

type TableMutationResult struct {
	OK     bool   `json:"ok"`
	Table  string `json:"table,omitempty"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
	Action string `json:"action,omitempty"`
}

type UserMutationResult struct {
	OK       bool   `json:"ok"`
	User     string `json:"user,omitempty"`
	Host     string `json:"host,omitempty"`
	Grant    string `json:"grant,omitempty"`
	Database string `json:"database,omitempty"`
}

type ConnectResult struct {
	OK         bool   `json:"ok"`
	Connection string `json:"connection,omitempty"`
	Message    string `json:"message,omitempty"`
}

type ConnectionCreateResult struct {
	OK          bool   `json:"ok"`
	Connection  string `json:"connection,omitempty"`
	Saved       bool   `json:"saved"`
	TestOK      *bool  `json:"test_ok,omitempty"`
	Warning     string `json:"warning,omitempty"`
	EditCommand string `json:"edit_command,omitempty"`
	Path        string `json:"path,omitempty"`
}

type DiagnosticStep struct {
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	Error   string         `json:"error,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type DiagnosticResult struct {
	OK         bool             `json:"ok"`
	Error      *ErrorResult     `json:"error,omitempty"`
	Connection string           `json:"connection"`
	Steps      []DiagnosticStep `json:"steps"`
}

type DoctorCheck struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Suggestion string `json:"suggestion,omitempty"`
}

type DoctorResult struct {
	OK         bool          `json:"ok"`
	Error      *ErrorResult  `json:"error,omitempty"`
	Connection string        `json:"connection"`
	Checks     []DoctorCheck `json:"checks"`
}

type AuditLogResult struct {
	OK      bool                 `json:"ok"`
	Entries []config.AuditRecord `json:"entries"`
}

type ErrorEnvelope struct {
	OK    bool         `json:"ok"`
	Error *ErrorResult `json:"error"`
}

type RedactedConnection struct {
	Name           string               `json:"name"`
	Driver         string               `json:"driver"`
	Mode           string               `json:"mode"`
	Host           string               `json:"host"`
	Port           int                  `json:"port"`
	User           string               `json:"user"`
	ConnectTimeout int                  `json:"connect_timeout_seconds"`
	QueryTimeout   int                  `json:"query_timeout_seconds"`
	Password       RedactedPassword     `json:"password"`
	Proxy          *RedactedProxyConfig `json:"proxy,omitempty"`
	SSH            *RedactedSSHSettings `json:"ssh,omitempty"`
}

type RedactedPassword struct {
	Mode  string `json:"mode"`
	Env   string `json:"env,omitempty"`
	Value string `json:"value,omitempty"`
}

type RedactedSSHSettings struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	PrivateKey   string `json:"private_key,omitempty"`
	PasswordEnv  string `json:"password_env,omitempty"`
	PasswordMode string `json:"password_mode,omitempty"`
}

type RedactedProxyConfig struct {
	URL string `json:"url"`
}

type StatusResult struct {
	OK                 bool                `json:"ok"`
	Connection         *RedactedConnection `json:"connection,omitempty"`
	ConnectionName     string              `json:"connection_name,omitempty"`
	Database           string              `json:"database,omitempty"`
	CurrentSession     string              `json:"current_session,omitempty"`
	ConnectionExists   bool                `json:"connection_exists,omitempty"`
	SelectedByFlag     bool                `json:"selected_by_flag,omitempty"`
	HasStoredSession   bool                `json:"has_stored_session,omitempty"`
	ConnectedInProcess bool                `json:"connected_in_process,omitempty"`
	DryRun             bool                `json:"dry_run,omitempty"`
	Message            string              `json:"message,omitempty"`
}

type ContextResult struct {
	OK         bool   `json:"ok"`
	Connection string `json:"connection,omitempty"`
	Database   string `json:"database,omitempty"`
	Mode       string `json:"mode,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
}

func summarizeConnection(cfg config.ConnectionConfig) ConnectionSummary {
	summary := ConnectionSummary{
		Name:    cfg.Name,
		Driver:  cfg.Driver,
		Mode:    cfg.Mode,
		Address: cfg.Address(),
	}
	if cfg.Mode == "proxy" || cfg.Mode == "proxy-ssh" {
		if cfg.Proxy != nil {
			summary.ViaProxy = config.RedactProxyURL(cfg.Proxy.URL)
		}
	}
	if cfg.Mode == "ssh" && cfg.SSH != nil {
		summary.ViaSSH = cfg.SSH.Host
	}
	if cfg.Mode == "proxy-ssh" {
		if cfg.SSH != nil {
			summary.ViaSSH = cfg.SSH.Host
		}
	}
	return summary
}

func redactConnection(cfg *config.ConnectionConfig) *RedactedConnection {
	if cfg == nil {
		return nil
	}

	cfg.ApplyDefaults()

	result := &RedactedConnection{
		Name:           cfg.Name,
		Driver:         cfg.Driver,
		Mode:           cfg.Mode,
		Host:           cfg.Host,
		Port:           cfg.Port,
		User:           cfg.User,
		ConnectTimeout: cfg.Timeout.ConnectSeconds,
		QueryTimeout:   cfg.Timeout.QuerySeconds,
		Password:       redactPassword(cfg),
	}

	if cfg.SSH != nil {
		sshSettings := &RedactedSSHSettings{
			Host:       cfg.SSH.Host,
			Port:       cfg.SSH.Port,
			User:       cfg.SSH.User,
			PrivateKey: cfg.SSH.PrivateKey,
		}
		if strings.TrimSpace(cfg.SSH.PasswordEnv) != "" {
			sshSettings.PasswordEnv = cfg.SSH.PasswordEnv
			sshSettings.PasswordMode = "env"
		} else if strings.TrimSpace(cfg.SSH.Password) != "" {
			sshSettings.PasswordMode = "saved"
		}
		result.SSH = sshSettings
	}
	if cfg.Proxy != nil && strings.TrimSpace(cfg.Proxy.URL) != "" {
		result.Proxy = &RedactedProxyConfig{
			URL: config.RedactProxyURL(cfg.Proxy.URL),
		}
	}

	return result
}

func toTableIndexResults(indexes []driver.TableIndex) []TableIndexResult {
	results := make([]TableIndexResult, 0, len(indexes))
	for _, index := range indexes {
		results = append(results, TableIndexResult{
			Name:   index.Name,
			Column: index.Column,
			Type:   index.Type,
		})
	}
	return results
}

func toSchemaColumnResults(columns []driver.SchemaColumn) []SchemaColumnResult {
	results := make([]SchemaColumnResult, 0, len(columns))
	for _, column := range columns {
		results = append(results, SchemaColumnResult{
			Name:     column.Name,
			Type:     column.Type,
			Nullable: column.Nullable,
			Key:      column.Key,
			Extra:    column.Extra,
		})
	}
	return results
}

func toTableStatusResults(statuses []driver.TableStatus) []TableStatusEntryResult {
	results := make([]TableStatusEntryResult, 0, len(statuses))
	for _, status := range statuses {
		results = append(results, TableStatusEntryResult{
			Name:        status.Name,
			Engine:      status.Engine,
			Rows:        status.Rows,
			DataLength:  status.DataLength,
			IndexLength: status.IndexLength,
			Collation:   status.Collation,
		})
	}
	return results
}

func toForeignKeyResults(keys []driver.ForeignKey) []ForeignKeyResult {
	results := make([]ForeignKeyResult, 0, len(keys))
	for _, key := range keys {
		results = append(results, ForeignKeyResult{
			Constraint:       key.Constraint,
			Column:           key.Column,
			ReferencedTable:  key.ReferencedTable,
			ReferencedColumn: key.ReferencedColumn,
		})
	}
	return results
}

func toProcessResults(processes []driver.Process) []ProcessResult {
	results := make([]ProcessResult, 0, len(processes))
	for _, process := range processes {
		results = append(results, ProcessResult{
			ID:          process.ID,
			User:        process.User,
			Host:        process.Host,
			Database:    process.Database,
			Command:     process.Command,
			TimeSeconds: process.TimeSeconds,
			State:       process.State,
			Info:        process.Info,
		})
	}
	return results
}

func toTriggerResults(triggers []driver.Trigger) []TriggerResult {
	results := make([]TriggerResult, 0, len(triggers))
	for _, trigger := range triggers {
		results = append(results, TriggerResult{
			Name:   trigger.Name,
			Timing: trigger.Timing,
			Event:  trigger.Event,
			Table:  trigger.Table,
		})
	}
	return results
}

func toVariableResults(variables []driver.SystemVariable) []VariableResult {
	results := make([]VariableResult, 0, len(variables))
	for _, variable := range variables {
		results = append(results, VariableResult{
			Name:  variable.Name,
			Value: variable.Value,
		})
	}
	return results
}

func redactPassword(cfg *config.ConnectionConfig) RedactedPassword {
	switch {
	case cfg.PasswordPrompt:
		return RedactedPassword{Mode: "prompt"}
	case strings.TrimSpace(cfg.PasswordEnv) != "":
		return RedactedPassword{Mode: "env", Env: cfg.PasswordEnv}
	case strings.TrimSpace(cfg.Password) != "":
		return RedactedPassword{Mode: "saved", Value: "[redacted]"}
	default:
		return RedactedPassword{Mode: "none"}
	}
}

func errorResult(err error) *ErrorResult {
	info := util.DescribeError(err)
	if info == nil {
		return nil
	}
	return &ErrorResult{
		Code:    info.Code,
		Message: info.Message,
		Layer:   info.Layer,
	}
}
