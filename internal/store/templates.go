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
	// ideally we sshould be able to define casting rules here
	// or skip some of the columns entirely from calculating the md5
	c := make([]string, len(columns))
	switch driver {
	case DatabaseDriverMysql:
		for i := range columns {
			name := columns[i].ColumnName
			// reserved words require quotes in mysql
			if name == "Desc" || name == "Trigger" {
				name = fmt.Sprintf("`%s`", name)
			}
			c[i] = fmt.Sprintf("coalesce(md5(%s), ' ')", name)
		}
		return strings.Join(c, ",\n")
	case DatabaseDriverPostgres:
		for i := range columns {
			name := columns[i].ColumnName
			switch columns[i].DataType {
			case "boolean":
				c[i] = fmt.Sprintf("coalesce((md5((\"%s\"::int)::text)), ' ') ", name)
			case "bytea":
				c[i] = fmt.Sprintf("coalesce((md5(\"%s\")), ' ') ", name)
			default:
				c[i] = fmt.Sprintf("coalesce(md5(\"%s\"::text), ' ') ", name)
			}

		}
		return strings.Join(c, "||\n")
	default:
		panic("unrecognized database driver")
	}
}
