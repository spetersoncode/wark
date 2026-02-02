package tasks

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
)

// UnblockResult represents the result of unblocking a single ticket.
type UnblockResult struct {
	TicketID     int64  `json:"ticket_id"`
	TicketKey    string `json:"ticket_key"`
	PreviousStatus string `json:"previous_status"`
	NewStatus    string `json:"new_status"`
	Reason       string `json:"reason"`
	ErrorMessage string `json:"error,omitempty"`
}

// ParentUpdateResult represents the result of updating a parent ticket.
type ParentUpdateResult struct {
	TicketID       int64  `json:"ticket_id"`
	TicketKey      string `json:"ticket_key"`
	PreviousStatus string `json:"previous_status"`
	NewStatus      string `json:"new_status"`
	ChildrenDone   int    `json:"children_done"`
	ChildrenTotal  int    `json:"children_total"`
	AutoAccepted   bool   `json:"auto_accepted"`
	ErrorMessage   string `json:"error,omitempty"`
}

// ResolutionResult represents the result of running dependency resolution.
type ResolutionResult struct {
	Unblocked      int                   `json:"unblocked"`
	ParentsUpdated int                   `json:"parents_updated"`
	Errors         int                   `json:"errors"`
	UnblockResults []*UnblockResult      `json:"unblock_results,omitempty"`
	ParentResults  []*ParentUpdateResult `json:"parent_results,omitempty"`
}

// DependencyResolver handles automatic dependency resolution.
type DependencyResolver struct {
	db           *sql.DB
	ticketRepo   *db.TicketRepo
	depRepo      *db.DependencyRepo
	activityRepo *db.ActivityRepo
}

// NewDependencyResolver creates a new DependencyResolver.
func NewDependencyResolver(database *sql.DB) *DependencyResolver {
	return &DependencyResolver{
		db:           database,
		ticketRepo:   db.NewTicketRepo(database),
		depRepo:      db.NewDependencyRepo(database),
		activityRepo: db.NewActivityRepo(database),
	}
}

// OnTicketCompleted is called when a ticket is marked as done.
// It checks all dependents and unblocks them if all their dependencies are resolved.
// It also checks if this was the last child of a parent ticket.
func (r *DependencyResolver) OnTicketCompleted(ticketID int64, autoAccept bool) (*ResolutionResult, error) {
	result := &ResolutionResult{}

	// Get the completed ticket
	ticket, err := r.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed ticket: %w", err)
	}
	if ticket == nil {
		return nil, fmt.Errorf("ticket not found")
	}

	// 1. Check dependents and unblock those whose dependencies are now all resolved
	dependents, err := r.depRepo.GetDependents(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependents: %w", err)
	}

	for _, dependent := range dependents {
		unblockResult := r.checkAndUnblock(dependent, ticket)
		result.UnblockResults = append(result.UnblockResults, unblockResult)
		if unblockResult.ErrorMessage != "" {
			result.Errors++
		} else if unblockResult.NewStatus != "" {
			result.Unblocked++
		}
	}

	// 2. Check parent ticket if this ticket has a parent
	if ticket.ParentTicketID != nil {
		parentResult := r.checkParentCompletion(*ticket.ParentTicketID, autoAccept)
		result.ParentResults = append(result.ParentResults, parentResult)
		if parentResult.ErrorMessage != "" {
			result.Errors++
		} else if parentResult.NewStatus != "" {
			result.ParentsUpdated++
		}
	}

	return result, nil
}

// checkAndUnblock checks if a dependent ticket can be unblocked and unblocks it.
func (r *DependencyResolver) checkAndUnblock(dependent *models.Ticket, completedDep *models.Ticket) *UnblockResult {
	result := &UnblockResult{
		TicketID:       dependent.ID,
		TicketKey:      dependent.TicketKey,
		PreviousStatus: string(dependent.Status),
	}

	// Only unblock tickets that are currently blocked
	if dependent.Status != models.StatusBlocked {
		return result
	}

	// Check if all dependencies are now resolved
	hasUnresolved, err := r.depRepo.HasUnresolvedDependencies(dependent.ID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to check dependencies: %v", err)
		return result
	}

	if hasUnresolved {
		// Still has unresolved dependencies
		return result
	}

	// All dependencies resolved - unblock the ticket
	dependent.Status = models.StatusReady
	if err := r.ticketRepo.Update(dependent); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to update ticket: %v", err)
		return result
	}

	result.NewStatus = string(models.StatusReady)
	result.Reason = fmt.Sprintf("dependency %s completed", completedDep.TicketKey)

	// Log the unblock activity
	r.activityRepo.LogActionWithDetails(dependent.ID, models.ActionUnblocked, models.ActorTypeSystem, "",
		fmt.Sprintf("Unblocked after %s completed", completedDep.TicketKey),
		map[string]interface{}{
			"resolved_dependency_id":  completedDep.ID,
			"resolved_dependency_key": completedDep.TicketKey,
		})

	return result
}

// checkParentCompletion checks if all children of a parent are done and updates the parent.
func (r *DependencyResolver) checkParentCompletion(parentID int64, autoAccept bool) *ParentUpdateResult {
	result := &ParentUpdateResult{
		TicketID: parentID,
	}

	// Get the parent ticket
	parent, err := r.ticketRepo.GetByID(parentID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to get parent: %v", err)
		return result
	}
	if parent == nil {
		result.ErrorMessage = "parent ticket not found"
		return result
	}

	result.TicketKey = parent.TicketKey
	result.PreviousStatus = string(parent.Status)

	// Only process parents that are not already done/cancelled
	if parent.Status.IsTerminal() {
		return result
	}

	// Get all children
	children, err := r.ticketRepo.GetChildren(parentID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to get children: %v", err)
		return result
	}

	result.ChildrenTotal = len(children)
	if result.ChildrenTotal == 0 {
		return result
	}

	// Count done children
	doneCount := 0
	for _, child := range children {
		if child.Status == models.StatusDone {
			doneCount++
		}
	}
	result.ChildrenDone = doneCount

	// Check if all children are done
	if doneCount < len(children) {
		// Not all children done yet
		return result
	}

	// All children are done - update parent status
	newStatus := models.StatusReview
	if autoAccept {
		newStatus = models.StatusDone
		result.AutoAccepted = true
	}

	parent.Status = newStatus
	if newStatus == models.StatusDone {
		now := time.Now()
		parent.CompletedAt = &now
	}

	if err := r.ticketRepo.Update(parent); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to update parent: %v", err)
		return result
	}

	result.NewStatus = string(newStatus)

	// Log activity
	action := models.ActionCompleted
	summary := "All child tickets completed - moved to review"
	if autoAccept {
		summary = "All child tickets completed - auto-accepted"
	}

	r.activityRepo.LogActionWithDetails(parent.ID, action, models.ActorTypeSystem, "",
		summary,
		map[string]interface{}{
			"children_done":  doneCount,
			"children_total": len(children),
			"auto_accepted":  autoAccept,
		})

	return result
}

// ResolveAll checks all blocked tickets and unblocks those with resolved dependencies.
// This is useful for manual batch resolution.
func (r *DependencyResolver) ResolveAll() (*ResolutionResult, error) {
	result := &ResolutionResult{}

	// Get all blocked tickets
	status := models.StatusBlocked
	blockedTickets, err := r.ticketRepo.List(db.TicketFilter{Status: &status})
	if err != nil {
		return nil, fmt.Errorf("failed to list blocked tickets: %w", err)
	}

	for _, ticket := range blockedTickets {
		// Check if all dependencies are resolved
		hasUnresolved, err := r.depRepo.HasUnresolvedDependencies(ticket.ID)
		if err != nil {
			result.Errors++
			continue
		}

		if hasUnresolved {
			continue
		}

		// Unblock the ticket
		unblockResult := &UnblockResult{
			TicketID:       ticket.ID,
			TicketKey:      ticket.TicketKey,
			PreviousStatus: string(ticket.Status),
		}

		ticket.Status = models.StatusReady
		if err := r.ticketRepo.Update(ticket); err != nil {
			unblockResult.ErrorMessage = fmt.Sprintf("failed to update ticket: %v", err)
			result.Errors++
		} else {
			unblockResult.NewStatus = string(models.StatusReady)
			unblockResult.Reason = "all dependencies resolved"
			result.Unblocked++

			// Log the unblock activity
			r.activityRepo.LogAction(ticket.ID, models.ActionUnblocked, models.ActorTypeSystem, "",
				"Unblocked - all dependencies resolved")
		}

		result.UnblockResults = append(result.UnblockResults, unblockResult)
	}

	return result, nil
}
