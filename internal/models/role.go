package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Role represents an agent persona/capability that can be applied when working on tickets.
// Roles define different execution contexts with specific instructions (e.g., "senior-engineer", "code-reviewer").
type Role struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Instructions string    `json:"instructions"`
	IsBuiltin    bool      `json:"is_builtin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// roleNameRegex validates role names (lowercase alphanumeric with hyphens, 2-50 chars).
var roleNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]{1,49}$`)

// ValidateRoleName validates a role name.
// Role names must be:
// - 2-50 characters
// - Start with a lowercase letter
// - Contain only lowercase letters, numbers, and hyphens
// - No consecutive hyphens
func ValidateRoleName(name string) error {
	if name == "" {
		return fmt.Errorf("role name cannot be empty")
	}
	if !roleNameRegex.MatchString(name) {
		return fmt.Errorf("role name must be 2-50 lowercase alphanumeric characters with hyphens, starting with a letter")
	}
	if strings.Contains(name, "--") {
		return fmt.Errorf("role name cannot contain consecutive hyphens")
	}
	return nil
}

// Validate validates the role fields.
func (r *Role) Validate() error {
	if err := ValidateRoleName(r.Name); err != nil {
		return err
	}
	if r.Description == "" {
		return fmt.Errorf("role description cannot be empty")
	}
	if r.Instructions == "" {
		return fmt.Errorf("role instructions cannot be empty")
	}
	return nil
}

// IsUserDefined returns true if the role is user-defined (not built-in).
func (r *Role) IsUserDefined() bool {
	return !r.IsBuiltin
}
