package cli

import (
	"github.com/spf13/cobra"
)

var (
	dbPath  string
	jsonOut bool
	quiet   bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "wark",
	Short: "Local-first CLI task management for AI agent orchestration",
	Long: `Wark is a command-line task management tool inspired by Jira,
purpose-built for coordinating AI coding agents.

It provides project-based organization, dependency-aware ticket management,
claim-based work distribution, and human-in-the-loop support.`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to database file (default ~/.wark/wark.db)")
	rootCmd.PersistentFlags().BoolVarP(&jsonOut, "json", "j", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}
