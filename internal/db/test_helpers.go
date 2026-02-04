package db

import (
	"database/sql"
	"testing"
)

// NewTestDB creates an in-memory SQLite database for testing.
//
// IMPORTANT: Always use this function in tests, never use file-based databases.
// Using file-based databases in tests risks accidentally destroying production data
// if the test database path isn't properly isolated.
//
// Example:
//
//	func TestSomething(t *testing.T) {
//	    db := NewTestDB(t)
//	    defer db.Close()
//
//	    // Use db for testing...
//	}
func NewTestDB(t *testing.T) *DB {
	t.Helper()

	// Use in-memory database with foreign keys enabled
	sqlDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Run migrations
	if err := Migrate(sqlDB); err != nil {
		sqlDB.Close()
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return &DB{DB: sqlDB, path: ":memory:"}
}

// NewTestSqlDB creates an in-memory SQLite sql.DB for testing.
// This is for tests that need the raw *sql.DB instead of *db.DB.
//
// Deprecated: Prefer NewTestDB when possible. This exists for backward
// compatibility with tests that use *sql.DB directly.
func NewTestSqlDB(t *testing.T) *sql.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Run migrations
	if err := Migrate(sqlDB); err != nil {
		sqlDB.Close()
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return sqlDB
}
