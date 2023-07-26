package store

import (
	"fmt"
	"strings"
)

// For each row we take the MD5 of each column. Use a space for NULL values.
// Concatenate those results, and MD5 this result.
// Split into 4 8-character hex strings.
// Convert into 32-bit integers and sum.

const MySQLChecksumTmpl = `select 
  sum(cast(conv(substring(hash, 1, 8), 16, 10) as unsigned)) as a, 
  sum(cast(conv(substring(hash, 9, 8), 16, 10) as unsigned)) as b, 
  sum(cast(conv(substring(hash, 17, 8), 16, 10) as unsigned)) as c, 
  sum(cast(conv(substring(hash, 25, 8), 16, 10) as unsigned)) as d
from (
  select md5(
    concat(
	  {{ .ColumnQuery}}
    )
  ) as "hash"
  from {{ .TableName }}
) as t;
`

// The ‘x’ prepended to the hash strings, which tells Postgres to interpret
// them as hex strings when casting to a number.
const PostgresChecksumTmpl = `select
  sum(('x' || substring(hash, 1, 8))::bit(32)::bigint) as a,
  sum(('x' || substring(hash, 9, 8))::bit(32)::bigint) as b,
  sum(('x' || substring(hash, 17, 8))::bit(32)::bigint) as c,
  sum(('x' || substring(hash, 25, 8))::bit(32)::bigint) as d
from (
  select md5 (
	{{ .ColumnQuery}}
  ) as "hash"
  from  {{.CurrentSchema}}.{{ .TableName }}
) as t;
`

// generateQueryForColumns creates the query for specific driver to calculate
// a md5 checksum of a table.
func generateQueryForColumns(driver string, columns []*ColumnInfo) string {
	c := make([]string, len(columns))
	switch driver {
	case DatabaseDriverMysql:
		for i := range columns {
			c[i] = fmt.Sprintf("coalesce(md5(%s), ' ')", columns[i].ColumnName)
		}
		return strings.Join(c, ",\n")
	case DatabaseDriverPostgres:
		for i := range columns {
			switch columns[i].DataType {
			case "boolean":
				c[i] = fmt.Sprintf("coalesce((md5((\"%s\"::int)::text)), ' ') ", columns[i].ColumnName)
			case "bytea":
				c[i] = fmt.Sprintf("coalesce((md5(\"%s\")), ' ') ", columns[i].ColumnName)
			default:
				c[i] = fmt.Sprintf("coalesce(md5(\"%s\"::text), ' ') ", columns[i].ColumnName)
			}

		}
		return strings.Join(c, "||\n")
	default:
		panic("unrecognized database driver")
	}
}
