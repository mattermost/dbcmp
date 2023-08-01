package store

import (
	"math/rand"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestTableList(t *testing.T) {
	ec := rand.Intn(100)
	h := newTestHelper(t).SeedTableData(ec)
	defer h.Teardown()

	h.RunForAllDrivers(t, func(t *testing.T, db *DB) {
		tables, err := db.TableList()
		require.NoError(t, err)
		require.Len(t, tables, 2)
	})
}

func TestTableCount(t *testing.T) {
	ec := rand.Intn(100)
	h := newTestHelper(t).SeedTableData(ec)
	defer h.Teardown()

	h.RunForAllDrivers(t, func(t *testing.T, db *DB) {
		tables, err := db.TableList()
		require.NoError(t, err)

		c, err := db.count(tables["table1"])
		require.NoError(t, err)
		require.Equal(t, ec, c)
	})
}

func TestCheksum(t *testing.T) {
	ec := rand.Intn(100)
	h := newTestHelper(t).SeedTableData(ec)
	defer h.Teardown()

	h.RunForAllDrivers(t, func(t *testing.T, db *DB) {
		tables, err := db.TableList()
		require.NoError(t, err)

		sum, err := db.checksum(tables["table1"])
		require.NoError(t, err)
		require.Len(t, sum, 4)
	})
}
