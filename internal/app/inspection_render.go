package app

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"pkg.gostartkit.com/dbx/internal/driver"
)

const rowPreviewCellLimit = 80

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

func formatSchemaColumnLine(column driver.SchemaColumn) string {
	line := fmt.Sprintf(
		"%-12s %-18s %-4s %-4s %s",
		column.Name,
		column.Type,
		boolToNullable(column.Nullable),
		emptyValue(column.Key, ""),
		emptyValue(column.Extra, ""),
	)
	return strings.TrimRight(line, " ")
}

func formatRowPreview(columns []string, rows [][]any) []string {
	if len(columns) == 0 {
		return nil
	}

	widths := make([]int, len(columns))
	for i, column := range columns {
		widths[i] = len(column)
	}

	displayRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		displayRow := make([]string, 0, len(columns))
		for i := range columns {
			value := "NULL"
			if i < len(row) {
				value = formatRowCell(row[i])
			}
			displayRow = append(displayRow, value)
			if len(value) > widths[i] {
				widths[i] = len(value)
			}
		}
		displayRows = append(displayRows, displayRow)
	}

	lines := make([]string, 0, len(displayRows)+1)
	lines = append(lines, joinRowCells(columns, widths))
	for _, row := range displayRows {
		lines = append(lines, joinRowCells(row, widths))
	}
	return lines
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

func boolToNullable(nullable bool) string {
	if nullable {
		return "YES"
	}
	return "NO"
}

func formatRowCell(value any) string {
	switch typed := value.(type) {
	case nil:
		return "NULL"
	case string:
		return truncateDisplayText(typed, rowPreviewCellLimit)
	case []byte:
		return truncateDisplayText(string(typed), rowPreviewCellLimit)
	case time.Time:
		return typed.Format("2006-01-02 15:04:05")
	case fmt.Stringer:
		return truncateDisplayText(typed.String(), rowPreviewCellLimit)
	case int:
		return strconv.Itoa(typed)
	case int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprint(typed)
	default:
		return truncateDisplayText(fmt.Sprint(typed), rowPreviewCellLimit)
	}
}

func joinRowCells(values []string, widths []int) string {
	parts := make([]string, 0, len(values))
	for i, value := range values {
		parts = append(parts, fmt.Sprintf("%-*s", widths[i], value))
	}
	return strings.TrimRight(strings.Join(parts, "  "), " ")
}
