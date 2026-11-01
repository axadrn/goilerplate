package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"
)

func setupGoose() error {
	err := goose.SetDialect(databaseDialect)
	if err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to get migrations directory: %w", err)
	}

	goose.SetBaseFS(migrationsDir)
	return nil
}

func RunMigrations(db *sql.DB) error {
	err := setupGoose()
	if err != nil {
		return err
	}

	err = goose.Up(db, ".")
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("migrations completed successfully")
	return nil
}

func MigrateDown(db *sql.DB) error {
	err := setupGoose()
	if err != nil {
		return err
	}

	err = goose.Down(db, ".")
	if err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	slog.Info("rolled back one migration")
	return nil
}
