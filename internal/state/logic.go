package state

import (
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// DependencyChecker provides dependency checking operations.
type DependencyChecker interface {
	HasUnresolvedDependencies(ticketID int64) (bool, error)
	GetUnresolvedDependencies(ticketID int64) ([]*models.Ticket, error)
}

// TicketFetcher provides ticket fetching operations.
type TicketFetcher interface {
	GetByID(id int64) (*models.Ticket, error)
	GetChildren(parentID int64) ([]*models.Ticket, error)
}

// ClaimChecker provides claim checking operations.
type ClaimChecker interface {
	HasActiveClaim(ticketID int64) (bool, error)
	ListExpired() ([]*models.Claim, error)
}

// Logic provides business logic operations for the state machine.
type Logic struct {
	depChecker    DependencyChecker
	ticketFetcher TicketFetcher
	claimChecker  ClaimChecker
}

// NewLogic creates a new Logic instance with the given dependencies.
func NewLogic(depChecker DependencyChecker, ticketFetcher TicketFetcher, claimChecker ClaimChecker) *Logic {
	return &Logic{
		depChecker:    depChecker,
		ticketFetcher: ticketFetcher,
		claimChecker:  claimChecker,
	}
}

// CheckDependencies checks if all dependencies for a ticket are resolved.
// Returns true if the ticket has no unresolved dependencies.
func (l *Logic) CheckDependencies(ticket *models.Ticket) (bool, error) {
	if l.depChecker == nil {
		return true, nil
	}
	hasUnresolved, err := l.depChecker.HasUnresolvedDependencies(ticket.ID)
	if err != nil {
		return false, err
	}
	return !hasUnresolved, nil
}

// GetBlockingDependencies returns all unresolved dependencies for a ticket.
func (l *Logic) GetBlockingDependencies(ticket *models.Ticket) ([]*models.Ticket, error) {
	if l.depChecker == nil {
		return nil, nil
	}
	return l.depChecker.GetUnresolvedDependencies(ticket.ID)
}

// ShouldBlock determines if a ticket should be in blocked status based on its dependencies.
// Only applies to draft, blocked, or ready tickets.
func (l *Logic) ShouldBlock(ticket *models.Ticket) (bool, error) {
	// Only check blocking for draft, blocked, or ready states
	switch ticket.Status {
	case models.StatusDraft, models.StatusBlocked, models.StatusReady:
		// These states can be affected by dependency changes
	default:
		return false, nil
	}

	resolved, err := l.CheckDependencies(ticket)
	if err != nil {
		return false, err
	}
	return !resolved, nil
}

// CheckClaimExpiration checks if a claim has expired.
func (l *Logic) CheckClaimExpiration(claim *models.Claim) bool {
	if claim == nil {
		return false
	}
	return claim.IsExpired()
}

// IsClaimExpired checks if a claim's expiration time has passed.
func (l *Logic) IsClaimExpired(expiresAt time.Time) bool {
	return time.Now().After(expiresAt)
}

// GetExpiredClaims returns all claims that have expired but are still marked active.
func (l *Logic) GetExpiredClaims() ([]*models.Claim, error) {
	if l.claimChecker == nil {
		return nil, nil
	}
	return l.claimChecker.ListExpired()
}

// HasActiveClaim checks if a ticket has an active (non-expired) claim.
func (l *Logic) HasActiveClaim(ticket *models.Ticket) (bool, error) {
	if l.claimChecker == nil {
		return false, nil
	}
	return l.claimChecker.HasActiveClaim(ticket.ID)
}

// ShouldEscalateToHuman determines if a ticket should be escalated to human attention.
// This occurs when the ticket has exceeded its maximum retry count.
func (l *Logic) ShouldEscalateToHuman(ticket *models.Ticket) bool {
	if ticket == nil {
		return false
	}
	return ticket.HasExceededRetries()
}

// ShouldAutoEscalate checks if a ticket should automatically escalate based on retry count.
// Returns the escalation reason if escalation is needed.
func (l *Logic) ShouldAutoEscalate(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, ""
	}

	if ticket.HasExceededRetries() {
		return true, "Max retries exceeded"
	}

	return false, ""
}

// CheckParentCompletion checks if all children of a parent ticket are closed.
// Returns true if all children are in a terminal state.
func (l *Logic) CheckParentCompletion(parentTicket *models.Ticket) (bool, error) {
	if l.ticketFetcher == nil {
		return false, nil
	}

	if parentTicket == nil {
		return false, nil
	}

	children, err := l.ticketFetcher.GetChildren(parentTicket.ID)
	if err != nil {
		return false, err
	}

	// No children means nothing to check
	if len(children) == 0 {
		return false, nil
	}

	// Check if all children are in terminal states
	for _, child := range children {
		if !child.Status.IsTerminal() {
			return false, nil
		}
	}

	// All children are closed
	return true, nil
}

// AllChildrenClosedSuccessfully checks if all children of a ticket are closed with completed resolution.
func (l *Logic) AllChildrenClosedSuccessfully(parentTicket *models.Ticket) (bool, error) {
	if l.ticketFetcher == nil {
		return false, nil
	}

	if parentTicket == nil {
		return false, nil
	}

	children, err := l.ticketFetcher.GetChildren(parentTicket.ID)
	if err != nil {
		return false, err
	}

	if len(children) == 0 {
		return false, nil
	}

	for _, child := range children {
		if !child.IsClosedSuccessfully() {
			return false, nil
		}
	}

	return true, nil
}

// HasIncompleteChildren checks if a parent ticket has any non-terminal children.
func (l *Logic) HasIncompleteChildren(parentTicket *models.Ticket) (bool, error) {
	if l.ticketFetcher == nil {
		return false, nil
	}

	if parentTicket == nil {
		return false, nil
	}

	children, err := l.ticketFetcher.GetChildren(parentTicket.ID)
	if err != nil {
		return false, err
	}

	for _, child := range children {
		if !child.Status.IsTerminal() {
			return true, nil
		}
	}

	return false, nil
}

// GetNextStatus determines the appropriate next status for a ticket based on its state.
// This is used for auto-transitions after certain events.
func (l *Logic) GetNextStatus(ticket *models.Ticket, event Event) (models.Status, *models.Resolution, bool) {
	switch event {
	case EventDependencyAdded:
		// Only block if currently ready and dependency is unresolved
		if ticket.Status == models.StatusReady {
			shouldBlock, err := l.ShouldBlock(ticket)
			if err == nil && shouldBlock {
				return models.StatusBlocked, nil, true
			}
		}

	case EventDependencyResolved:
		// Unblock if all dependencies are now resolved
		if ticket.Status == models.StatusBlocked {
			resolved, err := l.CheckDependencies(ticket)
			if err == nil && resolved {
				return models.StatusReady, nil, true
			}
		}

	case EventClaimExpired:
		if ticket.Status == models.StatusInProgress {
			if l.ShouldEscalateToHuman(ticket) {
				return models.StatusHuman, nil, true
			}
			return models.StatusReady, nil, true
		}

	case EventWorkCompleted:
		if ticket.Status == models.StatusInProgress {
			return models.StatusReview, nil, true
		}

	case EventAccepted:
		if ticket.Status == models.StatusReview {
			res := models.ResolutionCompleted
			return models.StatusClosed, &res, true
		}

	case EventRejected:
		if ticket.Status == models.StatusReview {
			return models.StatusInProgress, nil, true
		}

	case EventHumanResponded:
		if ticket.Status == models.StatusHuman {
			// Default to in_progress (resuming work)
			return models.StatusInProgress, nil, true
		}
	}

	return ticket.Status, nil, false
}

// Event represents a business event that may trigger a state transition.
type Event string

const (
	EventDependencyAdded    Event = "dependency_added"
	EventDependencyResolved Event = "dependency_resolved"
	EventDependencyRemoved  Event = "dependency_removed"
	EventClaimed            Event = "claimed"
	EventReleased           Event = "released"
	EventClaimExpired       Event = "claim_expired"
	EventWorkCompleted      Event = "work_completed"
	EventAccepted           Event = "accepted"
	EventRejected           Event = "rejected"
	EventEscalated          Event = "escalated"
	EventHumanResponded     Event = "human_responded"
	EventClosed             Event = "closed"
	EventReopened           Event = "reopened"
)

// CanClaim checks if a ticket can be claimed by a worker.
func (l *Logic) CanClaim(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	// Must be in ready status
	if ticket.Status != models.StatusReady {
		return false, "ticket must be in ready status to be claimed"
	}

	// Check for existing active claim
	hasClaim, err := l.HasActiveClaim(ticket)
	if err != nil {
		return false, "failed to check for existing claim"
	}
	if hasClaim {
		return false, "ticket already has an active claim"
	}

	// Check dependencies (should be resolved since it's ready, but double-check)
	resolved, err := l.CheckDependencies(ticket)
	if err != nil {
		return false, "failed to check dependencies"
	}
	if !resolved {
		return false, "ticket has unresolved dependencies"
	}

	return true, ""
}

// CanComplete checks if a ticket can be marked as completed (moved to review).
func (l *Logic) CanComplete(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	// Must be in progress
	if ticket.Status != models.StatusInProgress {
		return false, "ticket must be in_progress to be completed"
	}

	return true, ""
}

// CanAccept checks if a ticket can be accepted (moved to closed with completed resolution).
func (l *Logic) CanAccept(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	// Must be in review
	if ticket.Status != models.StatusReview {
		return false, "ticket must be in review to be accepted"
	}

	return true, ""
}

// CanReject checks if a ticket can be rejected (moved back to in_progress).
func (l *Logic) CanReject(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	// Must be in review
	if ticket.Status != models.StatusReview {
		return false, "ticket must be in review to be rejected"
	}

	return true, ""
}

// CanReopen checks if a ticket can be reopened.
func (l *Logic) CanReopen(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	if !CanBeReopened(ticket.Status) {
		return false, "ticket must be closed to be reopened"
	}

	return true, ""
}

// CanClose checks if a ticket can be closed.
func (l *Logic) CanClose(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	if !CanBeClosed(ticket.Status) {
		return false, "ticket cannot be closed from its current status"
	}

	return true, ""
}

// CanEscalate checks if a ticket can be escalated to human.
func (l *Logic) CanEscalate(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	if !CanBeEscalated(ticket.Status) {
		return false, "ticket cannot be escalated from its current status"
	}

	return true, ""
}

// CanPromote checks if a ticket can be promoted from draft to ready.
func (l *Logic) CanPromote(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	if !CanBePromoted(ticket.Status) {
		return false, "ticket must be in draft status to be promoted"
	}

	return true, ""
}

// CanAddDependency checks if a dependency can be added to the ticket.
func (l *Logic) CanAddDependency(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	if !ticket.CanModifyDependencies() {
		return false, "dependencies can only be modified when ticket is blocked or ready"
	}

	return true, ""
}

// CanRemoveDependency checks if a dependency can be removed from the ticket.
func (l *Logic) CanRemoveDependency(ticket *models.Ticket) (bool, string) {
	if ticket == nil {
		return false, "ticket is nil"
	}

	if !ticket.CanModifyDependencies() {
		return false, "dependencies can only be modified when ticket is blocked or ready"
	}

	return true, ""
}

// DetermineInitialStatus determines whether a new ticket should start as blocked or ready.
func (l *Logic) DetermineInitialStatus(hasOpenDeps bool) models.Status {
	return InitialStatus(hasOpenDeps)
}

// OnDependencyCompleted handles when a dependency ticket is completed.
// Returns the new status if a transition should occur, or the current status if not.
func (l *Logic) OnDependencyCompleted(ticket *models.Ticket) (models.Status, bool) {
	if ticket.Status != models.StatusBlocked {
		return ticket.Status, false
	}

	resolved, err := l.CheckDependencies(ticket)
	if err != nil {
		return ticket.Status, false
	}

	if resolved {
		return models.StatusReady, true
	}
	return ticket.Status, false
}

// OnDependencyAdded handles when a dependency is added to a ticket.
// Returns the new status if a transition should occur, or the current status if not.
func (l *Logic) OnDependencyAdded(ticket *models.Ticket, depIsResolved bool) (models.Status, bool) {
	// Only affect draft or ready tickets
	if ticket.Status != models.StatusReady && ticket.Status != models.StatusDraft {
		return ticket.Status, false
	}

	// If the dependency is not resolved (closed with completed), block the ticket
	if !depIsResolved {
		return models.StatusBlocked, true
	}

	return ticket.Status, false
}

// OnDependencyRemoved handles when a dependency is removed from a ticket.
// Returns the new status if a transition should occur, or the current status if not.
func (l *Logic) OnDependencyRemoved(ticket *models.Ticket) (models.Status, bool) {
	if ticket.Status != models.StatusBlocked {
		return ticket.Status, false
	}

	resolved, err := l.CheckDependencies(ticket)
	if err != nil {
		return ticket.Status, false
	}

	if resolved {
		return models.StatusReady, true
	}
	return ticket.Status, false
}
