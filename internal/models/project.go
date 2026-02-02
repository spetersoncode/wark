package models

import (
	"fmt"
	"regexp"
	"time"
)

// Project represents a top-level organizational container for tickets.
type Project struct {
	ID          int64     `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectStats holds statistics for a project.
type ProjectStats struct {
	TotalTickets    int `json:"total_tickets"`
	CreatedCount    int `json:"created_count"`
	ReadyCount      int `json:"ready_count"`
	InProgressCount int `json:"in_progress_count"`
	BlockedCount    int `json:"blocked_count"`
	NeedsHumanCount int `json:"needs_human_count"`
	ReviewCount     int `json:"review_count"`
	DoneCount       int `json:"done_count"`
	CancelledCount  int `json:"cancelled_count"`
}

// projectKeyRegex validates project keys (uppercase alphanumeric, 2-10 chars).
var projectKeyRegex = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)

// ValidateProjectKey validates a project key.
func ValidateProjectKey(key string) error {
	if key == "" {
		return fmt.Errorf("project key cannot be empty")
	}
	if !projectKeyRegex.MatchString(key) {
		return fmt.Errorf("project key must be 2-10 uppercase alphanumeric characters starting with a letter")
	}
	return nil
}

// Validate validates the project fields.
func (p *Project) Validate() error {
	if err := ValidateProjectKey(p.Key); err != nil {
		return err
	}
	if p.Name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	return nil
}
