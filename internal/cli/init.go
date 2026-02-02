package cli

import (
	"encoding/json"
	"fmt"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/spf13/cobra"
)

var initForce bool

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing database")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize wark for first-time use",
	Long: `Initialize wark by creating the ~/.wark/ directory and database.

This command:
- Creates ~/.wark/ directory if it doesn't exist
- Creates wark.db with the database schema
- Runs any pending migrations

Use --force to overwrite an existing database.`,
	RunE: runInit,
}

type initResult struct {
	Database string `json:"database"`
	Created  bool   `json:"created"`
	Schema   int64  `json:"schema_version"`
}

func runInit(cmd *cobra.Command, args []string) error {
	dbPath := GetDBPath()

	// Check if database already exists
	if db.Exists(dbPath) && !initForce {
		if IsJSON() {
			result := initResult{
				Database: dbPath,
				Created:  false,
			}
			if dbPath == "" {
				result.Database = db.DefaultDBPath
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		displayPath := dbPath
		if displayPath == "" {
			displayPath = db.DefaultDBPath
		}
		return fmt.Errorf("database already exists at %s (use --force to overwrite)", displayPath)
	}

	// Delete existing database if force is set
	if initForce && db.Exists(dbPath) {
		VerboseOutput("Removing existing database...\n")
		if err := db.Delete(dbPath); err != nil {
			return fmt.Errorf("failed to remove existing database: %w", err)
		}
	}

	// Open/create the database
	VerboseOutput("Creating database...\n")
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer database.Close()

	// Run migrations
	VerboseOutput("Running migrations...\n")
	if err := database.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get schema version
	version, err := database.MigrationStatus()
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	displayPath := database.Path()

	if IsJSON() {
		result := initResult{
			Database: displayPath,
			Created:  true,
			Schema:   version,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Initialized wark database at %s", displayPath)
	OutputLine("Schema version: %d", version)

	return nil
}
