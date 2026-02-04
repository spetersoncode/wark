package models

import (
	"fmt"
	"regexp"
	"time"
)

// Milestone represents a high-level goal within a project.
// Milestones provide generative context for AI planning and ticket organization.
type Milestone struct {
	ID         int64      `json:"id"`
	ProjectID  int64      `json:"project_id"`
	Key        string     `json:"key"`
	Name       string     `json:"name"`
	Goal       string     `json:"goal,omitempty"` // The generative context for AI
	TargetDate *time.Time `json:"target_date,omitempty"`
	Status     string     `json:"status"` // open, achieved, abandoned
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Joined fields
	ProjectKey string `json:"project_key,omitempty"`
}

// MilestoneWithStats extends Milestone with ticket statistics.
type MilestoneWithStats struct {
	Milestone
	TicketCount    int     `json:"ticket_count"`
	CompletedCount int     `json:"completed_count"`
	CompletionPct  float64 `json:"completion_pct"`
}

// Milestone statuses
const (
	MilestoneStatusOpen      = "open"
	MilestoneStatusAchieved  = "achieved"
	MilestoneStatusAbandoned = "abandoned"
)

// ValidMilestoneStatuses is the set of valid milestone statuses.
var ValidMilestoneStatuses = map[string]bool{
	MilestoneStatusOpen:      true,
	MilestoneStatusAchieved:  true,
	MilestoneStatusAbandoned: true,
}

// milestoneKeyRegex validates milestone keys (uppercase alphanumeric, 1-20 chars).
var milestoneKeyRegex = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,19}$`)

// ValidateMilestoneKey validates a milestone key.
func ValidateMilestoneKey(key string) error {
	if key == "" {
		return fmt.Errorf("milestone key cannot be empty")
	}
	if !milestoneKeyRegex.MatchString(key) {
		return fmt.Errorf("milestone key must be 1-20 uppercase alphanumeric characters (or underscores) starting with a letter")
	}
	return nil
}

// ValidateMilestoneStatus validates a milestone status.
func ValidateMilestoneStatus(status string) error {
	if status == "" {
		return fmt.Errorf("milestone status cannot be empty")
	}
	if !ValidMilestoneStatuses[status] {
		return fmt.Errorf("invalid milestone status: %s (must be open, achieved, or abandoned)", status)
	}
	return nil
}

// Validate validates the milestone fields.
func (m *Milestone) Validate() error {
	if m.ProjectID <= 0 {
		return fmt.Errorf("project_id is required")
	}
	if err := ValidateMilestoneKey(m.Key); err != nil {
		return err
	}
	if m.Name == "" {
		return fmt.Errorf("milestone name cannot be empty")
	}
	if err := ValidateMilestoneStatus(m.Status); err != nil {
		return err
	}
	return nil
}

// IsOpen returns true if the milestone is open.
func (m *Milestone) IsOpen() bool {
	return m.Status == MilestoneStatusOpen
}

// IsAchieved returns true if the milestone is achieved.
func (m *Milestone) IsAchieved() bool {
	return m.Status == MilestoneStatusAchieved
}

// IsAbandoned returns true if the milestone is abandoned.
func (m *Milestone) IsAbandoned() bool {
	return m.Status == MilestoneStatusAbandoned
}

// IsClosed returns true if the milestone is in a terminal state.
func (m *Milestone) IsClosed() bool {
	return m.IsAchieved() || m.IsAbandoned()
}

// FullKey returns the milestone key prefixed with project key.
func (m *Milestone) FullKey() string {
	if m.ProjectKey != "" {
		return fmt.Sprintf("%s/%s", m.ProjectKey, m.Key)
	}
	return m.Key
}
