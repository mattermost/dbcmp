package store

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestNewDB(t *testing.T) {
	h := newTestHelper(t)
	defer h.Teardown()
}

func TestTableList(t *testing.T) {
	h := newTestHelper(t).SeedTableData()
	defer h.Teardown()
}
