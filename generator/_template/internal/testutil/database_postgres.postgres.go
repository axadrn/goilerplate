package testutil

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/db"
	"{{.ModulePath}}/internal/testpostgres"
)

func NewDatabase(t testing.TB) *sqlx.DB {
	t.Helper()
	database, err := db.Init(testpostgres.DatabaseURL(t))
	if err != nil {
		t.Fatalf("initialize test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.RunMigrations(database.DB); err != nil {
		t.Fatalf("run test migrations: %v", err)
	}
	return database
}
