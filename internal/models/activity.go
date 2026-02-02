package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// ActivityLog represents an entry in the activity log for a ticket.
type ActivityLog struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	Action    Action    `json:"action"`
	ActorType ActorType `json:"actor_type"`
	ActorID   string    `json:"actor_id,omitempty"`
	Details   string    `json:"details,omitempty"` // JSON string
	Summary   string    `json:"summary,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	// Computed fields
	TicketKey string `json:"ticket_key,omitempty"`
}

// Validate validates the activity log entry.
func (a *ActivityLog) Validate() error {
	if a.TicketID <= 0 {
		return fmt.Errorf("ticket_id is required")
	}
	if !a.Action.IsValid() {
		return fmt.Errorf("invalid action: %s", a.Action)
	}
	if !a.ActorType.IsValid() {
		return fmt.Errorf("invalid actor_type: %s", a.ActorType)
	}
	return nil
}

// GetDetails parses the JSON details into a map.
func (a *ActivityLog) GetDetails() (map[string]interface{}, error) {
	if a.Details == "" {
		return nil, nil
	}
	var details map[string]interface{}
	if err := json.Unmarshal([]byte(a.Details), &details); err != nil {
		return nil, fmt.Errorf("failed to parse details: %w", err)
	}
	return details, nil
}

// SetDetails sets the details from a map.
func (a *ActivityLog) SetDetails(details map[string]interface{}) error {
	if details == nil {
		a.Details = ""
		return nil
	}
	data, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}
	a.Details = string(data)
	return nil
}

// FieldChangeDetails represents the details for a field_changed action.
type FieldChangeDetails struct {
	Field    string `json:"field"`
	OldValue string `json:"old"`
	NewValue string `json:"new"`
}

// GetFieldChangeDetails returns the field change details if this is a field_changed action.
func (a *ActivityLog) GetFieldChangeDetails() (*FieldChangeDetails, error) {
	if a.Action != ActionFieldChanged {
		return nil, fmt.Errorf("not a field_changed action")
	}
	if a.Details == "" {
		return nil, nil
	}
	var details FieldChangeDetails
	if err := json.Unmarshal([]byte(a.Details), &details); err != nil {
		return nil, fmt.Errorf("failed to parse field change details: %w", err)
	}
	return &details, nil
}

// NewActivityLog creates a new activity log entry.
func NewActivityLog(ticketID int64, action Action, actorType ActorType, actorID, summary string) *ActivityLog {
	return &ActivityLog{
		TicketID:  ticketID,
		Action:    action,
		ActorType: actorType,
		ActorID:   actorID,
		Summary:   summary,
		CreatedAt: time.Now(),
	}
}

// NewActivityLogWithDetails creates a new activity log entry with details.
func NewActivityLogWithDetails(ticketID int64, action Action, actorType ActorType, actorID, summary string, details map[string]interface{}) (*ActivityLog, error) {
	log := NewActivityLog(ticketID, action, actorType, actorID, summary)
	if err := log.SetDetails(details); err != nil {
		return nil, err
	}
	return log, nil
}
