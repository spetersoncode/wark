package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spetersoncode/wark/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDBPath_WithConfig(t *testing.T) {
	// Save original config
	origConfig := globalConfig
	defer func() { globalConfig = origConfig }()

	// Test with config value
	globalConfig = &config.Config{DB: "/config/path.db"}
	dbPath = "" // Reset flag
	assert.Equal(t, "/config/path.db", GetDBPath())

	// Test flag overrides config
	dbPath = "/flag/path.db"
	assert.Equal(t, "/flag/path.db", GetDBPath())

	// Reset
	dbPath = ""
}

func TestIsNoColor_WithConfig(t *testing.T) {
	// Save original values
	origConfig := globalConfig
	origNoColor := noColor
	defer func() {
		globalConfig = origConfig
		noColor = origNoColor
	}()

	// Test with config value
	globalConfig = &config.Config{NoColor: true}
	noColor = false
	assert.True(t, IsNoColor())

	// Test flag overrides config
	globalConfig = &config.Config{NoColor: false}
	noColor = true
	assert.True(t, IsNoColor())

	// Test config false, flag false
	globalConfig = &config.Config{NoColor: false}
	noColor = false
	assert.False(t, IsNoColor())
}

func TestGetDefaultProject(t *testing.T) {
	// Save original config
	origConfig := globalConfig
	defer func() { globalConfig = origConfig }()

	// Test with config value
	globalConfig = &config.Config{DefaultProject: "MYPROJ"}
	assert.Equal(t, "MYPROJ", GetDefaultProject())

	// Test with empty config
	globalConfig = &config.Config{}
	assert.Equal(t, "", GetDefaultProject())

	// Test with nil config
	globalConfig = nil
	assert.Equal(t, "", GetDefaultProject())
}

func TestGetDefaultWorkerID(t *testing.T) {
	// Save original config
	origConfig := globalConfig
	defer func() { globalConfig = origConfig }()

	// Test with config value
	globalConfig = &config.Config{DefaultWorkerID: "agent-1"}
	assert.Equal(t, "agent-1", GetDefaultWorkerID())

	// Test with empty config
	globalConfig = &config.Config{}
	assert.Equal(t, "", GetDefaultWorkerID())

	// Test with nil config
	globalConfig = nil
	assert.Equal(t, "", GetDefaultWorkerID())
}

func TestGetDefaultClaimDuration(t *testing.T) {
	// Save original config
	origConfig := globalConfig
	defer func() { globalConfig = origConfig }()

	// Test with config value
	globalConfig = &config.Config{ClaimDuration: 90}
	assert.Equal(t, 90, GetDefaultClaimDuration())

	// Test with zero config (should return default 30)
	globalConfig = &config.Config{ClaimDuration: 0}
	assert.Equal(t, 30, GetDefaultClaimDuration())

	// Test with nil config
	globalConfig = nil
	assert.Equal(t, 30, GetDefaultClaimDuration())
}

func TestGetProjectWithDefault(t *testing.T) {
	// Save original config
	origConfig := globalConfig
	defer func() { globalConfig = origConfig }()

	globalConfig = &config.Config{DefaultProject: "DEFAULT"}

	// Flag value should be used when provided
	assert.Equal(t, "PROVIDED", GetProjectWithDefault("PROVIDED"))

	// Config default should be used when flag is empty
	assert.Equal(t, "DEFAULT", GetProjectWithDefault(""))

	// Empty when no flag and no config
	globalConfig = &config.Config{}
	assert.Equal(t, "", GetProjectWithDefault(""))
}

func TestGetConfig(t *testing.T) {
	// Save original config
	origConfig := globalConfig
	defer func() { globalConfig = origConfig }()

	// Test with configured values
	globalConfig = &config.Config{
		DB:              "/custom/path.db",
		NoColor:         true,
		DefaultProject:  "TEST",
		DefaultWorkerID: "worker-1",
		ClaimDuration:   120,
	}

	cfg := GetConfig()
	assert.Equal(t, "/custom/path.db", cfg.DB)
	assert.True(t, cfg.NoColor)
	assert.Equal(t, "TEST", cfg.DefaultProject)
	assert.Equal(t, "worker-1", cfg.DefaultWorkerID)
	assert.Equal(t, 120, cfg.ClaimDuration)

	// Test with nil config returns defaults
	globalConfig = nil
	cfg = GetConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, 60, cfg.ClaimDuration)
}

func TestConfigWithEnvOverrides(t *testing.T) {
	// Save original config
	origConfig := globalConfig
	defer func() { globalConfig = origConfig }()

	// Create a temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
db = "/file/db/path.db"
default_project = "FILEPROJ"
default_worker_id = "file-worker"
claim_duration = 45
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	// Set environment overrides
	t.Setenv("WARK_DB", "/env/db/path.db")
	t.Setenv("WARK_DEFAULT_PROJECT", "ENVPROJ")

	// Load config from the file (env should override)
	cfg, err := config.LoadFromPath(configPath)
	require.NoError(t, err)
	globalConfig = cfg

	// Verify env overrides file values
	assert.Equal(t, "/env/db/path.db", GetDBPath())
	assert.Equal(t, "ENVPROJ", GetDefaultProject())

	// Verify file values are used when not in env
	assert.Equal(t, "file-worker", GetDefaultWorkerID())
	assert.Equal(t, 45, GetDefaultClaimDuration())
}

func TestConfigFlagPriority(t *testing.T) {
	// Save original config
	origConfig := globalConfig
	origDBPath := dbPath
	defer func() {
		globalConfig = origConfig
		dbPath = origDBPath
	}()

	// Set config value
	globalConfig = &config.Config{DB: "/config/path.db"}

	// Without flag, should use config
	dbPath = ""
	assert.Equal(t, "/config/path.db", GetDBPath())

	// With flag, should override config
	dbPath = "/flag/path.db"
	assert.Equal(t, "/flag/path.db", GetDBPath())
}
