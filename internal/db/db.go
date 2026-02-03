// Package db provides database connection management for wark.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	// DefaultDBPath is the default location for the wark database.
	DefaultDBPath = "~/.wark/wark.db"
	// DefaultDBDir is the directory containing the database.
	DefaultDBDir = "~/.wark"
)

// DB wraps a sql.DB connection with wark-specific functionality.
type DB struct {
	*sql.DB
	path string
}

// Open opens or creates a wark database at the specified path.
// If path is empty, it uses the default path (~/.wark/wark.db).
func Open(path string) (*DB, error) {
	if path == "" {
		path = expandPath(DefaultDBPath)
	} else {
		path = expandPath(path)
	}

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the database with SQLite pragmas for better performance
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for SQLite
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)

	// Verify the connection works
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DB{DB: db, path: path}, nil
}

// Path returns the file path of the database.
func (d *DB) Path() string {
	return d.path
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d.DB == nil {
		return nil
	}
	return d.DB.Close()
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) == 0 {
		return path
	}

	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}

	return path
}

// Exists checks if the database file exists at the given path.
// If path is empty, it checks the default path.
func Exists(path string) bool {
	if path == "" {
		path = expandPath(DefaultDBPath)
	} else {
		path = expandPath(path)
	}

	_, err := os.Stat(path)
	return err == nil
}

// Delete removes the database file at the given path.
// If path is empty, it uses the default path.
func Delete(path string) error {
	if path == "" {
		path = expandPath(DefaultDBPath)
	} else {
		path = expandPath(path)
	}

	// Remove WAL and SHM files as well
	os.Remove(path + "-wal")
	os.Remove(path + "-shm")

	return os.Remove(path)
}

// FormatTime formats a time.Time as an RFC 3339 string for SQLite compatibility.
// This ensures timestamps can be parsed by SQLite's julianday() and other date functions.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// FormatTimePtr formats an optional time.Time as an RFC 3339 string, or returns nil.
func FormatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return FormatTime(*t)
}

// NowRFC3339 returns the current time formatted as RFC 3339 for SQLite.
func NowRFC3339() string {
	return FormatTime(time.Now())
}
