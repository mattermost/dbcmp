package store

import (
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	mysqlTestDSN = "mysqluser:sspw@tcp(localhost:3316)/dbcmp_test?charset=utf8mb4,utf8"
	pgsqlTestDSN = "postgres://pguser:sspw@localhost:5442/mydb?sslmode=disable"
)

func TestNewDB(t *testing.T) {
	t.Run("Open MySQL database", func(t *testing.T) {
		db, err := NewDB(mysqlTestDSN)
		require.NoError(t, err)
		defer db.sqlDB.Close()
	})

	t.Run("Open PostgreSQL database", func(t *testing.T) {
		db, err := NewDB(pgsqlTestDSN)
		require.NoError(t, err)
		defer db.sqlDB.Close()
	})
}
