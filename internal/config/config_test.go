package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "", cfg.DB)
	assert.False(t, cfg.NoColor)
	assert.Equal(t, "", cfg.DefaultProject)
	assert.Equal(t, "", cfg.DefaultWorkerID)
	assert.Equal(t, 60, cfg.ClaimDuration)
}

func TestLoadFromPath_MissingFile(t *testing.T) {
	// Loading from a non-existent file should return defaults
	cfg, err := LoadFromPath("/nonexistent/path/config.toml")
	require.NoError(t, err)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoadFromPath_ValidFile(t *testing.T) {
	// Create a temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
db = "/custom/db/path.db"
no_color = true
default_project = "TESTPROJ"
default_worker_id = "worker-123"
claim_duration = 120
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(configPath)
	require.NoError(t, err)

	assert.Equal(t, "/custom/db/path.db", cfg.DB)
	assert.True(t, cfg.NoColor)
	assert.Equal(t, "TESTPROJ", cfg.DefaultProject)
	assert.Equal(t, "worker-123", cfg.DefaultWorkerID)
	assert.Equal(t, 120, cfg.ClaimDuration)
}

func TestLoadFromPath_PartialFile(t *testing.T) {
	// Config file with only some values
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
default_project = "MYPROJ"
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(configPath)
	require.NoError(t, err)

	// Specified value
	assert.Equal(t, "MYPROJ", cfg.DefaultProject)
	// Default values
	assert.Equal(t, "", cfg.DB)
	assert.False(t, cfg.NoColor)
	assert.Equal(t, "", cfg.DefaultWorkerID)
	assert.Equal(t, 60, cfg.ClaimDuration)
}

func TestLoadFromPath_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `invalid toml {{{{ content`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	_, err = LoadFromPath(configPath)
	assert.Error(t, err)
}

func TestLoadFromPath_EmptyPath(t *testing.T) {
	cfg, err := LoadFromPath("")
	require.NoError(t, err)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestEnvOverrides(t *testing.T) {
	// Create a temp config file with values
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
db = "/file/db/path.db"
no_color = false
default_project = "FILEPROJ"
default_worker_id = "file-worker"
claim_duration = 30
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	// Set environment variables
	t.Setenv("WARK_DB", "/env/db/path.db")
	t.Setenv("WARK_NO_COLOR", "1")
	t.Setenv("WARK_DEFAULT_PROJECT", "ENVPROJ")
	t.Setenv("WARK_DEFAULT_WORKER_ID", "env-worker")
	t.Setenv("WARK_CLAIM_DURATION", "90")

	cfg, err := LoadFromPath(configPath)
	require.NoError(t, err)

	// Environment variables should override file values
	assert.Equal(t, "/env/db/path.db", cfg.DB)
	assert.True(t, cfg.NoColor)
	assert.Equal(t, "ENVPROJ", cfg.DefaultProject)
	assert.Equal(t, "env-worker", cfg.DefaultWorkerID)
	assert.Equal(t, 90, cfg.ClaimDuration)
}

func TestEnvOverrides_PartialEnv(t *testing.T) {
	// Create a temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
db = "/file/db/path.db"
default_project = "FILEPROJ"
claim_duration = 30
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	// Set only some environment variables
	t.Setenv("WARK_DB", "/env/db/path.db")

	cfg, err := LoadFromPath(configPath)
	require.NoError(t, err)

	// WARK_DB should override
	assert.Equal(t, "/env/db/path.db", cfg.DB)
	// File values should be used for others
	assert.Equal(t, "FILEPROJ", cfg.DefaultProject)
	assert.Equal(t, 30, cfg.ClaimDuration)
}

func TestEnvOverrides_NoColorAnyValue(t *testing.T) {
	// WARK_NO_COLOR with any value should enable no_color
	testCases := []string{"1", "true", "yes", "anything", ""}

	for _, val := range testCases {
		t.Run("value="+val, func(t *testing.T) {
			t.Setenv("WARK_NO_COLOR", val)
			cfg, err := LoadFromPath("")
			require.NoError(t, err)
			assert.True(t, cfg.NoColor, "WARK_NO_COLOR=%q should enable no_color", val)
		})
	}
}

func TestEnvOverrides_InvalidClaimDuration(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `claim_duration = 45`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	// Invalid duration should be ignored
	t.Setenv("WARK_CLAIM_DURATION", "invalid")
	cfg, err := LoadFromPath(configPath)
	require.NoError(t, err)
	assert.Equal(t, 45, cfg.ClaimDuration)

	// Zero duration should be ignored
	t.Setenv("WARK_CLAIM_DURATION", "0")
	cfg, err = LoadFromPath(configPath)
	require.NoError(t, err)
	assert.Equal(t, 45, cfg.ClaimDuration)

	// Negative duration should be ignored
	t.Setenv("WARK_CLAIM_DURATION", "-10")
	cfg, err = LoadFromPath(configPath)
	require.NoError(t, err)
	assert.Equal(t, 45, cfg.ClaimDuration)
}

func TestGetDB(t *testing.T) {
	cfg := &Config{DB: "/custom/path.db"}
	assert.Equal(t, "/custom/path.db", cfg.GetDB())

	cfg = &Config{DB: ""}
	assert.Equal(t, "", cfg.GetDB())
}

func TestWriteConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "subdir", "config.toml")

	err := WriteConfigFile(configPath)
	require.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Wark Configuration File")
	assert.Contains(t, string(content), "db =")
	assert.Contains(t, string(content), "no_color")
	assert.Contains(t, string(content), "default_project")
	assert.Contains(t, string(content), "default_worker_id")
	assert.Contains(t, string(content), "claim_duration")
}

func TestSampleConfig(t *testing.T) {
	sample := SampleConfig()
	assert.Contains(t, sample, "Wark Configuration File")
	assert.Contains(t, sample, "WARK_DB")
	assert.Contains(t, sample, "WARK_NO_COLOR")
	assert.Contains(t, sample, "WARK_DEFAULT_PROJECT")
	assert.Contains(t, sample, "WARK_DEFAULT_WORKER_ID")
	assert.Contains(t, sample, "WARK_CLAIM_DURATION")
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	assert.Contains(t, path, ".wark")
	assert.Contains(t, path, "config.toml")
}

func TestPriorityOrder(t *testing.T) {
	// This test verifies the priority order:
	// 1. Environment variables
	// 2. Config file
	// 3. Built-in defaults

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	// Test with all three levels

	// Case 1: No file, no env -> defaults
	cfg, err := LoadFromPath(filepath.Join(dir, "nonexistent.toml"))
	require.NoError(t, err)
	assert.Equal(t, 60, cfg.ClaimDuration) // default

	// Case 2: File set, no env -> file value
	content := `claim_duration = 45`
	err = os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err = LoadFromPath(configPath)
	require.NoError(t, err)
	assert.Equal(t, 45, cfg.ClaimDuration) // file

	// Case 3: File set, env set -> env value
	t.Setenv("WARK_CLAIM_DURATION", "90")
	cfg, err = LoadFromPath(configPath)
	require.NoError(t, err)
	assert.Equal(t, 90, cfg.ClaimDuration) // env overrides file
}
