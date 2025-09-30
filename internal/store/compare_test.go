package store

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

func TestCompare(t *testing.T) {
	t.Run("Compare empty database", func(t *testing.T) {
		mismatches, err := Compare(mysqlTestDSN, pgsqlTestDSN, CompareOptions{})
		require.NoError(t, err)
		require.Empty(t, mismatches)
	})

	t.Run("Compare empty database (legacy)", func(t *testing.T) {
		mismatches, err := Compare(mysqlLegacyTestDSN, pgsqlTestDSN, CompareOptions{})
		require.NoError(t, err)
		require.Empty(t, mismatches)
	})

	t.Run("Compare databases with same data", func(t *testing.T) {
		ec := rand.Intn(100) + 20 // we add 20 to ensure pagination gets triggered
		h := newTestHelper(t).SeedTableData(ec)
		defer h.Teardown()

		mismatches, err := Compare(mysqlTestDSN, pgsqlTestDSN, CompareOptions{
			PageSize: 20,
		})
		require.NoError(t, err)
		require.Empty(t, mismatches)
	})

	t.Run("Compare databases with same data (legacy)", func(t *testing.T) {
		ec := rand.Intn(100) + 20 // we add 20 to ensure pagination gets triggered
		h := newTestHelper(t).SeedTableData(ec)
		defer h.Teardown()

		mismatches, err := Compare(mysqlLegacyTestDSN, pgsqlTestDSN, CompareOptions{
			PageSize: 20,
		})
		require.NoError(t, err)
		require.Empty(t, mismatches)
	})

	t.Run("Compare databases with other way around", func(t *testing.T) {
		ec := rand.Intn(100) + 20 // we add 20 to ensure pagination gets triggered
		h := newTestHelper(t).SeedTableData(ec)
		defer h.Teardown()

		mismatches, err := Compare(pgsqlTestDSN, mysqlTestDSN, CompareOptions{
			PageSize: 20,
		})
		require.NoError(t, err)
		require.Empty(t, mismatches)
	})

	t.Run("Compare databases with higher page size", func(t *testing.T) {
		ec := rand.Intn(100)
		h := newTestHelper(t).SeedTableData(ec)
		defer h.Teardown()

		mismatches, err := Compare(pgsqlTestDSN, mysqlTestDSN, CompareOptions{
			PageSize: 1000,
			Verbose:  true,
		})
		require.NoError(t, err)
		require.Empty(t, mismatches)
	})

	t.Run("Compare databases when there is data change", func(t *testing.T) {
		ec := rand.Intn(100) + 20 // we add 20 to ensure pagination gets triggered
		h := newTestHelper(t).SeedTableData(ec)
		defer h.Teardown()

		mysqldb, ok := h.dbInstances["mysql"]
		require.True(t, ok)

		// delete random entry
		_, err := mysqldb.sqlDB.Query("DELETE FROM Table1 LIMIT 1")
		require.NoError(t, err)

		mismatches, err := Compare(pgsqlTestDSN, mysqlTestDSN, CompareOptions{
			PageSize: 20,
		})
		require.NoError(t, err)
		require.Len(t, mismatches, 1)

		// add random entry
		_, err = mysqldb.sqlDB.Query("INSERT INTO Table1 (Id, CreateAt, Name, Description) VALUES (?, ?, ?, ?)",
			strings.Repeat("0", 26), gofakeit.Int64(),
			gofakeit.Name(),
			gofakeit.Sentence(10),
		)
		require.NoError(t, err)

		mismatches, err = Compare(pgsqlTestDSN, mysqlTestDSN, CompareOptions{
			PageSize: 20,
			Verbose:  true,
		})
		require.NoError(t, err)
		require.Len(t, mismatches, 1)
	})

	t.Run("Assert exclude and include flags", func(t *testing.T) {
		ec := rand.Intn(100) + 20 // we add 20 to ensure pagination gets triggered
		h := newTestHelper(t).SeedTableData(ec)
		defer h.Teardown()

		// test with exclude patterns
		mismatches, err := Compare(mysqlTestDSN, pgsqlTestDSN, CompareOptions{
			ExcludePatterns: []string{"Table1"},
		})
		require.NoError(t, err)
		require.Empty(t, mismatches)

		// Table2 is the same
		mismatches, err = Compare(mysqlTestDSN, pgsqlTestDSN, CompareOptions{
			IncludePatterns: []string{"Table2"},
		})
		require.NoError(t, err)
		require.Empty(t, mismatches)

		mysqldb, ok := h.dbInstances["mysql"]
		require.True(t, ok)

		// delete random entry
		_, err = mysqldb.sqlDB.Query("DELETE FROM Table1 LIMIT 1")
		require.NoError(t, err)

		// test with include patterns
		mismatches, err = Compare(mysqlTestDSN, pgsqlTestDSN, CompareOptions{
			IncludePatterns: []string{"Table1"},
		})
		require.NoError(t, err)
		require.Len(t, mismatches, 1)

		// test with both include and exclude patterns
		_, err = Compare(mysqlTestDSN, pgsqlTestDSN, CompareOptions{
			IncludePatterns: []string{"Table1"},
			ExcludePatterns: []string{"Table2"},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "include and exclude flags cannot be used together")
	})
}
