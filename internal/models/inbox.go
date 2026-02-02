package models

import (
	"fmt"
	"time"
)

// InboxMessage represents a message in the human inbox.
type InboxMessage struct {
	ID          int64       `json:"id"`
	TicketID    int64       `json:"ticket_id"`
	MessageType MessageType `json:"message_type"`
	Content     string      `json:"content"`
	FromAgent   string      `json:"from_agent,omitempty"`
	Response    string      `json:"response,omitempty"`
	RespondedAt *time.Time  `json:"responded_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`

	// Computed fields (populated by queries)
	TicketTitle string `json:"ticket_title,omitempty"`
	TicketKey   string `json:"ticket_key,omitempty"`
}

// Validate validates the inbox message fields.
func (m *InboxMessage) Validate() error {
	if m.TicketID <= 0 {
		return fmt.Errorf("ticket_id is required")
	}
	if !m.MessageType.IsValid() {
		return fmt.Errorf("invalid message_type: %s", m.MessageType)
	}
	if m.Content == "" {
		return fmt.Errorf("content cannot be empty")
	}
	return nil
}

// IsPending returns true if the message has not been responded to.
func (m *InboxMessage) IsPending() bool {
	return m.RespondedAt == nil
}

// IsResponded returns true if the message has been responded to.
func (m *InboxMessage) IsResponded() bool {
	return m.RespondedAt != nil
}

// RequiresResponse returns true if this message type typically requires a response.
func (m *InboxMessage) RequiresResponse() bool {
	return m.MessageType.RequiresResponse()
}

// NewInboxMessage creates a new inbox message.
func NewInboxMessage(ticketID int64, msgType MessageType, content, fromAgent string) *InboxMessage {
	return &InboxMessage{
		TicketID:    ticketID,
		MessageType: msgType,
		Content:     content,
		FromAgent:   fromAgent,
		CreatedAt:   time.Now(),
	}
}
