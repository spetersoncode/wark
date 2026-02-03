// Package skill provides embedded skill files for OpenClaw integration.
package skill

import (
	"embed"
	"io/fs"
)

//go:embed all:files
var skillFS embed.FS

// FS returns a filesystem rooted at the skill files directory.
// This allows direct access to SKILL.md, references/, etc.
func FS() (fs.FS, error) {
	return fs.Sub(skillFS, "files")
}
