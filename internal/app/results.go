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
	ActionStatusOK      ActionStatus = "ok"
	ActionStatusFailed  ActionStatus = "failed"
	ActionStatusDryRun  ActionStatus = "dry-run"
	ActionStatusPreview ActionStatus = "preview"
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

type TemplateSummaryResult struct {
	Name        string   `json:"name"`
	Scope       string   `json:"scope"`
	Category    string   `json:"category"`
	Command     string   `json:"command"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type TemplatesCatalogResult struct {
	OK         bool                    `json:"ok"`
	Connection string                  `json:"connection,omitempty"`
	Filter     string                  `json:"filter,omitempty"`
	Tag        string                  `json:"tag,omitempty"`
	Templates  []TemplateSummaryResult `json:"templates,omitempty"`
}

type TemplateInputResult struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

type TemplateActionSummaryResult struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	SQL         string `json:"sql,omitempty"`
}

type TemplateDescriptionResult struct {
	OK          bool                          `json:"ok"`
	Connection  string                        `json:"connection,omitempty"`
	Name        string                        `json:"name"`
	Scope       string                        `json:"scope"`
	Category    string                        `json:"category"`
	Command     string                        `json:"command"`
	Description string                        `json:"description,omitempty"`
	Transaction bool                          `json:"transaction"`
	Tags        []string                      `json:"tags,omitempty"`
	Inputs      []TemplateInputResult         `json:"inputs,omitempty"`
	Actions     []TemplateActionSummaryResult `json:"actions,omitempty"`
}

type TemplateInputValueResult struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Type      string `json:"type"`
	Defaulted bool   `json:"defaulted,omitempty"`
}

type TemplateRunResult struct {
	OK           bool                       `json:"ok"`
	Error        *ErrorResult               `json:"error,omitempty"`
	Connection   string                     `json:"connection,omitempty"`
	Command      string                     `json:"command,omitempty"`
	Template     string                     `json:"template,omitempty"`
	Layer        string                     `json:"layer,omitempty"`
	Category     string                     `json:"category,omitempty"`
	Source       string                     `json:"source,omitempty"`
	Preview      bool                       `json:"preview,omitempty"`
	DryRun       bool                       `json:"dry_run,omitempty"`
	Transaction  bool                       `json:"transaction,omitempty"`
	Committed    bool                       `json:"committed,omitempty"`
	RolledBack   bool                       `json:"rolled_back,omitempty"`
	Inputs       map[string]string          `json:"inputs,omitempty"`
	InputSummary []TemplateInputValueResult `json:"input_summary,omitempty"`
	Actions      []ActionResult             `json:"actions,omitempty"`
}

type TemplateValidationResult struct {
	OK         bool   `json:"ok"`
	Connection string `json:"connection,omitempty"`
	Name       string `json:"name"`
	Scope      string `json:"scope"`
	Category   string `json:"category"`
	Command    string `json:"command"`
	Valid      bool   `json:"valid"`
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

type UsersResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	User       string   `json:"user,omitempty"`
	Users      []string `json:"users,omitempty"`
}

type TablesResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	Database   string   `json:"database,omitempty"`
	Tables     []string `json:"tables,omitempty"`
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

type RowPreviewResult struct {
	OK         bool     `json:"ok"`
	Connection string   `json:"connection,omitempty"`
	Database   string   `json:"database,omitempty"`
	Table      string   `json:"table,omitempty"`
	Columns    []string `json:"columns,omitempty"`
	Rows       [][]any  `json:"rows,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

type CreateTableResult struct {
	OK          bool   `json:"ok"`
	Connection  string `json:"connection,omitempty"`
	Database    string `json:"database,omitempty"`
	Table       string `json:"table,omitempty"`
	CreateTable string `json:"create_table,omitempty"`
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
	OK               bool   `json:"ok"`
	Connection       string `json:"connection,omitempty"`
	Saved            bool   `json:"saved"`
	TestOK           *bool  `json:"test_ok,omitempty"`
	Warning          string `json:"warning,omitempty"`
	OverwriteCommand string `json:"overwrite_command,omitempty"`
	Path             string `json:"path,omitempty"`
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

type ContextResult struct {
	OK         bool   `json:"ok"`
	Connection string `json:"connection,omitempty"`
	Database   string `json:"database,omitempty"`
	Mode       string `json:"mode,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
}

type UseDatabaseResult struct {
	OK         bool   `json:"ok"`
	Connection string `json:"connection,omitempty"`
	Database   string `json:"database,omitempty"`
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
