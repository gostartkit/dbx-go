package app

import (
	"context"
	"time"

	"pkg.gostartkit.com/dbx/internal/config"
)

const auditLogLimit = 50

type auditMetadata struct {
	Command    string
	Connection string
	Mode       string
	DryRun     bool
	Success    *bool
}

func (a *Application) auditCommand(ctx context.Context, meta auditMetadata, fn func(meta *auditMetadata) error) error {
	_ = ctx
	startedAt := time.Now()
	err := fn(&meta)

	record := &config.AuditRecord{
		Timestamp:  time.Now().UTC(),
		Command:    meta.Command,
		Connection: meta.Connection,
		Mode:       meta.Mode,
		DryRun:     meta.DryRun,
		Success:    err == nil,
		DurationMS: time.Since(startedAt).Milliseconds(),
	}
	if meta.Success != nil {
		record.Success = *meta.Success
	}
	_ = a.store.AppendAudit(record)
	return err
}

func (a *Application) loadAuditLog() (*AuditLogResult, error) {
	entries, err := a.store.LoadAudit(auditLogLimit)
	if err != nil {
		return nil, err
	}
	return &AuditLogResult{
		OK:      true,
		Entries: entries,
	}, nil
}

func (a *Application) printAuditLog(result *AuditLogResult) {
	if result == nil {
		return
	}
	if len(result.Entries) == 0 {
		a.prompt.Println("No audit entries found.")
		return
	}

	a.prompt.Println("Recent audit entries:")
	for _, entry := range result.Entries {
		status := "ok"
		if !entry.Success {
			status = "fail"
		}
		a.prompt.Printf("  - %s %s", entry.Timestamp.Format(time.RFC3339), entry.Command)
		if entry.Connection != "" {
			a.prompt.Printf(" connection=%s", entry.Connection)
		}
		if entry.Mode != "" {
			a.prompt.Printf(" mode=%s", entry.Mode)
		}
		if entry.DryRun {
			a.prompt.Printf(" dry-run=true")
		}
		a.prompt.Printf(" status=%s duration=%dms\n", status, entry.DurationMS)
	}
}

func (a *Application) handleAuditLog(ctx context.Context) error {
	return a.auditCommand(ctx, auditMetadata{Command: "audit log"}, func(meta *auditMetadata) error {
		result, err := a.loadAuditLog()
		if err != nil {
			return err
		}
		a.printAuditLog(result)
		return nil
	})
}
