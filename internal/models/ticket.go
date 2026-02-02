package models

import (
	"fmt"
	"time"
)

// Ticket represents a unit of work in wark.
type Ticket struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`

	// Status (state machine)
	Status Status `json:"status"`

	// Human input flag (reason when needs_human)
	HumanFlagReason string `json:"human_flag_reason,omitempty"`

	// Classification
	Priority   Priority   `json:"priority"`
	Complexity Complexity `json:"complexity"`

	// Git integration
	BranchName string `json:"branch_name,omitempty"`

	// Retry tracking
	RetryCount int `json:"retry_count"`
	MaxRetries int `json:"max_retries"`

	// Hierarchy (for decomposition)
	ParentTicketID *int64 `json:"parent_ticket_id,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Computed fields (not stored in DB, populated by queries)
	ProjectKey string `json:"project_key,omitempty"`
	TicketKey  string `json:"ticket_key,omitempty"`
}

// Key returns the ticket key in the format PROJECT-NUMBER.
func (t *Ticket) Key() string {
	if t.TicketKey != "" {
		return t.TicketKey
	}
	if t.ProjectKey != "" {
		return fmt.Sprintf("%s-%d", t.ProjectKey, t.Number)
	}
	return fmt.Sprintf("?-%d", t.Number)
}

// Validate validates the ticket fields.
func (t *Ticket) Validate() error {
	if t.ProjectID <= 0 {
		return fmt.Errorf("project_id is required")
	}
	if t.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if !t.Status.IsValid() {
		return fmt.Errorf("invalid status: %s", t.Status)
	}
	if !t.Priority.IsValid() {
		return fmt.Errorf("invalid priority: %s", t.Priority)
	}
	if !t.Complexity.IsValid() {
		return fmt.Errorf("invalid complexity: %s", t.Complexity)
	}
	if t.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	if t.RetryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}
	return nil
}

// IsWorkable returns true if the ticket can be claimed for work.
func (t *Ticket) IsWorkable() bool {
	return t.Status.IsWorkable()
}

// IsTerminal returns true if the ticket is in a terminal state.
func (t *Ticket) IsTerminal() bool {
	return t.Status.IsTerminal()
}

// HasExceededRetries returns true if the ticket has exceeded its retry limit.
func (t *Ticket) HasExceededRetries() bool {
	return t.RetryCount >= t.MaxRetries
}

// TicketDependency represents a dependency between two tickets.
type TicketDependency struct {
	TicketID    int64     `json:"ticket_id"`
	DependsOnID int64     `json:"depends_on_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// Validate validates the dependency.
func (td *TicketDependency) Validate() error {
	if td.TicketID <= 0 {
		return fmt.Errorf("ticket_id is required")
	}
	if td.DependsOnID <= 0 {
		return fmt.Errorf("depends_on_id is required")
	}
	if td.TicketID == td.DependsOnID {
		return fmt.Errorf("ticket cannot depend on itself")
	}
	return nil
}
