// Package backup provides automatic database backup functionality for wark.
//
// The backup system creates rotating backups of the SQLite database on startup
// if the last backup is older than a configurable threshold. Backups are named
// wark.db.bak.1, wark.db.bak.2, etc., where 1 is the most recent.
package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/diogenes-ai-code/wark/internal/config"
)

const (
	// BackupPrefix is the prefix for backup files.
	BackupPrefix = "wark.db.bak."
)

// Manager handles database backup operations.
type Manager struct {
	dbPath    string
	backupDir string
	cfg       config.BackupConfig
}

// NewManager creates a new backup manager.
// dbPath is the path to the database file.
// cfg contains backup configuration settings.
func NewManager(dbPath string, cfg config.BackupConfig) *Manager {
	backupDir := cfg.Path
	if backupDir == "" {
		backupDir = filepath.Dir(dbPath)
	}

	return &Manager{
		dbPath:    dbPath,
		backupDir: backupDir,
		cfg:       cfg,
	}
}

// BackupIfNeeded checks if a backup is needed and creates one if so.
// Returns the path to the new backup file if created, or empty string if not needed.
// Returns an error only for unexpected failures (not for "backup not needed").
func (m *Manager) BackupIfNeeded() (string, error) {
	if !m.cfg.Enabled {
		return "", nil
	}

	// Check if the database file exists
	if _, err := os.Stat(m.dbPath); os.IsNotExist(err) {
		return "", nil // No database to backup
	}

	// Check if backup is needed based on threshold
	needed, err := m.isBackupNeeded()
	if err != nil {
		return "", fmt.Errorf("checking if backup needed: %w", err)
	}

	if !needed {
		return "", nil
	}

	// Create the backup
	backupPath, err := m.createBackup()
	if err != nil {
		return "", fmt.Errorf("creating backup: %w", err)
	}

	return backupPath, nil
}

// isBackupNeeded returns true if a new backup should be created.
func (m *Manager) isBackupNeeded() (bool, error) {
	lastBackupTime, err := m.getLastBackupTime()
	if err != nil {
		return false, err
	}

	// No existing backup = definitely need one
	if lastBackupTime.IsZero() {
		return true, nil
	}

	// Check against threshold
	threshold := time.Duration(m.cfg.IntervalHours) * time.Hour
	return time.Since(lastBackupTime) > threshold, nil
}

// getLastBackupTime returns the modification time of the most recent backup.
// Returns zero time if no backups exist.
func (m *Manager) getLastBackupTime() (time.Time, error) {
	backups, err := m.listBackups()
	if err != nil {
		return time.Time{}, err
	}

	if len(backups) == 0 {
		return time.Time{}, nil
	}

	// Backups are sorted newest first, so check the first one
	info, err := os.Stat(backups[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("stat backup file: %w", err)
	}

	return info.ModTime(), nil
}

// listBackups returns paths to existing backup files, sorted newest first.
func (m *Manager) listBackups() ([]string, error) {
	entries, err := os.ReadDir(m.backupDir)
	if os.IsNotExist(err) {
		return nil, nil // No backup directory yet
	}
	if err != nil {
		return nil, fmt.Errorf("reading backup directory: %w", err)
	}

	var backups []backupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, BackupPrefix) {
			continue
		}

		// Extract backup number
		numStr := strings.TrimPrefix(name, BackupPrefix)
		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue // Not a valid backup file name
		}

		backups = append(backups, backupFile{
			path:   filepath.Join(m.backupDir, name),
			number: num,
		})
	}

	// Sort by number (1 = newest, so ascending order puts newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].number < backups[j].number
	})

	paths := make([]string, len(backups))
	for i, b := range backups {
		paths[i] = b.path
	}

	return paths, nil
}

type backupFile struct {
	path   string
	number int
}

// createBackup creates a new backup file and rotates existing ones.
func (m *Manager) createBackup() (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(m.backupDir, 0755); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	// Rotate existing backups first
	if err := m.rotateBackups(); err != nil {
		return "", fmt.Errorf("rotating backups: %w", err)
	}

	// Copy the database to backup.1
	backupPath := filepath.Join(m.backupDir, BackupPrefix+"1")
	if err := copyFile(m.dbPath, backupPath); err != nil {
		return "", fmt.Errorf("copying database: %w", err)
	}

	return backupPath, nil
}

// rotateBackups rotates existing backup files and deletes old ones.
// After rotation: bak.1 -> bak.2, bak.2 -> bak.3, etc.
// Backups exceeding MaxCount are deleted.
func (m *Manager) rotateBackups() error {
	backups, err := m.listBackups()
	if err != nil {
		return err
	}

	// Process in reverse order (oldest first) to avoid overwriting
	for i := len(backups) - 1; i >= 0; i-- {
		path := backups[i]
		name := filepath.Base(path)
		numStr := strings.TrimPrefix(name, BackupPrefix)
		num, _ := strconv.Atoi(numStr) // Already validated in listBackups

		newNum := num + 1
		if newNum > m.cfg.MaxCount {
			// Delete backups that exceed the limit
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("deleting old backup %s: %w", path, err)
			}
		} else {
			// Rename to new number
			newPath := filepath.Join(m.backupDir, fmt.Sprintf("%s%d", BackupPrefix, newNum))
			if err := os.Rename(path, newPath); err != nil {
				return fmt.Errorf("renaming backup %s to %s: %w", path, newPath, err)
			}
		}
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copying data: %w", err)
	}

	// Sync to ensure data is written
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("syncing destination: %w", err)
	}

	return nil
}

// ListBackups returns the paths to all existing backup files, newest first.
// This is a public method for inspection/debugging.
func (m *Manager) ListBackups() ([]string, error) {
	return m.listBackups()
}

// GetBackupDir returns the directory where backups are stored.
func (m *Manager) GetBackupDir() string {
	return m.backupDir
}
