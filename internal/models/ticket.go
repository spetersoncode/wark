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

	// Resolution (only set when Status == StatusClosed)
	Resolution *Resolution `json:"resolution,omitempty"`

	// Human input flag (reason when human)
	HumanFlagReason string `json:"human_flag_reason,omitempty"`

	// Classification
	Priority   Priority   `json:"priority"`
	Complexity Complexity `json:"complexity"`
	Type       TicketType `json:"type"`

	// Git integration
	Worktree string `json:"worktree,omitempty"`

	// Role (reference to a role for execution context)
	RoleID *int64 `json:"role_id,omitempty"`

	// Retry tracking
	RetryCount int `json:"retry_count"`
	MaxRetries int `json:"max_retries"`

	// Hierarchy (for decomposition)
	ParentTicketID *int64 `json:"parent_ticket_id,omitempty"`

	// Milestone association
	MilestoneID *int64 `json:"milestone_id,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Computed fields (not stored in DB, populated by queries)
	ProjectKey   string `json:"project_key,omitempty"`
	TicketKey    string `json:"ticket_key,omitempty"`
	MilestoneKey string `json:"milestone_key,omitempty"`
	RoleName     string `json:"role_name,omitempty"`
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
	if t.Type != "" && !t.Type.IsValid() {
		return fmt.Errorf("invalid ticket type: %s", t.Type)
	}
	if t.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	if t.RetryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}
	// Validate resolution
	if t.Status == StatusClosed {
		if t.Resolution == nil {
			return fmt.Errorf("resolution is required when status is closed")
		}
		if !t.Resolution.IsValid() {
			return fmt.Errorf("invalid resolution: %s", *t.Resolution)
		}
	} else if t.Resolution != nil {
		return fmt.Errorf("resolution should only be set when status is closed")
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

// IsClosed returns true if the ticket is closed.
func (t *Ticket) IsClosed() bool {
	return t.Status == StatusClosed
}

// IsClosedSuccessfully returns true if the ticket is closed with completed resolution.
func (t *Ticket) IsClosedSuccessfully() bool {
	return t.Status == StatusClosed && t.Resolution != nil && *t.Resolution == ResolutionCompleted
}

// HasExceededRetries returns true if the ticket has exceeded its retry limit.
func (t *Ticket) HasExceededRetries() bool {
	return t.RetryCount >= t.MaxRetries
}

// CanModifyDependencies returns true if dependencies can be modified for this ticket.
func (t *Ticket) CanModifyDependencies() bool {
	return t.Status.CanModifyDependencies()
}

// IsEpic returns true if the ticket is an epic.
func (t *Ticket) IsEpic() bool {
	return t.Type == TicketTypeEpic
}

// IsTask returns true if the ticket is a task (or has no type set, defaulting to task).
func (t *Ticket) IsTask() bool {
	return t.Type == TicketTypeTask || t.Type == ""
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
