package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spetersoncode/wark/internal/skill"
)

type skillInstallResult struct {
	Installed   bool     `json:"installed"`
	Path        string   `json:"path"`
	Files       []string `json:"files"`
	Overwritten bool     `json:"overwritten,omitempty"`
}

type skillInstallMultiResult struct {
	Targets []skillInstallResult `json:"targets"`
}

// detectSkillTargets returns all target directories for skill installation.
// It checks for Claude Code (~/.claude/) and OpenClaw (~/.openclaw/).
// Returns empty slice if neither exists.
func detectSkillTargets(homeDir string) []string {
	var targets []string

	claudeDir := filepath.Join(homeDir, ".claude")
	openclawDir := filepath.Join(homeDir, ".openclaw")

	// Check Claude Code
	if info, err := os.Stat(claudeDir); err == nil && info.IsDir() {
		targets = append(targets, filepath.Join(claudeDir, "skills", "wark"))
	}

	// Check OpenClaw
	if info, err := os.Stat(openclawDir); err == nil && info.IsDir() {
		targets = append(targets, filepath.Join(openclawDir, "skills", "wark"))
	}

	return targets
}

// installSkillToDir installs the skill files to a single directory.
// Returns the result and any error encountered.
func installSkillToDir(targetDir string, force bool) (*skillInstallResult, error) {
	// Check if target directory exists and has files
	existingFiles, err := listExistingFiles(targetDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check existing files: %w", err)
	}

	// If files exist and not forcing, skip
	overwritten := false
	if len(existingFiles) > 0 && !force {
		return nil, fmt.Errorf("skill files already exist at %s (use --force to overwrite)", targetDir)
	} else if len(existingFiles) > 0 {
		overwritten = true
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Get embedded skill files
	skillFS, err := skill.FS()
	if err != nil {
		return nil, fmt.Errorf("failed to access embedded skill files: %w", err)
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
		return nil, fmt.Errorf("failed to install skill files: %w", err)
	}

	return &skillInstallResult{
		Installed:   true,
		Path:        targetDir,
		Files:       installedFiles,
		Overwritten: overwritten,
	}, nil
}

// InstallSkill installs the skill to all detected AI agent directories.
// Returns the results for each target and any error if no targets found.
func InstallSkill(force bool) (*skillInstallMultiResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	targets := detectSkillTargets(homeDir)
	if len(targets) == 0 {
		return nil, nil // No targets found, not an error
	}

	result := &skillInstallMultiResult{}
	for _, targetDir := range targets {
		installResult, err := installSkillToDir(targetDir, force)
		if err != nil {
			return nil, err
		}
		result.Targets = append(result.Targets, *installResult)
	}

	return result, nil
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
