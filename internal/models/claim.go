package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Claim represents an agent's claim on a ticket.
type Claim struct {
	ID         int64       `json:"id"`
	ClaimID    string      `json:"claim_id"`            // External identifier (e.g., "claim_abc123")
	TicketID   int64       `json:"ticket_id"`
	WorkerID   string      `json:"worker_id,omitempty"` // Deprecated: kept for backward compatibility
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
	if c.ClaimID == "" {
		return fmt.Errorf("claim_id cannot be empty")
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

// generateClaimID generates a unique claim identifier.
func generateClaimID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "claim_" + hex.EncodeToString(b)
}

// NewClaim creates a new claim for a ticket with the specified duration.
// Generates a unique claim ID internally.
func NewClaim(ticketID int64, duration time.Duration) *Claim {
	now := time.Now()
	return &Claim{
		ClaimID:   generateClaimID(),
		TicketID:  ticketID,
		ClaimedAt: now,
		ExpiresAt: now.Add(duration),
		Status:    ClaimStatusActive,
	}
}

// NewClaimWithWorker creates a new claim with an optional worker ID for backward compatibility.
// Deprecated: Use NewClaim instead.
func NewClaimWithWorker(ticketID int64, workerID string, duration time.Duration) *Claim {
	claim := NewClaim(ticketID, duration)
	claim.WorkerID = workerID
	return claim
}
