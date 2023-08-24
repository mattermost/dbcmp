package store

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/mattermost/dbcmp/internal/testlib"
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
	t           *testing.T
	dbInstances map[string]*DB
}

func newTestHelper(t *testing.T) *testHelper {
	helper := &testHelper{
		t:           t,
		dbInstances: map[string]*DB{},
	}

	helper.initializeInstances()

	return helper
}

func (h *testHelper) initializeInstances() {
	// mysql
	db, err := NewDB(mysqlTestDSN)
	require.NoError(h.t, err)

	h.dbInstances["mysql"] = db

	// postgres
	db2, err := NewDB(pgsqlTestDSN)
	require.NoError(h.t, err)

	h.dbInstances["postgres"] = db2

	assets := testlib.Assets()
	for name, instance := range h.dbInstances {
		b, err := assets.ReadFile(filepath.Join("sql", name, "init.sql"))
		require.NoError(h.t, err)
		_, err = instance.sqlDB.Query(string(b))
		require.NoError(h.t, err)
	}
}

// SeedTableData randomly generates table data
func (h *testHelper) SeedTableData(entryCount int) *testHelper {
	for i := 0; i < entryCount; i++ {
		wc := rand.Intn(50)

		s1 := testStruct1{
			Id:          newId(),
			CreateAt:    gofakeit.Int64(),
			Name:        gofakeit.Name(),
			Description: gofakeit.Sentence(wc),
		}

		s2 := testStruct2{
			Id:        newId(),
			AnotherId: newId(),
			IsActive:  gofakeit.Bool(),
			Props:     gofakeit.Map(),
		}

		for name, instance := range h.dbInstances {
			query := `INSERT INTO Table1
			(Id, CreateAt, Name, Description)
			VALUES
			(:Id, :CreateAt, :Name, :Description)
			`
			_, err := instance.sqlDB.NamedExec(query, s1)
			require.NoError(h.t, err, "could not insert s1 on %q", name)

			query = `INSERT INTO Table2
			(Id, AnotherId, IsActive, Props)
			VALUES
			(:Id, :AnotherId, :IsActive, :Props)
			`
			_, err = instance.sqlDB.NamedExec(query, s2)
			require.NoError(h.t, err, "could not insert s2 on %q", name)
		}
	}

	return h
}

// TearDown closes all database connections and removes all tables from the databases
func (h *testHelper) Teardown() {
	assets := testlib.Assets()
	for name, instance := range h.dbInstances {
		b, err := assets.ReadFile(filepath.Join("sql", name, "drop.sql"))
		require.NoError(h.t, err)
		_, err = instance.sqlDB.Query(string(b))
		require.NoError(h.t, err)
	}

	for _, instance := range h.dbInstances {
		err := instance.Close()
		require.NoError(h.t, err)
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
