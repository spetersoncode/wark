package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diogenes-ai-code/wark/internal/skill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillFSContainsExpectedFiles(t *testing.T) {
	fs, err := skill.FS()
	require.NoError(t, err)

	// Check SKILL.md exists
	f, err := fs.Open("SKILL.md")
	require.NoError(t, err)
	f.Close()

	// Check skill.yaml exists
	f, err = fs.Open("skill.yaml")
	require.NoError(t, err)
	f.Close()

	// Check references directory
	f, err = fs.Open("references/coder.md")
	require.NoError(t, err)
	f.Close()

	f, err = fs.Open("references/reviewer.md")
	require.NoError(t, err)
	f.Close()
}

func TestListExistingFiles(t *testing.T) {
	// Create temp directory with files
	tmpDir := t.TempDir()

	// Create some files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("test"), 0644))

	files, err := listExistingFiles(tmpDir)
	require.NoError(t, err)

	assert.Len(t, files, 2)
	assert.Contains(t, files, "file1.txt")
	assert.Contains(t, files, filepath.Join("subdir", "file2.txt"))
}

func TestListExistingFilesNonExistent(t *testing.T) {
	files, err := listExistingFiles("/nonexistent/path")
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
	assert.Empty(t, files)
}

func TestCopyEmbeddedFile(t *testing.T) {
	fs, err := skill.FS()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, "SKILL.md")

	err = copyEmbeddedFile(fs, "SKILL.md", dstPath)
	require.NoError(t, err)

	// Verify file was copied
	info, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
	assert.Greater(t, info.Size(), int64(0))
}

func TestSkillInstallResultStruct(t *testing.T) {
	result := skillInstallResult{
		Installed:   true,
		Path:        "/home/user/.openclaw/skills/wark",
		Files:       []string{"SKILL.md", "references/coder.md"},
		Overwritten: true,
	}

	assert.True(t, result.Installed)
	assert.Equal(t, "/home/user/.openclaw/skills/wark", result.Path)
	assert.Len(t, result.Files, 2)
	assert.True(t, result.Overwritten)
}
