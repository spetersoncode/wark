package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information (set at build time)
var (
	Version   = "0.1.0"
	BuildDate = "unknown"
	GoVersion = "unknown"
)

// Global flags
var (
	dbPath  string
	jsonOut bool
	quiet   bool
	verbose bool
)

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

var rootCmd = &cobra.Command{
	Use:   "wark",
	Short: "Local-first CLI task management for AI agent orchestration",
	Long: `Wark is a command-line task management tool inspired by Jira,
purpose-built for coordinating AI coding agents.

It provides project-based organization, dependency-aware ticket management,
claim-based work distribution, and human-in-the-loop support.

Use "wark init" to initialize a new wark database.
Use "wark --help" to see all available commands.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to database file (default ~/.wark/wark.db)")
	rootCmd.PersistentFlags().BoolVarP(&jsonOut, "json", "j", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add commands
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// GetDBPath returns the database path from flags or default
func GetDBPath() string {
	if dbPath != "" {
		return dbPath
	}
	// Check environment variable
	if envDB := os.Getenv("WARK_DB"); envDB != "" {
		return envDB
	}
	return "" // Will use default in db.Open
}

// IsJSON returns whether JSON output is requested
func IsJSON() bool {
	return jsonOut
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
