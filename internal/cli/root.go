package cli

import (
	"fmt"
	"os"

	"github.com/spetersoncode/wark/internal/backup"
	"github.com/spetersoncode/wark/internal/config"
	"github.com/spetersoncode/wark/internal/db"
	"github.com/spf13/cobra"
)

// Version information (set at build time via ldflags)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Global flags
var (
	dbPath  string
	jsonOut bool
	quiet   bool
	verbose bool
	noColor bool
)

// Global configuration (loaded once at startup)
var globalConfig *config.Config

// Exit codes as per spec
const (
	ExitSuccess           = 0
	ExitGeneralError      = 1
	ExitInvalidArgs       = 2
	ExitNotFound          = 3
	ExitStateError        = 4
	ExitDBError           = 5
	ExitConcurrentConflict = 6
)

// skipBackupCommands lists commands that should not trigger automatic backup.
// These are either commands that don't need a database, or that initialize it.
var skipBackupCommands = map[string]bool{
	"help":    true,
	"version": true,
	"init":    true,
}

var rootCmd = &cobra.Command{
	Use:   "wark",
	Short: "Local-first CLI task management for AI agent orchestration",
	Long: `Wark is a command-line task management tool inspired by Jira,
purpose-built for coordinating AI coding agents.

It provides project-based organization, dependency-aware ticket management,
claim-based work distribution, and human-in-the-loop support.

Use "wark init" to initialize a new wark database.
Use "wark --help" to see all available commands.`,
	Version:       Version,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return runAutoBackup(cmd)
	},
}

func init() {
	// Load global configuration at startup
	var err error
	globalConfig, err = config.Load()
	if err != nil {
		// If config file is invalid, print warning but continue with defaults
		fmt.Fprintf(os.Stderr, "Warning: failed to load config file: %v\n", err)
		globalConfig = config.DefaultConfig()
	}

	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to database file (default ~/.wark/wark.db)")
	rootCmd.PersistentFlags().BoolVarP(&jsonOut, "json", "j", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Set version template for --version flag
	rootCmd.SetVersionTemplate(fmt.Sprintf("wark %s (%s, %s)\n", Version, shortCommit(), shortDate()))

	// Add commands
	rootCmd.AddCommand(versionCmd)
}

// shortCommit returns the first 7 characters of the git commit hash
func shortCommit() string {
	if len(GitCommit) >= 7 {
		return GitCommit[:7]
	}
	return GitCommit
}

// shortDate returns just the date portion of BuildDate (YYYY-MM-DD)
func shortDate() string {
	if len(BuildDate) >= 10 {
		return BuildDate[:10]
	}
	return BuildDate
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// runAutoBackup performs automatic backup if needed before command execution.
// It skips backup for commands that don't need it (help, version, init).
func runAutoBackup(cmd *cobra.Command) error {
	// Skip for certain commands
	cmdName := cmd.Name()
	if skipBackupCommands[cmdName] {
		return nil
	}

	// Skip if no config loaded
	if globalConfig == nil {
		return nil
	}

	// Skip if backups are disabled
	if !globalConfig.Backup.Enabled {
		return nil
	}

	// Get the database path
	dbPath := GetDBPath()
	if dbPath == "" {
		dbPath = db.DefaultDBPath
	}
	// Expand ~ in the path
	dbPath = expandPath(dbPath)

	// Check if database exists - no point backing up a non-existent database
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil
	}

	// Create backup manager and run backup if needed
	mgr := backup.NewManager(dbPath, globalConfig.Backup)
	backupPath, err := mgr.BackupIfNeeded()
	if err != nil {
		// Log warning but don't fail the command
		VerboseOutput("Warning: automatic backup failed: %v\n", err)
		return nil
	}

	if backupPath != "" && verbose {
		VerboseOutput("Created backup: %s\n", backupPath)
	}

	return nil
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
		return home + path[1:]
	}

	return path
}

// GetDBPath returns the database path from flags, config, or default.
// Priority: flag > env > config file > default
func GetDBPath() string {
	// Command-line flag has highest priority
	if dbPath != "" {
		return dbPath
	}
	// Config already handles env > file > default
	if globalConfig != nil {
		return globalConfig.GetDB()
	}
	return "" // Will use default in db.Open
}

// IsJSON returns whether JSON output is requested
func IsJSON() bool {
	return jsonOut
}

// IsNoColor returns whether colored output should be disabled.
// Priority: flag > env > config file > default
func IsNoColor() bool {
	// Command-line flag has highest priority
	if noColor {
		return true
	}
	// Config already handles env > file > default
	if globalConfig != nil {
		return globalConfig.NoColor
	}
	return false
}

// GetDefaultProject returns the default project from config.
func GetDefaultProject() string {
	if globalConfig != nil {
		return globalConfig.DefaultProject
	}
	return ""
}

// GetDefaultWorkerID returns the default worker ID from config.
func GetDefaultWorkerID() string {
	if globalConfig != nil {
		return globalConfig.DefaultWorkerID
	}
	return ""
}

// GetDefaultClaimDuration returns the default claim duration in minutes from config.
func GetDefaultClaimDuration() int {
	if globalConfig != nil && globalConfig.ClaimDuration > 0 {
		return globalConfig.ClaimDuration
	}
	return 30
}

// GetConfig returns the global configuration.
// This should only be used when direct access to all config values is needed.
func GetConfig() *config.Config {
	if globalConfig != nil {
		return globalConfig
	}
	return config.DefaultConfig()
}

// GetProjectWithDefault returns the provided project or the default from config.
// Use this when getting a project value that may come from flag or config.
func GetProjectWithDefault(flagProject string) string {
	if flagProject != "" {
		return flagProject
	}
	return GetDefaultProject()
}

// IsQuiet returns whether quiet mode is enabled
func IsQuiet() bool {
	return quiet
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}

// Output prints to stdout unless quiet mode is enabled
func Output(format string, args ...interface{}) {
	if !quiet {
		fmt.Printf(format, args...)
	}
}

// OutputLine prints a line to stdout unless quiet mode is enabled
func OutputLine(format string, args ...interface{}) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// VerboseOutput prints to stdout only in verbose mode
func VerboseOutput(format string, args ...interface{}) {
	if verbose && !quiet {
		fmt.Printf(format, args...)
	}
}

// ErrorOutput prints to stderr
func ErrorOutput(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// ExitWithError prints an error and exits with the given code
func ExitWithError(code int, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(code)
}
