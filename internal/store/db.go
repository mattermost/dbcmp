package store

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

const (
	DatabaseDriverPostgres = "postgres"
	DatabaseDriverMysql    = "mysql"
)

var (
	ErrEmptyTable = errors.New("the table has no rows to calculate md5")
)

// DB is the sql.DB wrapper with some utilities
type DB struct {
	sqlDB  *sqlx.DB
	dbType string
}

type TableInfo struct {
	TableName string
	Columns   []*ColumnInfo
}

// ColumnInfo is the column info
type ColumnInfo struct {
	ColumnName string `db:"column_name"`
	DataType   string `db:"data_type"`
}

// NewDB creates a DB instance from the given data source
func NewDB(dsn string) (*DB, error) {
	dbType := DatabaseDriverMysql
	if strings.HasPrefix(dsn, "postgres") {
		dbType = DatabaseDriverPostgres

	}

	newDsn, err := normalizeDSN(dsn)
	if err != nil {
		return nil, err
	}

	db, err := sqlx.Open(dbType, newDsn)
	if err != nil {
		return nil, err
	}

	return &DB{
		sqlDB:  db,
		dbType: dbType,
	}, nil
}

func (db *DB) Close() error {
	if db.sqlDB != nil {
		return nil
	}

	return db.sqlDB.Close()
}

func (db *DB) TableList() (map[string]*TableInfo, error) {
	tables := []string{}
	switch db.dbType {
	case DatabaseDriverMysql:
		err := db.sqlDB.Select(&tables, `show tables`)
		if err != nil {
			return nil, err
		}
	case DatabaseDriverPostgres:
		err := db.sqlDB.Select(&tables, `SELECT tablename
		FROM pg_catalog.pg_tables
		WHERE schemaname != 'pg_catalog' AND 
			schemaname != 'information_schema';`)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("could not list tables: unknown database driver")
	}

	elementMap := make(map[string]*TableInfo)
	for _, s := range tables {
		columns, err := db.dataTypes(s)
		if err != nil {
			return nil, fmt.Errorf("could not populate columns: %w", err)
		}
		elementMap[strings.ToLower(s)] = &TableInfo{
			TableName: s,
			Columns:   columns,
		}
	}

	return elementMap, nil
}

func (db *DB) dataTypes(table string) ([]*ColumnInfo, error) {
	sqt := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	if db.dbType == DatabaseDriverPostgres {
		sqt = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	}

	sqb := sqt.Select("column_name, data_type").
		From("information_schema.columns").
		Where(sq.And{sq.Eq{"table_name": table}})

	query, args, err := sqb.ToSql()
	if err != nil {
		return nil, err
	}

	if db.dbType == DatabaseDriverMysql {
		query += " AND table_schema = Database()"
	}

	var v []*ColumnInfo
	err = db.sqlDB.Select(&v, query, args...)
	if err != nil {
		return nil, err
	}

	sort.Slice(v, func(i, j int) bool {
		return strings.ToLower(v[i].ColumnName) < strings.ToLower(v[j].ColumnName)
	})

	return v, nil
}

func (db *DB) count(table *TableInfo) (int, error) {
	var query string
	switch db.dbType {
	case DatabaseDriverMysql:
		query = "SELECT count(*) FROM " + table.TableName
	case DatabaseDriverPostgres:
		var currentSchema string
		var schemaName sql.NullString
		err := db.sqlDB.Get(&schemaName, "SELECT current_schema()")
		if err != nil {
			return 0, fmt.Errorf("could not get current schema: %w", err)
		} else if schemaName.String == "" {
			currentSchema = "public"
		} else {
			currentSchema = schemaName.String
		}
		query = "SELECT count(*) FROM " + strings.Join([]string{currentSchema, table.TableName}, ".")
	default:
		return 0, fmt.Errorf("unrecognized database driver: %s", db.dbType)
	}

	var count int
	err := db.sqlDB.Get(&count, query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (db *DB) checksum(table *TableInfo) (string, error) {
	q := struct {
		TableName     string
		ColumnQuery   string
		CurrentSchema string
	}{
		TableName:   table.TableName,
		ColumnQuery: generateQueryForColumns(db.dbType, table.Columns),
	}

	t := template.New("query")
	var tmpl string

	switch db.dbType {
	case DatabaseDriverMysql:
		tmpl = MySQLChecksumTmpl
	case DatabaseDriverPostgres:
		tmpl = PostgresChecksumTmpl
		var schemaName sql.NullString
		err := db.sqlDB.Get(&schemaName, "SELECT current_schema()")
		if err != nil {
			return "", fmt.Errorf("could not get current schema: %w", err)
		} else if schemaName.String == "" {
			q.CurrentSchema = "public"
		} else {
			q.CurrentSchema = schemaName.String
		}
	default:
		return "", fmt.Errorf("unrecognized database driver: %s", db.dbType)
	}

	t, err := t.Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("could not parse template: %w", err)
	}

	out := bytes.NewBufferString("")
	err = t.Execute(out, q)
	if err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}

	var v = []struct {
		A uint `db:"a"`
		B uint `db:"b"`
		C uint `db:"c"`
		D uint `db:"d"`
	}{}
	err = db.sqlDB.Select(&v, out.String())
	if err != nil {
		return "", err
	}
	ints := v[0]

	return fmt.Sprintf("%d%d%d%d", ints.A, ints.B, ints.C, ints.D), nil
}
