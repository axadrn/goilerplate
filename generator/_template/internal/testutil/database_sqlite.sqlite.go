package testutil

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/db"
)

func NewDatabase(t testing.TB) *sqlx.DB {
	t.Helper()
	connection := filepath.Join(t.TempDir(), "test.db") + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	database, err := db.Init(connection)
	if err != nil {
		t.Fatalf("initialize test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.RunMigrations(database.DB); err != nil {
		t.Fatalf("run test migrations: %v", err)
	}
	return database
}
