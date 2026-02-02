// Package tasks provides background task runners for wark.
package tasks

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
)

// ExpirationResult represents the result of processing a single expired claim.
type ExpirationResult struct {
	TicketID     int64  `json:"ticket_id"`
	TicketKey    string `json:"ticket_key"`
	WorkerID     string `json:"worker_id"`
	NewStatus    string `json:"new_status"`
	RetryCount   int    `json:"retry_count"`
	MaxRetries   int    `json:"max_retries"`
	Escalated    bool   `json:"escalated"`
	ErrorMessage string `json:"error,omitempty"`
}

// ExpireClaimsResult represents the result of running the expiration task.
type ExpireClaimsResult struct {
	Processed   int                 `json:"processed"`
	Expired     int                 `json:"expired"`
	Escalated   int                 `json:"escalated"`
	Errors      int                 `json:"errors"`
	Results     []*ExpirationResult `json:"results,omitempty"`
	DryRun      bool                `json:"dry_run"`
}

// ClaimExpirer handles the expiration of claims.
type ClaimExpirer struct {
	db           *sql.DB
	claimRepo    *db.ClaimRepo
	ticketRepo   *db.TicketRepo
	activityRepo *db.ActivityRepo
}

// NewClaimExpirer creates a new ClaimExpirer.
func NewClaimExpirer(database *sql.DB) *ClaimExpirer {
	return &ClaimExpirer{
		db:           database,
		claimRepo:    db.NewClaimRepo(database),
		ticketRepo:   db.NewTicketRepo(database),
		activityRepo: db.NewActivityRepo(database),
	}
}

// ExpireAll finds and processes all expired claims.
// If dryRun is true, it returns what would be expired without making changes.
func (e *ClaimExpirer) ExpireAll(dryRun bool) (*ExpireClaimsResult, error) {
	result := &ExpireClaimsResult{
		DryRun: dryRun,
	}

	// Find all expired claims
	expiredClaims, err := e.claimRepo.ListExpired()
	if err != nil {
		return nil, fmt.Errorf("failed to list expired claims: %w", err)
	}

	result.Processed = len(expiredClaims)

	for _, claim := range expiredClaims {
		expResult := e.processExpiredClaim(claim, dryRun)
		result.Results = append(result.Results, expResult)

		if expResult.ErrorMessage != "" {
			result.Errors++
		} else if expResult.Escalated {
			result.Escalated++
			result.Expired++
		} else {
			result.Expired++
		}
	}

	return result, nil
}

// ExpireTicket expires the claim for a specific ticket.
func (e *ClaimExpirer) ExpireTicket(ticketID int64, dryRun bool) (*ExpirationResult, error) {
	claim, err := e.claimRepo.GetActiveByTicketID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claim: %w", err)
	}
	if claim == nil {
		return nil, fmt.Errorf("no active claim found for ticket")
	}

	return e.processExpiredClaim(claim, dryRun), nil
}

// processExpiredClaim processes a single expired claim.
func (e *ClaimExpirer) processExpiredClaim(claim *models.Claim, dryRun bool) *ExpirationResult {
	result := &ExpirationResult{
		TicketID:  claim.TicketID,
		TicketKey: claim.TicketKey,
		WorkerID:  claim.WorkerID,
	}

	// Get the ticket
	ticket, err := e.ticketRepo.GetByID(claim.TicketID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to get ticket: %v", err)
		return result
	}
	if ticket == nil {
		result.ErrorMessage = "ticket not found"
		return result
	}

	// Only process if ticket is still in progress
	if ticket.Status != models.StatusInProgress {
		result.ErrorMessage = fmt.Sprintf("ticket not in progress (status: %s)", ticket.Status)
		return result
	}

	// Calculate new retry count
	newRetryCount := ticket.RetryCount + 1
	result.RetryCount = newRetryCount
	result.MaxRetries = ticket.MaxRetries

	// Determine new status: needs_human if exceeded max retries, otherwise ready
	newStatus := models.StatusReady
	if newRetryCount >= ticket.MaxRetries {
		newStatus = models.StatusNeedsHuman
		result.Escalated = true
	}
	result.NewStatus = string(newStatus)

	if dryRun {
		return result
	}

	// Update claim status
	if err := e.claimRepo.Release(claim.ID, models.ClaimStatusExpired); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to expire claim: %v", err)
		return result
	}

	// Update ticket
	ticket.Status = newStatus
	ticket.RetryCount = newRetryCount
	if newStatus == models.StatusNeedsHuman {
		ticket.HumanFlagReason = "max_retries_exceeded"
	}
	if err := e.ticketRepo.Update(ticket); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to update ticket: %v", err)
		return result
	}

	// Log activity
	action := models.ActionExpired
	summary := "Claim expired"
	details := map[string]interface{}{
		"worker_id":   claim.WorkerID,
		"retry_count": newRetryCount,
		"max_retries": ticket.MaxRetries,
	}

	if result.Escalated {
		summary = fmt.Sprintf("Claim expired - escalated to human (retry %d/%d)", newRetryCount, ticket.MaxRetries)
		details["escalated"] = true
		details["reason"] = "max_retries_exceeded"
	}

	e.activityRepo.LogActionWithDetails(ticket.ID, action, models.ActorTypeSystem, "",
		summary, details)

	return result
}

// RunDaemon runs the claim expiration check in a loop.
// It checks for expired claims every interval and processes them.
func (e *ClaimExpirer) RunDaemon(ctx context.Context, interval time.Duration, callback func(*ExpireClaimsResult)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	if result, err := e.ExpireAll(false); err == nil && callback != nil {
		callback(result)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			result, err := e.ExpireAll(false)
			if err != nil {
				// Log error but continue running
				continue
			}
			if callback != nil {
				callback(result)
			}
		}
	}
}
