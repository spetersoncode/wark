// Package config provides configuration file and environment variable support for wark.
//
// Configuration priority (highest to lowest):
//  1. Command-line flags
//  2. Environment variables
//  3. Config file (~/.wark/config.toml)
//  4. Built-in defaults
package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Config represents the wark configuration.
type Config struct {
	// DB is the path to the database file.
	// Default: ~/.wark/wark.db
	DB string `toml:"db"`

	// NoColor disables colored output.
	// Default: false
	NoColor bool `toml:"no_color"`

	// DefaultProject is the default project key for commands.
	// Used when --project/-p flag is not specified.
	DefaultProject string `toml:"default_project"`

	// DefaultWorkerID is the default worker ID for claims.
	// Used when --worker-id flag is not specified.
	DefaultWorkerID string `toml:"default_worker_id"`

	// ClaimDuration is the default claim duration in minutes.
	// Used when --duration flag is not specified.
	// Default: 60
	ClaimDuration int `toml:"claim_duration"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		DB:            "", // Empty means use db.DefaultDBPath
		NoColor:       false,
		ClaimDuration: 60,
	}
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".wark", "config.toml")
}

// Load loads configuration from the config file and environment variables.
// Environment variables take precedence over file settings.
// Returns default config if the config file doesn't exist.
func Load() (*Config, error) {
	return LoadFromPath(DefaultConfigPath())
}

// LoadFromPath loads configuration from a specific file path.
// Environment variables take precedence over file settings.
// Returns default config if the config file doesn't exist.
func LoadFromPath(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if _, err := toml.DecodeFile(configPath, cfg); err != nil {
				return nil, err
			}
		}
		// If file doesn't exist, just continue with defaults
	}

	// Apply environment variable overrides
	cfg.applyEnv()

	return cfg, nil
}

// applyEnv applies environment variable overrides to the config.
func (c *Config) applyEnv() {
	// Check WARK_DB first
	if db := os.Getenv("WARK_DB"); db != "" {
		c.DB = db
	}
	// WARK_DB_PATH takes precedence over WARK_DB (more explicit name)
	if dbPath := os.Getenv("WARK_DB_PATH"); dbPath != "" {
		c.DB = dbPath
	}

	// WARK_NO_COLOR - any value means true
	if _, ok := os.LookupEnv("WARK_NO_COLOR"); ok {
		c.NoColor = true
	}

	if project := os.Getenv("WARK_DEFAULT_PROJECT"); project != "" {
		c.DefaultProject = project
	}

	if workerID := os.Getenv("WARK_DEFAULT_WORKER_ID"); workerID != "" {
		c.DefaultWorkerID = workerID
	}

	if duration := os.Getenv("WARK_CLAIM_DURATION"); duration != "" {
		if d, err := strconv.Atoi(duration); err == nil && d > 0 {
			c.ClaimDuration = d
		}
	}
}

// GetDB returns the database path, using the default if not set.
func (c *Config) GetDB() string {
	if c.DB != "" {
		return c.DB
	}
	return "" // Return empty to signal use of db.DefaultDBPath
}

// SampleConfig returns a sample configuration file content.
func SampleConfig() string {
	return `# Wark Configuration File
# Location: ~/.wark/config.toml
#
# Configuration priority (highest to lowest):
#   1. Command-line flags
#   2. Environment variables (WARK_*)
#   3. This config file
#   4. Built-in defaults

# Path to the database file
# Default: ~/.wark/wark.db
# Environment: WARK_DB or WARK_DB_PATH (WARK_DB_PATH takes precedence)
# db = "/path/to/wark.db"

# Disable colored output
# Default: false
# Environment: WARK_NO_COLOR (any value = true)
# no_color = false

# Default project key for commands
# Used when --project/-p flag is not specified
# Environment: WARK_DEFAULT_PROJECT
# default_project = "MYPROJ"

# Default worker ID for claims
# Used when --worker-id flag is not specified
# Environment: WARK_DEFAULT_WORKER_ID
# default_worker_id = "agent-1"

# Default claim duration in minutes
# Used when --duration flag is not specified
# Default: 60
# Environment: WARK_CLAIM_DURATION
# claim_duration = 60
`
}

// WriteConfigFile writes the sample config file to the specified path.
// Creates parent directories if needed.
func WriteConfigFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(SampleConfig()), 0644)
}
