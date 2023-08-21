package store

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompare(t *testing.T) {
	// compare empty databases
	mismatches, err := Compare(mysqlTestDSN, pgsqlTestDSN, CompareOptions{})
	require.NoError(t, err)
	require.Empty(t, mismatches)

	ec := rand.Intn(100)
	h := newTestHelper(t).SeedTableData(ec)
	defer h.Teardown()

	mismatches, err = Compare(mysqlTestDSN, pgsqlTestDSN, CompareOptions{
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Empty(t, mismatches)

	mismatches, err = Compare(pgsqlTestDSN, mysqlTestDSN, CompareOptions{
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Empty(t, mismatches)

	mysqldb, ok := h.dbInstances["mysql"]
	require.True(t, ok)

	// delete random entry
	_, err = mysqldb.sqlDB.Query("DELETE FROM Table1 LIMIT 1")
	require.NoError(t, err)

	mismatches, err = Compare(pgsqlTestDSN, mysqlTestDSN, CompareOptions{
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Len(t, mismatches, 1)
}
