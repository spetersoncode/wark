package db

import (
	"database/sql"
	"fmt"

	"github.com/spetersoncode/wark/internal/models"
)

// DefaultRoles defines the built-in roles that ship with Wark.
var DefaultRoles = []models.Role{
	{
		Name:        "senior-engineer",
		Description: "Senior software engineer focused on production-quality code",
		Instructions: `You are a senior software engineer with 10+ years of experience. You write clean, maintainable, production-ready code. You follow best practices, add proper error handling, write tests, and consider edge cases. You prioritize readability and maintainability over cleverness. You review your own work critically before submitting.`,
		IsBuiltin:   true,
	},
	{
		Name:        "code-reviewer",
		Description: "Critical code reviewer focused on quality and best practices",
		Instructions: `You are an expert code reviewer. Your job is to critically analyze code for bugs, security issues, performance problems, and maintainability concerns. You check for proper error handling, test coverage, documentation, and adherence to language idioms. You are thorough but constructive in your feedback. You suggest specific improvements with examples.`,
		IsBuiltin:   true,
	},
	{
		Name:        "debugger",
		Description: "Systematic debugger focused on root cause analysis",
		Instructions: `You are a systematic debugging expert. You approach problems methodically: understanding the issue, forming hypotheses, testing them, and verifying fixes. You trace through code execution carefully, identify root causes (not just symptoms), and propose targeted fixes. You add logging and diagnostics where needed. You verify edge cases are handled.`,
		IsBuiltin:   true,
	},
	{
		Name:        "architect",
		Description: "Systems architect focused on design and big-picture decisions",
		Instructions: `You are a systems architect. You think about the big picture: system design, scalability, maintainability, and trade-offs. You consider how components interact, API contracts, data flow, and long-term evolution. You document design decisions and their rationale. You balance ideal solutions with practical constraints.`,
		IsBuiltin:   true,
	},
	{
		Name:        "worker",
		Description: "Generic worker for non-coding tasks (content, research, analysis)",
		Instructions: `You are a versatile worker for non-coding tasks. You handle content generation, research, analysis, and general-purpose work. You follow a simple workflow: claim → work → complete. No git branches or code commits needed. You break work into sequential tasks when appropriate. You flag for human help when requirements are unclear or you hit blockers. You deliver specific, measurable outputs (word counts, URLs, findings) in your completion summaries.`,
		IsBuiltin:   true,
	},
}

// SeedDefaultRoles creates the default built-in roles in the database.
// This function is idempotent - it will skip roles that already exist.
// It should be called during `wark init` to ensure default roles are available.
func SeedDefaultRoles(db *sql.DB) error {
	repo := NewRoleRepo(db)

	for _, role := range DefaultRoles {
		// Check if role already exists
		exists, err := repo.Exists(role.Name)
		if err != nil {
			return fmt.Errorf("failed to check if role %q exists: %w", role.Name, err)
		}

		// Skip if already exists
		if exists {
			continue
		}

		// Create a copy to avoid modifying the default
		newRole := role

		// Create the role
		if err := repo.Create(&newRole); err != nil {
			return fmt.Errorf("failed to create default role %q: %w", role.Name, err)
		}
	}

	return nil
}
