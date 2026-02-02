package models

import (
	"fmt"
	"time"
)

// DefaultClaimDuration is the default duration for a claim (1 hour).
const DefaultClaimDuration = time.Hour

// Claim represents an agent's claim on a ticket.
type Claim struct {
	ID         int64       `json:"id"`
	TicketID   int64       `json:"ticket_id"`
	WorkerID   string      `json:"worker_id"`
	ClaimedAt  time.Time   `json:"claimed_at"`
	ExpiresAt  time.Time   `json:"expires_at"`
	ReleasedAt *time.Time  `json:"released_at,omitempty"`
	Status     ClaimStatus `json:"status"`

	// Computed fields (populated by queries)
	TicketTitle      string `json:"ticket_title,omitempty"`
	TicketKey        string `json:"ticket_key,omitempty"`
	MinutesRemaining int    `json:"minutes_remaining,omitempty"`
}

// Validate validates the claim fields.
func (c *Claim) Validate() error {
	if c.TicketID <= 0 {
		return fmt.Errorf("ticket_id is required")
	}
	if c.WorkerID == "" {
		return fmt.Errorf("worker_id cannot be empty")
	}
	if c.ExpiresAt.IsZero() {
		return fmt.Errorf("expires_at is required")
	}
	if !c.Status.IsValid() {
		return fmt.Errorf("invalid status: %s", c.Status)
	}
	return nil
}

// IsActive returns true if the claim is currently active.
func (c *Claim) IsActive() bool {
	return c.Status == ClaimStatusActive && c.ExpiresAt.After(time.Now())
}

// IsExpired returns true if the claim has expired.
func (c *Claim) IsExpired() bool {
	return c.ExpiresAt.Before(time.Now()) && c.Status == ClaimStatusActive
}

// TimeRemaining returns the time remaining before the claim expires.
func (c *Claim) TimeRemaining() time.Duration {
	if c.Status != ClaimStatusActive {
		return 0
	}
	remaining := time.Until(c.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// NewClaim creates a new claim for a ticket with the default duration.
func NewClaim(ticketID int64, workerID string, duration time.Duration) *Claim {
	if duration <= 0 {
		duration = DefaultClaimDuration
	}
	now := time.Now()
	return &Claim{
		TicketID:  ticketID,
		WorkerID:  workerID,
		ClaimedAt: now,
		ExpiresAt: now.Add(duration),
		Status:    ClaimStatusActive,
	}
}
