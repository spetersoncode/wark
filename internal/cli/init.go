package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/diogenes-ai-code/wark/internal/config"
	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	initForce      bool
	initWithConfig bool
)

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing database")
	initCmd.Flags().BoolVar(&initWithConfig, "with-config", false, "Create a sample config file")
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
- Optionally creates a sample config file (--with-config)

Use --force to overwrite an existing database.
Use --with-config to create a sample config file at ~/.wark/config.toml.`,
	RunE: runInit,
}

type initResult struct {
	Database   string `json:"database"`
	Created    bool   `json:"created"`
	Schema     int64  `json:"schema_version"`
	ConfigFile string `json:"config_file,omitempty"`
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
		displayPath := dbPath
		if displayPath == "" {
			displayPath = db.DefaultDBPath
		}

		// Check if database has data before destroying
		stats, err := getDBStats(dbPath)
		if err != nil {
			// If we can't read the database, just warn and continue
			VerboseOutput("Warning: could not check existing database for data: %v\n", err)
		} else if stats.hasData {
			// Database has data, require confirmation
			fmt.Fprintf(os.Stderr, "Database at %s contains data:\n", displayPath)
			fmt.Fprintf(os.Stderr, "  - %d project(s)\n", stats.projects)
			fmt.Fprintf(os.Stderr, "  - %d ticket(s)\n", stats.tickets)
			fmt.Fprintf(os.Stderr, "  - %d inbox message(s)\n", stats.inboxMessages)
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "This will be PERMANENTLY DESTROYED.")

			// Check if stdin is a TTY
			if term.IsTerminal(int(os.Stdin.Fd())) {
				// Interactive mode: require confirmation
				fmt.Fprint(os.Stderr, "Type 'yes' to confirm: ")
				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read confirmation: %w", err)
				}
				response = strings.TrimSpace(response)
				if response != "yes" {
					return fmt.Errorf("aborted: confirmation not received")
				}
			} else {
				// Non-interactive mode: refuse to destroy data without interactive confirmation
				return fmt.Errorf("cannot destroy database with data in non-interactive mode\n\nRun interactively to confirm, or use a fresh database path")
			}
		}

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

	// Optionally create sample config file
	var configPath string
	if initWithConfig {
		configPath = config.DefaultConfigPath()
		if _, err := os.Stat(configPath); err == nil && !initForce {
			VerboseOutput("Config file already exists at %s, skipping...\n", configPath)
			configPath = "" // Don't report it as created
		} else {
			VerboseOutput("Creating sample config file...\n")
			if err := config.WriteConfigFile(configPath); err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}
		}
	}

	if IsJSON() {
		result := initResult{
			Database:   displayPath,
			Created:    true,
			Schema:     version,
			ConfigFile: configPath,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Initialized wark database at %s", displayPath)
	OutputLine("Schema version: %d", version)
	if configPath != "" {
		OutputLine("Created sample config at %s", configPath)
	}

	return nil
}

// dbStats holds counts of data in the database
type dbStats struct {
	hasData       bool
	projects      int
	tickets       int
	inboxMessages int
}

// getDBStats checks if a database has data worth protecting
func getDBStats(dbPath string) (*dbStats, error) {
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	stats := &dbStats{}

	// Count projects
	row := database.QueryRow("SELECT COUNT(*) FROM projects")
	if err := row.Scan(&stats.projects); err != nil {
		// Table might not exist (old schema), treat as no data
		stats.projects = 0
	}

	// Count tickets
	row = database.QueryRow("SELECT COUNT(*) FROM tickets")
	if err := row.Scan(&stats.tickets); err != nil {
		stats.tickets = 0
	}

	// Count inbox messages
	row = database.QueryRow("SELECT COUNT(*) FROM inbox")
	if err := row.Scan(&stats.inboxMessages); err != nil {
		stats.inboxMessages = 0
	}

	stats.hasData = stats.projects > 0 || stats.tickets > 0 || stats.inboxMessages > 0
	return stats, nil
}
