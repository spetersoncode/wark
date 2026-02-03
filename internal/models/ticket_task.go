package models

import (
	"fmt"
	"time"
)

// TicketTask represents a sequential task within a ticket.
type TicketTask struct {
	ID          int64     `json:"id"`
	TicketID    int64     `json:"ticket_id"`
	Position    int       `json:"position"`
	Description string    `json:"description"`
	Complete    bool      `json:"complete"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Validate validates the ticket task fields.
func (t *TicketTask) Validate() error {
	if t.TicketID <= 0 {
		return fmt.Errorf("ticket_id is required")
	}
	if t.Position < 0 {
		return fmt.Errorf("position cannot be negative")
	}
	if t.Description == "" {
		return fmt.Errorf("description cannot be empty")
	}
	return nil
}

// NewTicketTask creates a new ticket task.
func NewTicketTask(ticketID int64, position int, description string) *TicketTask {
	now := time.Now()
	return &TicketTask{
		TicketID:    ticketID,
		Position:    position,
		Description: description,
		Complete:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
