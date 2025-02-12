package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// GetRow returns a single row as a map where keys are column names and values are the actual values.
// offset specifies which row to start from.
func (db *DB) GetRow(table *TableInfo, cursor cursorData, offset int) (map[string]interface{}, error) {
	// pagination query is basically the condtion for the WHERE statement
	paginationQuery, args, err := generateQueryForPagination(db.dbType, table.PrimaryKeys, cursor.cursors)
	if err != nil {
		return nil, fmt.Errorf("could not generate query for the cursor: %w", err)
	}
	paginationQuery = fmt.Sprintf("%s LIMIT %d", paginationQuery, 1)

	args = append(args, offset)

	// Build the query based on database type
	var query string
	switch db.dbType {
	case DatabaseDriverMysql:
		query = fmt.Sprintf("SELECT * FROM %s %s OFFSET ?", table.TableName, paginationQuery)
	case DatabaseDriverPostgres:
		query = fmt.Sprintf("SELECT * FROM %s.%s %s OFFSET $%d", db.currentSchema, table.TableName, paginationQuery, len(args))
	default:
		return nil, fmt.Errorf("unrecognized database driver: %s", db.dbType)
	}

	// Query the row
	rows, err := db.sqlDB.Queryx(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Check if we have a row
	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	// Get the row as a map
	result := make(map[string]interface{})
	if err := rows.MapScan(result); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	// Convert []byte values to string for MySQL
	if db.dbType == DatabaseDriverMysql {
		for k, v := range result {
			if b, ok := v.([]byte); ok {
				result[k] = string(b)
			}
		}
	}

	// Convert all map keys to lowercase
	lowercaseResult := make(map[string]interface{}, len(result))
	for k, v := range result {
		lowercaseResult[strings.ToLower(k)] = v
	}

	return lowercaseResult, nil

}
