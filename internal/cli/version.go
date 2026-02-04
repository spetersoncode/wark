package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version of wark, build date, Go version, and database information.`,
	RunE:  runVersion,
}

type versionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	Database  string `json:"database,omitempty"`
	Schema    int64  `json:"schema_version,omitempty"`
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := versionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	// Try to get database info
	dbPath := GetDBPath()
	if db.Exists(dbPath) {
		info.Database = dbPath
		if dbPath == "" {
			info.Database = db.DefaultDBPath
		}

		// Try to get schema version
		database, err := db.Open(dbPath)
		if err == nil {
			defer database.Close()
			if version, err := database.MigrationStatus(); err == nil {
				info.Schema = version
			}
		}
	}

	if IsJSON() {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal version info: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Compact format matching --version: wark v0.1.0 (9f61316, 2026-02-02)
	fmt.Printf("wark %s (%s, %s)\n", info.Version, shortCommit(), shortDate())

	// Additional details in verbose mode or always for version subcommand
	fmt.Printf("Go: %s\n", info.GoVersion)
	fmt.Printf("Platform: %s\n", info.Platform)

	if info.Database != "" {
		fmt.Printf("Database: %s (schema v%d)\n", info.Database, info.Schema)
	} else {
		fmt.Println("Database: not initialized (run 'wark init')")
	}

	return nil
}
