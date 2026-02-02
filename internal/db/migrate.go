package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func init() {
	// Configure goose to use the embedded migrations
	goose.SetBaseFS(embedMigrations)
}

// Migrate runs all pending database migrations.
func (d *DB) Migrate() error {
	return Migrate(d.DB)
}

// Migrate runs all pending database migrations on the given database connection.
func Migrate(db *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// MigrateDown rolls back the last migration.
func (d *DB) MigrateDown() error {
	return MigrateDown(d.DB)
}

// MigrateDown rolls back the last migration on the given database connection.
func MigrateDown(db *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Down(db, "migrations"); err != nil {
		return fmt.Errorf("failed to roll back migration: %w", err)
	}

	return nil
}

// MigrateReset rolls back all migrations.
func (d *DB) MigrateReset() error {
	return MigrateReset(d.DB)
}

// MigrateReset rolls back all migrations on the given database connection.
func MigrateReset(db *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Reset(db, "migrations"); err != nil {
		return fmt.Errorf("failed to reset migrations: %w", err)
	}

	return nil
}

// MigrationStatus returns the current migration version.
func (d *DB) MigrationStatus() (int64, error) {
	return MigrationStatus(d.DB)
}

// MigrationStatus returns the current migration version for the given database.
func MigrationStatus(db *sql.DB) (int64, error) {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return 0, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, nil
}
