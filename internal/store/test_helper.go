package store

import (
	"path/filepath"
	"testing"

	"github.com/isacikgoz/dbcmp/internal/testlib"
	"github.com/stretchr/testify/require"
)

const (
	mysqlTestDSN = "mysqluser:sspw@tcp(localhost:3316)/dbcmp_test?charset=utf8mb4,utf8"
	pgsqlTestDSN = "postgres://pguser:sspw@localhost:5442/dbcmp_test?sslmode=disable"
)

// testHelper is a helper struct for testing morph engine.
// It contains all the necessary information to run tests for all drivers.
// It also provides helper functions to create dummy migrations.
type testHelper struct {
	dbInstances map[string]*DB
}

func newTestHelper(t *testing.T) *testHelper {
	helper := &testHelper{
		dbInstances: map[string]*DB{},
	}

	helper.initializeInstances(t)

	return helper
}

func (h *testHelper) initializeInstances(t *testing.T) {
	// mysql
	db, err := NewDB(mysqlTestDSN)
	require.NoError(t, err)

	h.dbInstances["mysql"] = db

	// postgres
	db2, err := NewDB(pgsqlTestDSN)
	require.NoError(t, err)

	h.dbInstances["postgres"] = db2
}

// TearDown closes all database connections and removes all tables from the databases
func (h *testHelper) Teardown(t *testing.T) {
	assets := testlib.Assets()
	for name, instance := range h.dbInstances {
		b, err := assets.ReadFile(filepath.Join("sql", name, "drop.sql"))
		require.NoError(t, err)
		_, err = instance.sqlDB.Query(string(b))
		require.NoError(t, err)
	}

	for _, instance := range h.dbInstances {
		err := instance.Close()
		require.NoError(t, err)
	}
}

// RunForAllDrivers runs the given test function for all drivers of the test helper
func (h *testHelper) RunForAllDrivers(t *testing.T, f func(t *testing.T, db *DB), name ...string) {
	var testName string
	if len(name) > 0 {
		testName = name[0] + "/"
	}

	for name, instance := range h.dbInstances {
		t.Run(testName+name, func(t *testing.T) {
			f(t, instance)
		})
	}
}
