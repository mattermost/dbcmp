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
	TableName   string
	PrimaryKeys []string
	Columns     []*ColumnInfo
}

// ColumnInfo is the column info
type ColumnInfo struct {
	ColumnName string `db:"column_name"`
	DataType   string `db:"data_type"`
}

type cursorData struct {
	cursors []any
	limit   int
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
		pks, err := db.primaryKeys(s)
		if err != nil {
			return nil, fmt.Errorf("could not determine primary keys: %w", err)
		}
		elementMap[strings.ToLower(s)] = &TableInfo{
			TableName:   s,
			Columns:     columns,
			PrimaryKeys: pks,
		}
	}

	return elementMap, nil
}

func (db *DB) dataTypes(table string) ([]*ColumnInfo, error) {
	sqt := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	if db.dbType == DatabaseDriverPostgres {
		sqt = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	}

	// with mysql-8, column_name is capitalized and it complains when
	// querying like this. This works for both.
	sqb := sqt.Select("column_name as column_name, data_type as data_type").
		From("information_schema.columns").
		Where(sq.And{sq.Eq{"table_name": table}})

	query, args, err := sqb.ToSql()
	if err != nil {
		return nil, err
	}

	if db.dbType == DatabaseDriverMysql {
		query += " AND table_schema = Database()"
	} else if db.dbType == DatabaseDriverPostgres {
		query += " AND table_schema = CURRENT_SCHEMA()"
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

func (db *DB) checksum(table *TableInfo, cursor cursorData) (string, cursorData, error) {
	q := struct {
		TableName     string
		ColumnQuery   string
		CurrentSchema string
		CursorQuery   string
	}{
		TableName:   table.TableName,
		ColumnQuery: generateQueryForColumns(db.dbType, table.Columns),
	}

	t := template.New("query")
	var tmpl string

	// we need the decide on which template to use for the checksum query
	switch db.dbType {
	case DatabaseDriverMysql:
		tmpl = MySQLChecksumTmpl
	case DatabaseDriverPostgres:
		tmpl = PostgresChecksumTmpl
		// in addition to the template, postgres reuqies the selected schema id
		// to access objects, it's generally public but it's not guaranteed
		var schemaName sql.NullString
		err := db.sqlDB.Get(&schemaName, "SELECT current_schema()")
		if err != nil {
			return "", cursorData{}, fmt.Errorf("could not get current schema: %w", err)
		} else if schemaName.String == "" {
			q.CurrentSchema = "public"
		} else {
			q.CurrentSchema = schemaName.String
		}
	default:
		return "", cursorData{}, fmt.Errorf("unrecognized database driver: %s", db.dbType)
	}

	t, err := t.Parse(tmpl)
	if err != nil {
		return "", cursorData{}, fmt.Errorf("could not parse template: %w", err)
	}

	// pagination query is basically the condtion for the WHERE statement for the
	// checksum query.
	paginationQuery, args, err := generateQueryForPagination(db.dbType, table.PrimaryKeys, cursor.cursors)
	if err != nil {
		return "", cursorData{}, fmt.Errorf("could not generate query for the cursor: %w", err)
	}
	paginationQuery = fmt.Sprintf("%s LIMIT %d", paginationQuery, cursor.limit)
	q.CursorQuery = paginationQuery

	out := bytes.NewBufferString("")
	err = t.Execute(out, q)
	if err != nil {
		return "", cursorData{}, fmt.Errorf("could not execute template: %w", err)
	}

	var ints = struct {
		A uint `db:"a"`
		B uint `db:"b"`
		C uint `db:"c"`
		D uint `db:"d"`
	}{}
	err = db.sqlDB.Get(&ints, out.String(), args...)
	if err != nil {
		return "", cursorData{}, fmt.Errorf("could not select checksum: %w", err)
	}
	// we convert the result to a string, it's easier to compare
	result := fmt.Sprintf("%d%d%d%d", ints.A, ints.B, ints.C, ints.D)

	// the remaining part is about determining the next cursors.
	// since we don't actually read any rows from the tables for sum operation,
	// we simply need to pick required rows from the query executed above.
	var c sq.And
	for i := range cursor.cursors {
		c = append(c, sq.Gt{table.PrimaryKeys[i]: cursor.cursors[i]})
	}

	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	tableName := table.TableName
	if db.dbType == DatabaseDriverPostgres {
		builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
		tableName = strings.Join([]string{q.CurrentSchema, tableName}, ".")
	}

	cursorQueryBuilder := builder.
		Select(table.PrimaryKeys...).
		From(tableName).
		Where(c).
		OrderBy(strings.Join(table.PrimaryKeys, ",") + " ASC").
		Limit(uint64(cursor.limit))

	// if we don't have anything for a where condtion, we avoid adding it.
	if cursor.cursors == nil {
		cursorQueryBuilder = builder.
			Select(table.PrimaryKeys...).
			From(tableName).
			OrderBy(strings.Join(table.PrimaryKeys, ",") + " ASC").
			Limit(uint64(cursor.limit))
	}

	// to determine whether we should continue iterating the cursor,
	// we get the count of remained rows here
	cursorCountQueryBuilder := builder.
		Select("COUNT(*)").
		FromSelect(cursorQueryBuilder, "q2")

	cursorCountQuery, cursorCountArgs, err := cursorCountQueryBuilder.ToSql()
	if err != nil {
		return "", cursorData{}, fmt.Errorf("could not build cursor query: %w", err)
	}

	var count int
	err = db.sqlDB.Get(&count, cursorCountQuery, cursorCountArgs...)
	if err != nil {
		return "", cursorData{}, fmt.Errorf("could not select cursors: %w", err)
	}

	// we break here because that means the items received from the previous call
	// was the last remaining rows from the cursor.
	// in short, it means that we are already at the end of the cursor
	if count < cursor.limit {
		return result, cursorData{}, nil
	}

	cursors := []any{}
	// We can't scan a single row into a slice, it needs to be a struct.
	// In our case we have no idea what the struct would look like as
	// there can be many primary keys. As a workaround we select individual
	// primary key and append it to the cursors slice
	for _, pk := range table.PrimaryKeys {
		lastCursorQueryBuilder := builder.
			Select(pk).
			FromSelect(cursorQueryBuilder, "q1").
			OrderBy(strings.Join(table.PrimaryKeys, ",") + " DESC").
			Limit(1)

		lastCursorQuery, lastCursorArgs, err := lastCursorQueryBuilder.ToSql()
		if err != nil {
			return "", cursorData{}, fmt.Errorf("could not build cursor query: %w", err)
		}

		var c any
		err = db.sqlDB.Get(&c, lastCursorQuery, lastCursorArgs...)
		if err != nil {
			return "", cursorData{}, fmt.Errorf("could not select cursors: %w", err)
		}

		cursors = append(cursors, c)
	}

	return result, cursorData{
		cursors: cursors,
		limit:   count,
	}, nil
}

// pirmaryKeys returns the primary keys of a table. Essentially we want to
// do a ORDER BY PRIMARY KEY operation. Apparently it's not that simple in the
// sql world.
func (db *DB) primaryKeys(tableName string) ([]string, error) {
	pks := []string{}
	switch db.dbType {
	case DatabaseDriverMysql:
		query := `SELECT 
			COLUMN_NAME
		FROM 
			INFORMATION_SCHEMA.COLUMNS
		WHERE 
			TABLE_SCHEMA = DATABASE()
		AND 
			TABLE_NAME = ?
		AND
			COLUMN_KEY = 'PRI'`

		err := db.sqlDB.Select(&pks, query, tableName)
		if err != nil {
			return nil, err
		}

	case DatabaseDriverPostgres:
		var currentSchema string
		var schemaName sql.NullString
		err := db.sqlDB.Get(&schemaName, "SELECT current_schema()")
		if err != nil {
			return nil, fmt.Errorf("could not get current schema: %w", err)
		} else if schemaName.String == "" {
			currentSchema = "public"
		} else {
			currentSchema = schemaName.String
		}
		// interestingly postgres append schema name into the attname
		// hence we prefix the currentSchema name to the table name
		query := `SELECT
			pg_attribute.attname
		FROM
			pg_index, pg_class, pg_attribute, pg_namespace
		WHERE
			pg_class.oid = $1 ::regclass
		AND
			indrelid = pg_class.oid
		AND
			pg_class.relnamespace = pg_namespace.oid
		AND
			pg_attribute.attrelid = pg_class.oid
		AND
			pg_attribute.attnum = any(pg_index.indkey)
		AND
			indisprimary`
		err = db.sqlDB.Select(&pks, query, strings.Join([]string{currentSchema, tableName}, "."))
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("could not list primary keys: unknown database driver")
	}

	return pks, nil
}
