package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/diogenes-ai-code/wark/internal/skill"
	"github.com/spf13/cobra"
)

// Skill command flags
var (
	skillForce bool
)

func init() {
	// skill install
	skillInstallCmd.Flags().BoolVarP(&skillForce, "force", "f", false, "Overwrite existing files without confirmation")

	// Add subcommands
	skillCmd.AddCommand(skillInstallCmd)

	rootCmd.AddCommand(skillCmd)
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Skill management commands",
	Long:  `Manage the Wark skill for OpenClaw integration.`,
}

// skill install
var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Wark skill to OpenClaw",
	Long: `Install the bundled Wark skill to ~/.openclaw/skills/wark/.

This copies the SKILL.md and reference files that help AI agents
understand how to use Wark effectively.

If files already exist, you will be prompted to confirm overwriting
unless --force is specified.

Examples:
  wark skill install          # Install with confirmation if files exist
  wark skill install --force  # Overwrite without confirmation`,
	Args: cobra.NoArgs,
	RunE: runSkillInstall,
}

type skillInstallResult struct {
	Installed   bool     `json:"installed"`
	Path        string   `json:"path"`
	Files       []string `json:"files"`
	Overwritten bool     `json:"overwritten,omitempty"`
}

func runSkillInstall(cmd *cobra.Command, args []string) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ErrGeneralWithCause(err, "failed to get home directory")
	}

	targetDir := filepath.Join(homeDir, ".openclaw", "skills", "wark")

	// Check if target directory exists and has files
	existingFiles, err := listExistingFiles(targetDir)
	if err != nil && !os.IsNotExist(err) {
		return ErrGeneralWithCause(err, "failed to check existing files")
	}

	// If files exist and not forcing, prompt for confirmation
	overwritten := false
	if len(existingFiles) > 0 && !skillForce {
		if IsJSON() {
			// In JSON mode, require --force for overwrites
			return ErrStateErrorWithSuggestion(
				"Use --force to overwrite existing files.",
				"skill files already exist at %s", targetDir,
			)
		}

		fmt.Printf("Skill files already exist at %s:\n", targetDir)
		for _, f := range existingFiles {
			fmt.Printf("  %s\n", f)
		}
		fmt.Print("\nThis will overwrite existing files. Continue? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			return ErrGeneral("installation cancelled")
		}
		overwritten = true
	} else if len(existingFiles) > 0 {
		overwritten = true
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return ErrGeneralWithCause(err, "failed to create directory %s", targetDir)
	}

	// Get embedded skill files
	skillFS, err := skill.FS()
	if err != nil {
		return ErrGeneralWithCause(err, "failed to access embedded skill files")
	}

	// Copy all files
	var installedFiles []string
	err = fs.WalkDir(skillFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, path)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Copy file
		if err := copyEmbeddedFile(skillFS, path, targetPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", path, err)
		}

		installedFiles = append(installedFiles, path)
		return nil
	})

	if err != nil {
		return ErrGeneralWithCause(err, "failed to install skill files")
	}

	result := skillInstallResult{
		Installed:   true,
		Path:        targetDir,
		Files:       installedFiles,
		Overwritten: overwritten,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Installed Wark skill to %s", targetDir)
	OutputLine("Files:")
	for _, f := range installedFiles {
		OutputLine("  %s", f)
	}

	return nil
}

// listExistingFiles returns a list of files in the given directory
func listExistingFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			relPath, _ := filepath.Rel(dir, path)
			files = append(files, relPath)
		}
		return nil
	})

	return files, err
}

// copyEmbeddedFile copies a file from the embedded FS to the target path
func copyEmbeddedFile(srcFS fs.FS, srcPath, dstPath string) error {
	src, err := srcFS.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
