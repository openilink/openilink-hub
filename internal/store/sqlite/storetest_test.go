package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/openilink/openilink-hub/internal/store/sqlite"
	"github.com/openilink/openilink-hub/internal/store/storetest"
)

func testSQLiteStore(t *testing.T) *sqlite.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteStore(t *testing.T) {
	db := testSQLiteStore(t)
	storetest.RunAll(t, db)
}
