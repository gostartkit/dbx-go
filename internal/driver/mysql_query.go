package driver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"

	"pkg.gostartkit.com/dbx/internal/util"
)

func ListDatabases(ctx context.Context, db *sql.DB) ([]string, error) {
	return QueryStrings(ctx, db, "SHOW DATABASES")
}

type SchemaColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Key      string `json:"key,omitempty"`
	Extra    string `json:"extra,omitempty"`
}

type RowSet struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

type TableStatus struct {
	Name        string `json:"name"`
	Engine      string `json:"engine"`
	Rows        int64  `json:"rows"`
	DataLength  int64  `json:"data_length"`
	IndexLength int64  `json:"index_length"`
	Collation   string `json:"collation,omitempty"`
}

func Ping(ctx context.Context, db *sql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return util.WrapLayer("mysql", "ping database", err)
	}
	return nil
}

func QueryStrings(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.WrapLayer("sql execution", "run query", err)
	}
	defer rows.Close()

	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, util.WrapLayer("sql execution", "scan query result", err)
		}
		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "read query rows", err)
	}

	return values, nil
}

func ListTables(ctx context.Context, db *sql.DB, database string) ([]string, error) {
	query := "SHOW TABLES FROM " + util.QuoteMySQLIdentifier(database)
	return QueryStrings(ctx, db, query)
}

func ShowColumns(ctx context.Context, db *sql.DB, database string, table string) ([]SchemaColumn, error) {
	rows, err := db.QueryContext(ctx, `
SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, EXTRA
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
ORDER BY ORDINAL_POSITION
`, database, table)
	if err != nil {
		return nil, util.WrapLayer("sql execution", "run query", err)
	}
	defer rows.Close()

	columns := make([]SchemaColumn, 0)
	for rows.Next() {
		var (
			name       string
			columnType string
			nullable   string
			key        sql.NullString
			extra      sql.NullString
		)
		if err := rows.Scan(&name, &columnType, &nullable, &key, &extra); err != nil {
			return nil, util.WrapLayer("sql execution", "scan query result", err)
		}
		columns = append(columns, SchemaColumn{
			Name:     name,
			Type:     columnType,
			Nullable: strings.EqualFold(nullable, "YES"),
			Key:      key.String,
			Extra:    extra.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, util.WrapLayer("sql execution", "read query rows", err)
	}
	if len(columns) == 0 {
		exists, err := tableExists(ctx, db, database, table)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, util.WrapLayer("validation", "show columns", fmt.Errorf("table not found: %s", table))
		}
	}
	return columns, nil
}

func PeekRows(ctx context.Context, db *sql.DB, database string, table string, limit int) (*RowSet, error) {
	return queryRows(ctx, db, database, table, limit, false)
}

func SampleRows(ctx context.Context, db *sql.DB, database string, table string, limit int) (*RowSet, error) {
	return queryRows(ctx, db, database, table, limit, true)
}

func ShowCreateTable(ctx context.Context, db *sql.DB, database string, table string) (string, error) {
	var ddl string
	err := withDatabaseConn(ctx, db, database, func(conn *sql.Conn) error {
		rows, err := conn.QueryContext(ctx, "SHOW CREATE TABLE "+util.QuoteMySQLIdentifier(table))
		if err != nil {
			return classifyTableOperationError("show create table", table, err)
		}
		defer rows.Close()

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return util.WrapLayer("sql execution", "read query rows", err)
			}
			return util.WrapLayer("validation", "show create table", fmt.Errorf("table not found: %s", table))
		}

		var tableName string
		if err := rows.Scan(&tableName, &ddl); err != nil {
			return util.WrapLayer("sql execution", "scan query result", err)
		}
		if err := rows.Err(); err != nil {
			return util.WrapLayer("sql execution", "read query rows", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return ddl, nil
}

func ShowTableStatus(ctx context.Context, db *sql.DB, database string, table string) ([]TableStatus, error) {
	statuses := make([]TableStatus, 0)
	err := withDatabaseConn(ctx, db, database, func(conn *sql.Conn) error {
		query := "SHOW TABLE STATUS"
		if strings.TrimSpace(table) != "" {
			query += " LIKE '" + util.EscapeMySQLString(table) + "'"
		}

		rows, err := conn.QueryContext(ctx, query)
		if err != nil {
			return util.WrapLayer("sql execution", "run query", err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				name          string
				engine        sql.NullString
				version       sql.NullInt64
				rowFormat     sql.NullString
				rowsValue     sql.NullInt64
				avgRowLength  sql.NullInt64
				dataLength    sql.NullInt64
				maxDataLength sql.NullInt64
				indexLength   sql.NullInt64
				dataFree      sql.NullInt64
				autoIncrement sql.NullInt64
				createTime    sql.NullTime
				updateTime    sql.NullTime
				checkTime     sql.NullTime
				collation     sql.NullString
				checksum      sql.NullInt64
				createOptions sql.NullString
				comment       sql.NullString
			)
			if err := rows.Scan(
				&name,
				&engine,
				&version,
				&rowFormat,
				&rowsValue,
				&avgRowLength,
				&dataLength,
				&maxDataLength,
				&indexLength,
				&dataFree,
				&autoIncrement,
				&createTime,
				&updateTime,
				&checkTime,
				&collation,
				&checksum,
				&createOptions,
				&comment,
			); err != nil {
				return util.WrapLayer("sql execution", "scan query result", err)
			}
			statuses = append(statuses, TableStatus{
				Name:        name,
				Engine:      engine.String,
				Rows:        rowsValue.Int64,
				DataLength:  dataLength.Int64,
				IndexLength: indexLength.Int64,
				Collation:   collation.String,
			})
		}
		if err := rows.Err(); err != nil {
			return util.WrapLayer("sql execution", "read query rows", err)
		}
		if strings.TrimSpace(table) != "" && len(statuses) == 0 {
			return util.WrapLayer("validation", "show table status", fmt.Errorf("table not found: %s", table))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return statuses, nil
}

func ExecStatement(ctx context.Context, db *sql.DB, statement string) error {
	if _, err := db.ExecContext(ctx, statement); err != nil {
		return util.WrapLayer("sql execution", "execute statement", err)
	}
	return nil
}

func queryRows(ctx context.Context, db *sql.DB, database string, table string, limit int, random bool) (*RowSet, error) {
	result := &RowSet{Columns: []string{}, Rows: [][]any{}}
	err := withDatabaseConn(ctx, db, database, func(conn *sql.Conn) error {
		query := "SELECT * FROM " + util.QuoteMySQLIdentifier(table)
		if random {
			query += " ORDER BY RAND()"
		}
		query += " LIMIT ?"

		rows, err := conn.QueryContext(ctx, query, limit)
		if err != nil {
			return classifyTableOperationError(operationNameForRows(random), table, err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return util.WrapLayer("sql execution", "read query columns", err)
		}
		result.Columns = append(result.Columns, columns...)

		for rows.Next() {
			values := make([]any, len(columns))
			dest := make([]any, len(columns))
			for i := range values {
				dest[i] = &values[i]
			}
			if err := rows.Scan(dest...); err != nil {
				return util.WrapLayer("sql execution", "scan query result", err)
			}
			row := make([]any, 0, len(values))
			for _, value := range values {
				row = append(row, normalizeRowValue(value))
			}
			result.Rows = append(result.Rows, row)
		}
		if err := rows.Err(); err != nil {
			return util.WrapLayer("sql execution", "read query rows", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func tableExists(ctx context.Context, db *sql.DB, database string, table string) (bool, error) {
	row := db.QueryRowContext(ctx, `
SELECT 1
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
LIMIT 1
`, database, table)
	var marker int
	if err := row.Scan(&marker); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, util.WrapLayer("sql execution", "run query", err)
	}
	return true, nil
}

func normalizeRowValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format("2006-01-02 15:04:05")
	default:
		return typed
	}
}

func operationNameForRows(random bool) string {
	if random {
		return "sample rows"
	}
	return "peek rows"
}

func withDatabaseConn(ctx context.Context, db *sql.DB, database string, fn func(conn *sql.Conn) error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return util.WrapLayer("mysql", "open database connection", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "USE "+util.QuoteMySQLIdentifier(database)); err != nil {
		return util.WrapLayer("mysql", "select database", err)
	}
	return fn(conn)
}

func execWithDatabase(ctx context.Context, db *sql.DB, database string, statement string, operation string, table string) error {
	return withDatabaseConn(ctx, db, database, func(conn *sql.Conn) error {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return classifyTableOperationError(operation, table, err)
		}
		return nil
	})
}

func classifyTableOperationError(operation string, table string, err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
		return util.WrapLayer("validation", operation, fmt.Errorf("table not found: %s", table))
	}
	return util.WrapLayer("sql execution", "run query", err)
}
