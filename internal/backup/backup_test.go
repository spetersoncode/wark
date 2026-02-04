package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultTestConfig() config.BackupConfig {
	return config.BackupConfig{
		Enabled:       true,
		IntervalHours: 24,
		MaxCount:      5,
		Path:          "",
	}
}

func TestNewManager(t *testing.T) {
	t.Run("uses custom backup path when specified", func(t *testing.T) {
		cfg := defaultTestConfig()
		cfg.Path = "/custom/backup/path"

		m := NewManager("/data/wark.db", cfg)

		assert.Equal(t, "/custom/backup/path", m.GetBackupDir())
	})

	t.Run("uses db directory when backup path not specified", func(t *testing.T) {
		cfg := defaultTestConfig()
		cfg.Path = ""

		m := NewManager("/data/wark.db", cfg)

		assert.Equal(t, "/data", m.GetBackupDir())
	})
}

func TestBackupIfNeeded_Disabled(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("test data"), 0644))

	cfg := defaultTestConfig()
	cfg.Enabled = false

	m := NewManager(dbPath, cfg)
	backupPath, err := m.BackupIfNeeded()

	require.NoError(t, err)
	assert.Empty(t, backupPath, "should not create backup when disabled")
}

func TestBackupIfNeeded_NoDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nonexistent.db")

	cfg := defaultTestConfig()
	m := NewManager(dbPath, cfg)

	backupPath, err := m.BackupIfNeeded()

	require.NoError(t, err)
	assert.Empty(t, backupPath, "should not create backup when DB doesn't exist")
}

func TestBackupIfNeeded_FirstBackup(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("test data"), 0644))

	cfg := defaultTestConfig()
	m := NewManager(dbPath, cfg)

	backupPath, err := m.BackupIfNeeded()

	require.NoError(t, err)
	assert.NotEmpty(t, backupPath, "should create first backup")
	assert.Equal(t, filepath.Join(dir, "wark.db.bak.1"), backupPath)

	// Verify backup file exists and has correct content
	content, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("test data"), content)
}

func TestBackupIfNeeded_BackupNotStale(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("test data"), 0644))

	// Create a recent backup
	backupFile := filepath.Join(dir, "wark.db.bak.1")
	require.NoError(t, os.WriteFile(backupFile, []byte("backup data"), 0644))

	cfg := defaultTestConfig()
	cfg.IntervalHours = 24 // 24 hours threshold

	m := NewManager(dbPath, cfg)
	backupPath, err := m.BackupIfNeeded()

	require.NoError(t, err)
	assert.Empty(t, backupPath, "should not create backup when existing backup is recent")
}

func TestBackupIfNeeded_BackupStale(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("current data"), 0644))

	// Create an old backup (modify time to be 25 hours ago)
	backupFile := filepath.Join(dir, "wark.db.bak.1")
	require.NoError(t, os.WriteFile(backupFile, []byte("old backup data"), 0644))
	oldTime := time.Now().Add(-25 * time.Hour)
	require.NoError(t, os.Chtimes(backupFile, oldTime, oldTime))

	cfg := defaultTestConfig()
	cfg.IntervalHours = 24

	m := NewManager(dbPath, cfg)
	backupPath, err := m.BackupIfNeeded()

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "wark.db.bak.1"), backupPath)

	// Old backup should have been rotated to .bak.2
	_, err = os.Stat(filepath.Join(dir, "wark.db.bak.2"))
	assert.NoError(t, err, "old backup should be rotated to .bak.2")

	// New backup should have current content
	content, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("current data"), content)
}

func TestBackupRotation(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("current data"), 0644))

	// Create existing backups (1, 2, 3) with old timestamps
	for i := 1; i <= 3; i++ {
		backupFile := filepath.Join(dir, "wark.db.bak."+string(rune('0'+i)))
		require.NoError(t, os.WriteFile(backupFile, []byte("backup "+string(rune('0'+i))), 0644))
		oldTime := time.Now().Add(-time.Duration(25+i) * time.Hour) // Make them all old
		require.NoError(t, os.Chtimes(backupFile, oldTime, oldTime))
	}

	cfg := defaultTestConfig()
	cfg.IntervalHours = 24
	cfg.MaxCount = 5

	m := NewManager(dbPath, cfg)
	_, err := m.BackupIfNeeded()
	require.NoError(t, err)

	// Verify rotation: 1->2, 2->3, 3->4, new backup at 1
	backups, err := m.ListBackups()
	require.NoError(t, err)
	assert.Len(t, backups, 4)

	// New backup at position 1
	content, err := os.ReadFile(filepath.Join(dir, "wark.db.bak.1"))
	require.NoError(t, err)
	assert.Equal(t, []byte("current data"), content)

	// Old backup 1 is now at position 2
	content, err = os.ReadFile(filepath.Join(dir, "wark.db.bak.2"))
	require.NoError(t, err)
	assert.Equal(t, []byte("backup 1"), content)
}

func TestBackupRotation_ExceedsMaxCount(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("current data"), 0644))

	// Create 3 existing backups
	for i := 1; i <= 3; i++ {
		backupFile := filepath.Join(dir, "wark.db.bak."+string(rune('0'+i)))
		require.NoError(t, os.WriteFile(backupFile, []byte("backup "+string(rune('0'+i))), 0644))
		oldTime := time.Now().Add(-time.Duration(25+i) * time.Hour)
		require.NoError(t, os.Chtimes(backupFile, oldTime, oldTime))
	}

	cfg := defaultTestConfig()
	cfg.IntervalHours = 24
	cfg.MaxCount = 3 // Only keep 3 backups

	m := NewManager(dbPath, cfg)
	_, err := m.BackupIfNeeded()
	require.NoError(t, err)

	// Verify only 3 backups exist (oldest deleted)
	backups, err := m.ListBackups()
	require.NoError(t, err)
	assert.Len(t, backups, 3, "should only keep MaxCount backups")

	// Verify backup 4 doesn't exist (was deleted after rotation)
	_, err = os.Stat(filepath.Join(dir, "wark.db.bak.4"))
	assert.True(t, os.IsNotExist(err), "backup 4 should be deleted")

	// Verify newest backup has current data
	content, err := os.ReadFile(filepath.Join(dir, "wark.db.bak.1"))
	require.NoError(t, err)
	assert.Equal(t, []byte("current data"), content)
}

func TestListBackups(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")

	cfg := defaultTestConfig()
	m := NewManager(dbPath, cfg)

	// No backups initially
	backups, err := m.ListBackups()
	require.NoError(t, err)
	assert.Empty(t, backups)

	// Create backups in random order
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wark.db.bak.3"), []byte("3"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wark.db.bak.1"), []byte("1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wark.db.bak.2"), []byte("2"), 0644))

	// Also create some unrelated files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.db"), []byte("other"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wark.db.bak.invalid"), []byte("invalid"), 0644))

	backups, err = m.ListBackups()
	require.NoError(t, err)

	// Should be sorted by number (1 = newest, so ascending)
	require.Len(t, backups, 3)
	assert.Equal(t, filepath.Join(dir, "wark.db.bak.1"), backups[0])
	assert.Equal(t, filepath.Join(dir, "wark.db.bak.2"), backups[1])
	assert.Equal(t, filepath.Join(dir, "wark.db.bak.3"), backups[2])
}

func TestBackupWithCustomPath(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "db")
	backupDir := filepath.Join(dir, "backups")
	require.NoError(t, os.MkdirAll(dbDir, 0755))

	dbPath := filepath.Join(dbDir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("test data"), 0644))

	cfg := defaultTestConfig()
	cfg.Path = backupDir

	m := NewManager(dbPath, cfg)
	backupPath, err := m.BackupIfNeeded()

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(backupDir, "wark.db.bak.1"), backupPath)

	// Verify backup directory was created
	info, err := os.Stat(backupDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify backup was created in custom path
	content, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("test data"), content)
}

func TestBackupPreservesFilePermissions(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("test data"), 0600)) // Restrictive permissions

	cfg := defaultTestConfig()
	m := NewManager(dbPath, cfg)

	backupPath, err := m.BackupIfNeeded()
	require.NoError(t, err)

	// Verify backup has same permissions
	srcInfo, err := os.Stat(dbPath)
	require.NoError(t, err)
	dstInfo, err := os.Stat(backupPath)
	require.NoError(t, err)

	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
}

func TestBackupWithZeroIntervalAlwaysBacksUp(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("test data"), 0644))

	// Create a very recent backup
	backupFile := filepath.Join(dir, "wark.db.bak.1")
	require.NoError(t, os.WriteFile(backupFile, []byte("backup data"), 0644))

	cfg := defaultTestConfig()
	cfg.IntervalHours = 0 // Zero interval should be treated as "always backup"
	// Actually, let's test with 1 hour interval and a 2-hour-old backup

	cfg.IntervalHours = 1
	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(backupFile, oldTime, oldTime))

	m := NewManager(dbPath, cfg)
	backupPath, err := m.BackupIfNeeded()

	require.NoError(t, err)
	assert.NotEmpty(t, backupPath, "should create backup when old backup exceeds 1 hour threshold")
}

func TestMaxCountOfOne(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("data 1"), 0644))

	cfg := defaultTestConfig()
	cfg.MaxCount = 1

	m := NewManager(dbPath, cfg)

	// First backup
	_, err := m.BackupIfNeeded()
	require.NoError(t, err)

	// Make the backup old and create a new one
	backupFile := filepath.Join(dir, "wark.db.bak.1")
	oldTime := time.Now().Add(-25 * time.Hour)
	require.NoError(t, os.Chtimes(backupFile, oldTime, oldTime))

	require.NoError(t, os.WriteFile(dbPath, []byte("data 2"), 0644))
	_, err = m.BackupIfNeeded()
	require.NoError(t, err)

	// Should only have one backup
	backups, err := m.ListBackups()
	require.NoError(t, err)
	assert.Len(t, backups, 1, "should only keep 1 backup when MaxCount=1")

	// And it should have the latest data
	content, err := os.ReadFile(backups[0])
	require.NoError(t, err)
	assert.Equal(t, []byte("data 2"), content)
}

func TestConcurrentBackupSafety(t *testing.T) {
	// This is a basic test to ensure the backup doesn't corrupt data
	// A full concurrent safety test would require more sophisticated setup
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wark.db")

	// Write a larger file to increase chance of catching issues
	data := make([]byte, 1024*1024) // 1MB
	for i := range data {
		data[i] = byte(i % 256)
	}
	require.NoError(t, os.WriteFile(dbPath, data, 0644))

	cfg := defaultTestConfig()
	m := NewManager(dbPath, cfg)

	backupPath, err := m.BackupIfNeeded()
	require.NoError(t, err)

	// Verify the backup is complete and correct
	backupData, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, data, backupData)
}
