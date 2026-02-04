// Package service provides business logic services for wark.
package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/diogenes-ai-code/wark/internal/state"
	"github.com/diogenes-ai-code/wark/internal/tasks"
)

// TicketService provides business logic for ticket workflow operations.
// It coordinates between repositories, state machine, and dependency resolver
// to implement claim, release, complete, accept, reject, flag, close, reopen, and promote operations.
type TicketService struct {
	db           *sql.DB
	ticketRepo   *db.TicketRepo
	claimRepo    *db.ClaimRepo
	depRepo      *db.DependencyRepo
	tasksRepo    *db.TasksRepo
	activityRepo *db.ActivityRepo
	inboxRepo    *db.InboxRepo
	depResolver  *tasks.DependencyResolver
	stateMachine *state.Machine
}

// NewTicketService creates a new TicketService with all required dependencies.
func NewTicketService(database *sql.DB) *TicketService {
	return &TicketService{
		db:           database,
		ticketRepo:   db.NewTicketRepo(database),
		claimRepo:    db.NewClaimRepo(database),
		depRepo:      db.NewDependencyRepo(database),
		tasksRepo:    db.NewTasksRepo(database),
		activityRepo: db.NewActivityRepo(database),
		inboxRepo:    db.NewInboxRepo(database),
		depResolver:  tasks.NewDependencyResolver(database),
		stateMachine: state.NewMachine(),
	}
}

// ClaimResult contains the result of claiming a ticket.
type ClaimResult struct {
	Ticket     *models.Ticket     `json:"ticket"`
	Claim      *models.Claim      `json:"claim"`
	Branch     string             `json:"branch"`
	NextTask   *models.TicketTask `json:"next_task,omitempty"`
	TasksTotal int                `json:"tasks_total,omitempty"`
}

// CompleteResult contains the result of completing a ticket.
type CompleteResult struct {
	Ticket           *models.Ticket            `json:"ticket"`
	AutoAccepted     bool                      `json:"auto_accepted"`
	DepsResolved     int                       `json:"deps_resolved"`
	ResolutionResult *tasks.ResolutionResult   `json:"resolution_result,omitempty"`
}

// AcceptResult contains the result of accepting a ticket.
type AcceptResult struct {
	Ticket           *models.Ticket            `json:"ticket"`
	DepsResolved     int                       `json:"deps_resolved"`
	ResolutionResult *tasks.ResolutionResult   `json:"resolution_result,omitempty"`
}

// TicketError represents a domain-specific error from the ticket service.
type TicketError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *TicketError) Error() string {
	return e.Message
}

// Error codes for ticket operations
const (
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeInvalidState       = "INVALID_STATE"
	ErrCodeAlreadyClaimed     = "ALREADY_CLAIMED"
	ErrCodeUnresolvedDeps     = "UNRESOLVED_DEPS"
	ErrCodeIncompleteTasks    = "INCOMPLETE_TASKS"
	ErrCodeInvalidReason      = "INVALID_REASON"
	ErrCodeInvalidResolution  = "INVALID_RESOLUTION"
	ErrCodeDatabase           = "DATABASE_ERROR"
)

func newTicketError(code, message string, details map[string]interface{}) *TicketError {
	return &TicketError{Code: code, Message: message, Details: details}
}

// Claim acquires a time-limited claim on a ticket for the specified worker.
// The ticket must be in ready or review status. Review claims don't change ticket status.
// Returns ClaimResult with ticket, claim, branch name, and task info.
func (s *TicketService) Claim(ticketID int64, workerID string, duration time.Duration) (*ClaimResult, error) {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return nil, newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Check if ticket can be claimed (ready or review status)
	isReviewClaim := ticket.Status == models.StatusReview
	if !isReviewClaim {
		// For ready tickets, validate the state transition
		if err := s.stateMachine.CanTransition(ticket, models.StatusInProgress, state.TransitionTypeManual, "", nil); err != nil {
			return nil, newTicketError(ErrCodeInvalidState, fmt.Sprintf("cannot claim ticket: %v", err), 
				map[string]interface{}{"current_status": ticket.Status})
		}
	} else if ticket.Status != models.StatusReview {
		// Not ready and not review - can't claim
		return nil, newTicketError(ErrCodeInvalidState, 
			fmt.Sprintf("cannot claim ticket: must be in ready or review status (current: %s)", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	// Check for existing active claim
	existingClaim, err := s.claimRepo.GetActiveByTicketID(ticket.ID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to check existing claims: %v", err), nil)
	}
	if existingClaim != nil {
		return nil, newTicketError(ErrCodeAlreadyClaimed,
			fmt.Sprintf("ticket is already claimed by %s (expires: %s)", existingClaim.WorkerID, existingClaim.ExpiresAt.Format("15:04:05")),
			map[string]interface{}{
				"worker_id":  existingClaim.WorkerID,
				"expires_at": existingClaim.ExpiresAt,
			})
	}

	// Check for unresolved dependencies (only for ready tickets, not review)
	if !isReviewClaim {
		hasUnresolved, err := s.depRepo.HasUnresolvedDependencies(ticket.ID)
		if err != nil {
			return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to check dependencies: %v", err), nil)
		}
		if hasUnresolved {
			return nil, newTicketError(ErrCodeUnresolvedDeps, "ticket has unresolved dependencies", nil)
		}
	}

	// Create claim
	claim := models.NewClaim(ticket.ID, workerID, duration)
	if err := s.claimRepo.Create(claim); err != nil {
		// Handle race condition: another agent claimed between check and insert
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, newTicketError(ErrCodeAlreadyClaimed, "ticket already claimed", nil)
		}
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to create claim: %v", err), nil)
	}

	// Update ticket status (only for ready tickets, review stays at review)
	if !isReviewClaim {
		ticket.Status = models.StatusInProgress
		if err := s.ticketRepo.Update(ticket); err != nil {
			// Rollback claim
			s.claimRepo.Release(claim.ID, models.ClaimStatusReleased)
			return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket status: %v", err), nil)
		}
	}

	// Log activity
	claimType := "Claimed"
	if isReviewClaim {
		claimType = "Claimed for review"
	}
	durationMins := int(duration.Minutes())
	s.activityRepo.LogActionWithDetails(ticket.ID, models.ActionClaimed, models.ActorTypeAgent, workerID,
		fmt.Sprintf("%s (expires in %dm)", claimType, durationMins),
		map[string]interface{}{
			"worker_id":     workerID,
			"duration_mins": durationMins,
			"expires_at":    claim.ExpiresAt.Format(time.RFC3339),
			"review_claim":  isReviewClaim,
		})

	// Generate branch name if needed
	branchName := ticket.BranchName
	if branchName == "" {
		branchName = GenerateBranchName(ticket.ProjectKey, ticket.Number, ticket.Title)
	}

	result := &ClaimResult{
		Ticket: ticket,
		Claim:  claim,
		Branch: branchName,
	}

	// Get task info if ticket has tasks
	ctx := context.Background()
	taskCounts, err := s.tasksRepo.GetTaskCounts(ctx, ticket.ID)
	if err == nil && taskCounts.Total > 0 {
		result.TasksTotal = taskCounts.Total
		nextTask, err := s.tasksRepo.GetNextIncompleteTask(ctx, ticket.ID)
		if err == nil && nextTask != nil {
			result.NextTask = nextTask
		}
	}

	return result, nil
}

// Release releases a claimed ticket back to the ready queue.
// The ticket must be in in_progress status with an active claim.
func (s *TicketService) Release(ticketID int64, reason string) error {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Check if ticket is in progress
	if ticket.Status != models.StatusInProgress {
		return newTicketError(ErrCodeInvalidState,
			fmt.Sprintf("ticket is not in progress (current status: %s)", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	// Get active claim
	claim, err := s.claimRepo.GetActiveByTicketID(ticket.ID)
	if err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get claim: %v", err), nil)
	}
	if claim == nil {
		return newTicketError(ErrCodeInvalidState, "no active claim found for ticket", nil)
	}

	// Release claim
	if err := s.claimRepo.Release(claim.ID, models.ClaimStatusReleased); err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to release claim: %v", err), nil)
	}

	// Update ticket status
	ticket.Status = models.StatusReady
	ticket.RetryCount++
	if err := s.ticketRepo.Update(ticket); err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket status: %v", err), nil)
	}

	// Log activity
	summary := "Released"
	if reason != "" {
		summary = fmt.Sprintf("Released: %s", reason)
	}
	s.activityRepo.LogActionWithDetails(ticket.ID, models.ActionReleased, models.ActorTypeAgent, claim.WorkerID,
		summary,
		map[string]interface{}{
			"reason":          reason,
			"retry_count":     ticket.RetryCount,
			"previous_status": string(models.StatusInProgress),
			"new_status":      string(ticket.Status),
		})

	return nil
}

// Complete marks a ticket as complete and moves it to review status.
// If autoAccept is true, the ticket is immediately closed with completed resolution.
// All tasks must be complete before the ticket can be completed.
func (s *TicketService) Complete(ticketID int64, summary string, autoAccept bool) (*CompleteResult, error) {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return nil, newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Check if ticket is in progress
	if ticket.Status != models.StatusInProgress {
		return nil, newTicketError(ErrCodeInvalidState,
			fmt.Sprintf("ticket is not in progress (current status: %s)", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	// Get active claim for logging
	claim, _ := s.claimRepo.GetActiveByTicketID(ticket.ID)
	workerID := ""
	if claim != nil {
		workerID = claim.WorkerID
	}

	// Check if ticket has incomplete tasks - block completion if so
	ctx := context.Background()
	taskCounts, err := s.tasksRepo.GetTaskCounts(ctx, ticket.ID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get task counts: %v", err), nil)
	}

	if taskCounts.Total > 0 {
		incompleteTasks, err := s.tasksRepo.ListIncompleteTasks(ctx, ticket.ID)
		if err != nil {
			return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to check incomplete tasks: %v", err), nil)
		}

		if len(incompleteTasks) > 0 {
			taskDescs := make([]string, len(incompleteTasks))
			for i, task := range incompleteTasks {
				taskDescs[i] = task.Description
			}
			return nil, newTicketError(ErrCodeIncompleteTasks,
				fmt.Sprintf("cannot complete ticket: %d task(s) incomplete", len(incompleteTasks)),
				map[string]interface{}{
					"incomplete_count": len(incompleteTasks),
					"incomplete_tasks": taskDescs,
				})
		}
	}

	// Complete the claim
	if claim != nil {
		s.claimRepo.Release(claim.ID, models.ClaimStatusCompleted)
	}

	// Determine final status
	finalStatus := models.StatusReview
	var resolution *models.Resolution
	if autoAccept {
		finalStatus = models.StatusClosed
		res := models.ResolutionCompleted
		resolution = &res
	}

	// Update ticket
	ticket.Status = finalStatus
	ticket.Resolution = resolution
	if finalStatus == models.StatusClosed {
		now := time.Now()
		ticket.CompletedAt = &now
	}
	if err := s.ticketRepo.Update(ticket); err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket: %v", err), nil)
	}

	// Log activity
	activitySummary := "Work completed"
	if summary != "" {
		activitySummary = summary
	}
	if taskCounts.Total > 0 {
		activitySummary = fmt.Sprintf("All %d tasks completed", taskCounts.Total)
		if summary != "" {
			activitySummary = fmt.Sprintf("%s - %s", activitySummary, summary)
		}
	}
	s.activityRepo.LogActionWithDetails(ticket.ID, models.ActionCompleted, models.ActorTypeAgent, workerID,
		activitySummary,
		map[string]interface{}{
			"summary":     summary,
			"auto_accept": autoAccept,
			"tasks_total": taskCounts.Total,
		})

	result := &CompleteResult{
		Ticket:       ticket,
		AutoAccepted: autoAccept,
	}

	if autoAccept {
		s.activityRepo.LogAction(ticket.ID, models.ActionAccepted, models.ActorTypeSystem, "", "Auto-accepted")

		// Run dependency resolution when ticket is done
		resResult, err := s.depResolver.OnTicketCompleted(ticket.ID, true)
		if err == nil && resResult != nil {
			result.DepsResolved = resResult.Unblocked + resResult.ParentsUpdated
			result.ResolutionResult = resResult
		}
	}

	return result, nil
}

// Accept accepts completed work and closes the ticket with completed resolution.
// The ticket must be in review status and have no incomplete tasks.
func (s *TicketService) Accept(ticketID int64) (*AcceptResult, error) {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return nil, newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Check if ticket is in review
	if ticket.Status != models.StatusReview {
		return nil, newTicketError(ErrCodeInvalidState,
			fmt.Sprintf("ticket is not in review (current status: %s)", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	// Check for incomplete tasks - block acceptance if any exist
	ctx := context.Background()
	incompleteTasks, err := s.tasksRepo.ListIncompleteTasks(ctx, ticket.ID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to check incomplete tasks: %v", err), nil)
	}
	if len(incompleteTasks) > 0 {
		taskDescs := make([]string, len(incompleteTasks))
		for i, task := range incompleteTasks {
			taskDescs[i] = task.Description
		}
		return nil, newTicketError(ErrCodeIncompleteTasks,
			fmt.Sprintf("cannot accept ticket: %d task(s) incomplete", len(incompleteTasks)),
			map[string]interface{}{
				"incomplete_count": len(incompleteTasks),
				"incomplete_tasks": taskDescs,
			})
	}

	// Validate transition
	resolution := models.ResolutionCompleted
	if err := s.stateMachine.CanTransition(ticket, models.StatusClosed, state.TransitionTypeManual, "", &resolution); err != nil {
		return nil, newTicketError(ErrCodeInvalidState, fmt.Sprintf("cannot accept ticket: %v", err), nil)
	}

	// Update ticket
	ticket.Status = models.StatusClosed
	ticket.Resolution = &resolution
	now := time.Now()
	ticket.CompletedAt = &now
	if err := s.ticketRepo.Update(ticket); err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket: %v", err), nil)
	}

	// Log activity
	s.activityRepo.LogAction(ticket.ID, models.ActionAccepted, models.ActorTypeHuman, "", "Work accepted")

	result := &AcceptResult{
		Ticket: ticket,
	}

	// Run dependency resolution: unblock dependents and update parent
	resResult, err := s.depResolver.OnTicketCompleted(ticket.ID, false) // false = parents go to review, not auto-done
	if err == nil && resResult != nil {
		result.DepsResolved = resResult.Unblocked + resResult.ParentsUpdated
		result.ResolutionResult = resResult
	}

	return result, nil
}

// Reject rejects completed work and returns the ticket to ready status.
// The ticket must be in review status. Reason is required.
func (s *TicketService) Reject(ticketID int64, reason string) error {
	if reason == "" {
		return newTicketError(ErrCodeInvalidReason, "reason is required for rejection", nil)
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Check if ticket is in review
	if ticket.Status != models.StatusReview {
		return newTicketError(ErrCodeInvalidState,
			fmt.Sprintf("ticket is not in review (current status: %s)", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	// Validate transition (reject goes back to ready for fresh pickup)
	if err := s.stateMachine.CanTransition(ticket, models.StatusReady, state.TransitionTypeManual, reason, nil); err != nil {
		return newTicketError(ErrCodeInvalidState, fmt.Sprintf("cannot reject ticket: %v", err), nil)
	}

	// Release any active claim so ticket can be picked up fresh
	claim, _ := s.claimRepo.GetActiveByTicketID(ticket.ID)
	if claim != nil {
		s.claimRepo.Release(claim.ID, models.ClaimStatusReleased)
	}

	// Update ticket
	ticket.Status = models.StatusReady
	ticket.RetryCount++
	if err := s.ticketRepo.Update(ticket); err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket: %v", err), nil)
	}

	// Log activity
	s.activityRepo.LogActionWithDetails(ticket.ID, models.ActionRejected, models.ActorTypeHuman, "",
		fmt.Sprintf("Rejected: %s", reason),
		map[string]interface{}{
			"reason":      reason,
			"retry_count": ticket.RetryCount,
		})

	return nil
}

// Flag flags a ticket for human attention and moves it to human status.
// The ticket must be in ready or in_progress status.
func (s *TicketService) Flag(ticketID int64, reason models.FlagReason, message string, workerID string) error {
	if message == "" {
		return newTicketError(ErrCodeInvalidReason, "message is required", nil)
	}
	if !reason.IsValid() {
		return newTicketError(ErrCodeInvalidReason, fmt.Sprintf("invalid flag reason: %s", reason), nil)
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Check if ticket can be escalated to human
	if !state.CanBeEscalated(ticket.Status) {
		return newTicketError(ErrCodeInvalidState,
			fmt.Sprintf("ticket cannot be flagged in status: %s", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	previousStatus := ticket.Status

	// Get worker ID if claimed
	claim, _ := s.claimRepo.GetActiveByTicketID(ticket.ID)
	if claim != nil {
		if workerID == "" {
			workerID = claim.WorkerID
		}
		// Release the claim
		s.claimRepo.Release(claim.ID, models.ClaimStatusReleased)
	}

	// Update ticket status
	ticket.Status = models.StatusHuman
	ticket.HumanFlagReason = string(reason)
	if err := s.ticketRepo.Update(ticket); err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket: %v", err), nil)
	}

	// Create inbox message
	msgType := models.MessageTypeQuestion
	if reason == models.FlagReasonDecisionNeeded {
		msgType = models.MessageTypeDecision
	} else if reason == models.FlagReasonRiskAssessment || reason == models.FlagReasonIrreconcilableConflict {
		msgType = models.MessageTypeEscalation
	}

	inboxMsg := models.NewInboxMessage(ticket.ID, msgType, message, workerID)
	if err := s.inboxRepo.Create(inboxMsg); err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to create inbox message: %v", err), nil)
	}

	// Log activity
	s.activityRepo.LogActionWithDetails(ticket.ID, models.ActionEscalated, models.ActorTypeAgent, workerID,
		fmt.Sprintf("Flagged: %s", reason),
		map[string]interface{}{
			"reason":           string(reason),
			"message":          message,
			"inbox_message_id": inboxMsg.ID,
			"previous_status":  string(previousStatus),
		})

	return nil
}

// Close closes a ticket with the specified resolution.
// Any active claim is released. The ticket must not already be closed.
func (s *TicketService) Close(ticketID int64, resolution models.Resolution, reason string) error {
	if !resolution.IsValid() {
		return newTicketError(ErrCodeInvalidResolution,
			fmt.Sprintf("invalid resolution: %s (must be completed, wont_do, duplicate, invalid, or obsolete)", resolution),
			nil)
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Check if ticket can be closed
	if !state.CanBeClosed(ticket.Status) {
		return newTicketError(ErrCodeInvalidState,
			fmt.Sprintf("ticket cannot be closed in status: %s", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	// Release any active claim
	claim, _ := s.claimRepo.GetActiveByTicketID(ticket.ID)
	if claim != nil {
		s.claimRepo.Release(claim.ID, models.ClaimStatusReleased)
	}

	// Update ticket
	ticket.Status = models.StatusClosed
	ticket.Resolution = &resolution
	now := time.Now()
	ticket.CompletedAt = &now
	if err := s.ticketRepo.Update(ticket); err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket: %v", err), nil)
	}

	// Log activity
	summary := fmt.Sprintf("Ticket closed: %s", resolution)
	if reason != "" {
		summary = fmt.Sprintf("Closed (%s): %s", resolution, reason)
	}
	s.activityRepo.LogActionWithDetails(ticket.ID, models.ActionClosed, models.ActorTypeHuman, "",
		summary,
		map[string]interface{}{
			"resolution": string(resolution),
			"reason":     reason,
		})

	return nil
}

// Reopen reopens a closed ticket.
// The ticket will be set to ready or blocked status depending on dependencies.
func (s *TicketService) Reopen(ticketID int64) error {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Check if ticket can be reopened
	if !state.CanBeReopened(ticket.Status) {
		return newTicketError(ErrCodeInvalidState,
			fmt.Sprintf("ticket cannot be reopened in status: %s (must be closed)", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	previousStatus := ticket.Status
	previousResolution := ticket.Resolution

	// Determine new status: blocked if has deps, ready otherwise
	hasUnresolved, err := s.depRepo.HasUnresolvedDependencies(ticket.ID)
	if err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to check dependencies: %v", err), nil)
	}

	newStatus := models.StatusReady
	if hasUnresolved {
		newStatus = models.StatusBlocked
	}

	// Update ticket
	ticket.Status = newStatus
	ticket.Resolution = nil
	ticket.CompletedAt = nil
	if err := s.ticketRepo.Update(ticket); err != nil {
		return newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket: %v", err), nil)
	}

	// Log activity
	details := map[string]interface{}{
		"previous_status": string(previousStatus),
	}
	if previousResolution != nil {
		details["previous_resolution"] = string(*previousResolution)
	}
	s.activityRepo.LogActionWithDetails(ticket.ID, models.ActionReopened, models.ActorTypeHuman, "",
		fmt.Sprintf("Reopened from %s", previousStatus),
		details)

	return nil
}

// ResumeResult contains the result of resuming a ticket.
type ResumeResult struct {
	Ticket *models.Ticket `json:"ticket"`
	Claim  *models.Claim  `json:"claim"`
	Branch string         `json:"branch"`
}

// Resume resumes work on a ticket that is in human status.
// This is used when an agent wants to continue work after human input.
// It creates a new claim and transitions the ticket to in_progress.
func (s *TicketService) Resume(ticketID int64, workerID string, duration time.Duration) (*ResumeResult, error) {
	if workerID == "" {
		return nil, newTicketError(ErrCodeInvalidReason, "worker ID is required", nil)
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return nil, newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}

	// Must be in human status
	if ticket.Status != models.StatusHuman {
		return nil, newTicketError(ErrCodeInvalidState,
			fmt.Sprintf("ticket must be in human status to resume (current: %s)", ticket.Status),
			map[string]interface{}{"current_status": ticket.Status})
	}

	// Validate state machine transition
	if err := s.stateMachine.CanTransition(ticket, models.StatusInProgress, state.TransitionTypeManual, "", nil); err != nil {
		return nil, newTicketError(ErrCodeInvalidState, fmt.Sprintf("cannot resume ticket: %v", err), nil)
	}

	// Check for existing active claim (shouldn't happen for human status, but be safe)
	existingClaim, err := s.claimRepo.GetActiveByTicketID(ticket.ID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to check existing claims: %v", err), nil)
	}
	if existingClaim != nil {
		return nil, newTicketError(ErrCodeAlreadyClaimed,
			fmt.Sprintf("ticket already has an active claim by %s", existingClaim.WorkerID),
			map[string]interface{}{"worker_id": existingClaim.WorkerID})
	}

	// Create claim
	claim := models.NewClaim(ticket.ID, workerID, duration)
	if err := s.claimRepo.Create(claim); err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to create claim: %v", err), nil)
	}

	// Update ticket status
	previousReason := ticket.HumanFlagReason
	ticket.Status = models.StatusInProgress
	ticket.RetryCount = 0           // Reset retry count on resume
	ticket.HumanFlagReason = ""     // Clear the flag reason
	if err := s.ticketRepo.Update(ticket); err != nil {
		// Rollback claim
		s.claimRepo.Release(claim.ID, models.ClaimStatusReleased)
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to update ticket status: %v", err), nil)
	}

	// Log activity
	durationMins := int(duration.Minutes())
	s.activityRepo.LogActionWithDetails(ticket.ID, models.ActionHumanResponded, models.ActorTypeAgent, workerID,
		fmt.Sprintf("Resumed work (expires in %dm)", durationMins),
		map[string]interface{}{
			"worker_id":          workerID,
			"duration_mins":      durationMins,
			"expires_at":         claim.ExpiresAt.Format(time.RFC3339),
			"previous_flag_reason": previousReason,
		})

	// Generate branch name if needed
	branchName := ticket.BranchName
	if branchName == "" {
		branchName = GenerateBranchName(ticket.ProjectKey, ticket.Number, ticket.Title)
	}

	return &ResumeResult{
		Ticket: ticket,
		Claim:  claim,
		Branch: branchName,
	}, nil
}

// GenerateBranchName generates a git branch name for a ticket.
// This is exported for use by CLI when displaying branch info.
func GenerateBranchName(projectKey string, number int, title string) string {
	// Convert title to slug
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, slug)

	// Remove consecutive dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")

	// Truncate to 50 chars
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.TrimRight(slug, "-")
	}

	return fmt.Sprintf("%s-%d-%s", projectKey, number, slug)
}

// GetTicketByID retrieves a ticket by ID. Helper method for CLI.
func (s *TicketService) GetTicketByID(ticketID int64) (*models.Ticket, error) {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, newTicketError(ErrCodeDatabase, fmt.Sprintf("failed to get ticket: %v", err), nil)
	}
	if ticket == nil {
		return nil, newTicketError(ErrCodeNotFound, "ticket not found", nil)
	}
	return ticket, nil
}
