package app

import (
	"fmt"
	"slices"
	"strings"

	"pkg.gostartkit.com/dbx/internal/driver"
)

const processInfoPreviewLimit = 48

var commonVariableNames = []string{
	"max_connections",
	"wait_timeout",
	"interactive_timeout",
	"sql_mode",
	"innodb_buffer_pool_size",
	"innodb_flush_log_at_trx_commit",
}

func sortedIndexes(indexes []driver.TableIndex) []driver.TableIndex {
	sorted := append([]driver.TableIndex(nil), indexes...)
	slices.SortFunc(sorted, func(a driver.TableIndex, b driver.TableIndex) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		case a.SeqInIndex < b.SeqInIndex:
			return -1
		case a.SeqInIndex > b.SeqInIndex:
			return 1
		case a.Column < b.Column:
			return -1
		case a.Column > b.Column:
			return 1
		case a.Type < b.Type:
			return -1
		case a.Type > b.Type:
			return 1
		default:
			return 0
		}
	})
	return sorted
}

func sortedProcesses(processes []driver.Process) []driver.Process {
	sorted := append([]driver.Process(nil), processes...)
	slices.SortFunc(sorted, func(a driver.Process, b driver.Process) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		case a.User < b.User:
			return -1
		case a.User > b.User:
			return 1
		case a.Host < b.Host:
			return -1
		case a.Host > b.Host:
			return 1
		default:
			return 0
		}
	})
	return sorted
}

func sortedVariables(variables []driver.SystemVariable) []driver.SystemVariable {
	sorted := append([]driver.SystemVariable(nil), variables...)
	slices.SortFunc(sorted, func(a driver.SystemVariable, b driver.SystemVariable) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		case a.Value < b.Value:
			return -1
		case a.Value > b.Value:
			return 1
		default:
			return 0
		}
	})
	return sorted
}

func sortedTableStatuses(statuses []driver.TableStatus) []driver.TableStatus {
	sorted := append([]driver.TableStatus(nil), statuses...)
	slices.SortFunc(sorted, func(a driver.TableStatus, b driver.TableStatus) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		default:
			return 0
		}
	})
	return sorted
}

func formatIndexLine(index driver.TableIndex) string {
	return fmt.Sprintf("%-16s %-8s %s", index.Name, emptyValue(index.Type, "<unknown>"), index.Column)
}

func formatProcessLine(process driver.Process) string {
	base := fmt.Sprintf(
		"%-4d %-16s %-24s %-8s %4s",
		process.ID,
		emptyValue(process.User, "<unknown>"),
		emptyValue(process.Host, "<unknown>"),
		emptyValue(process.Command, "<unknown>"),
		fmt.Sprintf("%ds", process.TimeSeconds),
	)
	info := truncateDisplayText(strings.TrimSpace(process.Info), processInfoPreviewLimit)
	if info == "" {
		return strings.TrimRight(base, " ")
	}
	return base + " " + info
}

func formatVariableLine(variable driver.SystemVariable) string {
	return fmt.Sprintf("%-24s %s", variable.Name, variable.Value)
}

func formatTableStatusSummary(status driver.TableStatus) string {
	return fmt.Sprintf(
		"%-16s %-7s rows=%-8d data=%-6s index=%s",
		status.Name,
		emptyValue(status.Engine, "<none>"),
		status.Rows,
		formatByteSize(status.DataLength),
		formatByteSize(status.IndexLength),
	)
}

func formatTableStatusDetail(status driver.TableStatus) []string {
	return []string{
		"Name: " + status.Name,
		"Engine: " + emptyValue(status.Engine, "<none>"),
		fmt.Sprintf("Rows: %d", status.Rows),
		"Data Size: " + formatByteSize(status.DataLength),
		"Index Size: " + formatByteSize(status.IndexLength),
		"Collation: " + emptyValue(status.Collation, "<none>"),
	}
}

func truncateDisplayText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func variableScopeLabel(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	return " matching " + pattern
}

func tableStatusScopeLabel(table string) string {
	table = strings.TrimSpace(table)
	if table == "" {
		return ""
	}
	return " for " + table
}

func formatByteSize(size int64) string {
	if size <= 0 {
		return "0B"
	}
	type unit struct {
		name  string
		value int64
	}
	units := []unit{
		{name: "GB", value: 1024 * 1024 * 1024},
		{name: "MB", value: 1024 * 1024},
		{name: "KB", value: 1024},
	}
	for _, unit := range units {
		if size >= unit.value {
			return fmt.Sprintf("%d%s", size/unit.value, unit.name)
		}
	}
	return fmt.Sprintf("%dB", size)
}
