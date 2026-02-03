// Package state implements the ticket state machine for wark.
//
// State Machine (WARK-12, WARK-21):
//
// States:
//   - draft: being planned, not ready for AI to work on
//   - blocked: has open dependencies, cannot be worked
//   - ready: no blockers, available for work
//   - in_progress: actively being worked
//   - human: needs human decision/input (escalation)
//   - review: work complete, awaiting approval
//   - closed: terminal state (with resolution enum)
//
// Auto-transitions (dependency-triggered):
//   - ON create: if --draft flag → draft, else if has_open_deps → blocked, else → ready
//   - ON dep completed: if blocked ticket has all deps done → ready
//   - ON dep added to ready ticket: if dep not closed(completed) → blocked
//   - ON dep removed: if blocked and all deps done → ready
//
// Manual transitions:
//   - draft → ready (promote)
//   - draft → blocked (add dependency)
//   - ready → in_progress (claim)
//   - ready → human (escalate before starting)
//   - in_progress → ready (release)
//   - in_progress → human (escalate)
//   - in_progress → review (complete)
//   - human → in_progress (resume after human input)
//   - human → closed (human resolves)
//   - review → ready (reject)
//   - review → closed (accept)
//   - {any except closed} → closed (cancel with resolution)
//
// Constraint: Dependencies can only be modified when ticket is draft, blocked, or ready.
package state

import (
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// TransitionType describes the kind of transition being performed.
type TransitionType string

const (
	TransitionTypeAuto   TransitionType = "auto"   // System-triggered transition (dependency changes)
	TransitionTypeManual TransitionType = "manual" // User/agent-triggered transition
	TransitionTypeExpire TransitionType = "expire" // Expiration-triggered transition
)

// Transition represents a state transition request.
type Transition struct {
	From       models.Status
	To         models.Status
	Type       TransitionType
	Actor      models.ActorType
	ActorID    string
	Reason     string
	Resolution *models.Resolution // Required when To == StatusClosed
	Timestamp  time.Time
}

// NewTransition creates a new transition request.
func NewTransition(from, to models.Status, transType TransitionType, actor models.ActorType, actorID, reason string) *Transition {
	return &Transition{
		From:      from,
		To:        to,
		Type:      transType,
		Actor:     actor,
		ActorID:   actorID,
		Reason:    reason,
		Timestamp: time.Now(),
	}
}

// NewCloseTransition creates a transition to closed state with a resolution.
func NewCloseTransition(from models.Status, resolution models.Resolution, transType TransitionType, actor models.ActorType, actorID, reason string) *Transition {
	return &Transition{
		From:       from,
		To:         models.StatusClosed,
		Type:       transType,
		Actor:      actor,
		ActorID:    actorID,
		Reason:     reason,
		Resolution: &resolution,
		Timestamp:  time.Now(),
	}
}

// TransitionRule defines a valid state transition and its requirements.
type TransitionRule struct {
	From          models.Status
	To            models.Status
	AllowedTypes  []TransitionType
	RequireReason bool
	Description   string
}

// validTransitions defines all valid state transitions.
var validTransitions = []TransitionRule{
	// Draft transitions (WARK-21)
	// draft → ready (promote)
	{
		From:         models.StatusDraft,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket promoted from draft to ready",
	},
	// draft → blocked (dependency added or promote with deps)
	{
		From:         models.StatusDraft,
		To:           models.StatusBlocked,
		AllowedTypes: []TransitionType{TransitionTypeAuto, TransitionTypeManual},
		Description:  "Draft ticket blocked by unresolved dependency",
	},
	// draft → closed (cancel)
	{
		From:         models.StatusDraft,
		To:           models.StatusClosed,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Draft ticket closed",
	},

	// Auto-transitions (dependency-triggered)
	// blocked → ready (all dependencies resolved)
	{
		From:         models.StatusBlocked,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeAuto},
		Description:  "Dependencies resolved, ticket unblocked",
	},
	// ready → blocked (dependency added)
	{
		From:         models.StatusReady,
		To:           models.StatusBlocked,
		AllowedTypes: []TransitionType{TransitionTypeAuto},
		Description:  "Blocked by unresolved dependency",
	},

	// Manual transitions
	// ready → in_progress (claim)
	{
		From:         models.StatusReady,
		To:           models.StatusInProgress,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket claimed by worker",
	},
	// ready → human (escalate before starting)
	{
		From:          models.StatusReady,
		To:            models.StatusHuman,
		AllowedTypes:  []TransitionType{TransitionTypeManual, TransitionTypeAuto},
		RequireReason: true,
		Description:   "Escalated for human decision",
	},
	// in_progress → ready (release)
	{
		From:         models.StatusInProgress,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeManual, TransitionTypeExpire},
		Description:  "Ticket released or claim expired",
	},
	// in_progress → human (escalate)
	{
		From:          models.StatusInProgress,
		To:            models.StatusHuman,
		AllowedTypes:  []TransitionType{TransitionTypeManual, TransitionTypeAuto},
		RequireReason: true,
		Description:   "Escalated for human decision",
	},
	// in_progress → review (complete)
	{
		From:         models.StatusInProgress,
		To:           models.StatusReview,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Work completed, pending review",
	},
	// human → in_progress (resume after human input)
	{
		From:         models.StatusHuman,
		To:           models.StatusInProgress,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Human responded, resuming work",
	},
	// human → closed (human resolves)
	{
		From:         models.StatusHuman,
		To:           models.StatusClosed,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Human resolved the ticket",
	},
	// review → ready (reject)
	{
		From:          models.StatusReview,
		To:            models.StatusReady,
		AllowedTypes:  []TransitionType{TransitionTypeManual},
		RequireReason: true,
		Description:   "Work rejected, returned to queue",
	},
	// review → closed (accept)
	{
		From:         models.StatusReview,
		To:           models.StatusClosed,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Work accepted and completed",
	},

	// Cancel from any non-terminal state
	{
		From:         models.StatusBlocked,
		To:           models.StatusClosed,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket closed",
	},
	{
		From:         models.StatusReady,
		To:           models.StatusClosed,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket closed",
	},
	{
		From:         models.StatusInProgress,
		To:           models.StatusClosed,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket closed",
	},

	// Reopen from closed
	{
		From:         models.StatusClosed,
		To:           models.StatusDraft,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket reopened as draft",
	},
	{
		From:         models.StatusClosed,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket reopened",
	},
	{
		From:         models.StatusClosed,
		To:           models.StatusBlocked,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket reopened (has dependencies)",
	},
}

// transitionRuleMap provides fast lookup of transition rules.
var transitionRuleMap map[string]*TransitionRule

func init() {
	transitionRuleMap = make(map[string]*TransitionRule)
	for i := range validTransitions {
		rule := &validTransitions[i]
		key := makeTransitionKey(rule.From, rule.To)
		transitionRuleMap[key] = rule
	}
}

func makeTransitionKey(from, to models.Status) string {
	return string(from) + "->" + string(to)
}

// Machine provides state machine operations for tickets.
type Machine struct{}

// NewMachine creates a new state machine instance.
func NewMachine() *Machine {
	return &Machine{}
}

// GetTransitionRule returns the rule for a transition, or nil if invalid.
func (m *Machine) GetTransitionRule(from, to models.Status) *TransitionRule {
	return transitionRuleMap[makeTransitionKey(from, to)]
}

// CanTransition checks if a transition is valid for the given ticket.
// It returns nil if the transition is allowed, or an error explaining why not.
func (m *Machine) CanTransition(ticket *models.Ticket, to models.Status, transType TransitionType, reason string, resolution *models.Resolution) error {
	if ticket == nil {
		return fmt.Errorf("ticket is nil")
	}

	from := ticket.Status

	// Same state is not a transition
	if from == to {
		return fmt.Errorf("ticket is already in status %s", to)
	}

	// Find the transition rule
	rule := m.GetTransitionRule(from, to)
	if rule == nil {
		return fmt.Errorf("transition from %s to %s is not allowed", from, to)
	}

	// Check if the transition type is allowed
	typeAllowed := false
	for _, allowedType := range rule.AllowedTypes {
		if allowedType == transType {
			typeAllowed = true
			break
		}
	}
	if !typeAllowed {
		return fmt.Errorf("transition type %s is not allowed for %s -> %s", transType, from, to)
	}

	// Check if reason is required
	if rule.RequireReason && reason == "" {
		return fmt.Errorf("reason is required for transition from %s to %s", from, to)
	}

	// Resolution is required when transitioning to closed
	if to == models.StatusClosed {
		if resolution == nil {
			return fmt.Errorf("resolution is required when closing a ticket")
		}
		if !resolution.IsValid() {
			return fmt.Errorf("invalid resolution: %s", *resolution)
		}
	}

	return nil
}

// ValidateTransition validates a full transition request.
func (m *Machine) ValidateTransition(ticket *models.Ticket, t *Transition) error {
	if t == nil {
		return fmt.Errorf("transition is nil")
	}

	// Verify the from state matches
	if ticket.Status != t.From {
		return fmt.Errorf("ticket status is %s, but transition expects %s", ticket.Status, t.From)
	}

	return m.CanTransition(ticket, t.To, t.Type, t.Reason, t.Resolution)
}

// GetValidTransitions returns all valid transitions from the given status.
func (m *Machine) GetValidTransitions(from models.Status) []TransitionRule {
	var transitions []TransitionRule
	for _, rule := range validTransitions {
		if rule.From == from {
			transitions = append(transitions, rule)
		}
	}
	return transitions
}

// GetAllTransitionRules returns all defined transition rules.
func (m *Machine) GetAllTransitionRules() []TransitionRule {
	result := make([]TransitionRule, len(validTransitions))
	copy(result, validTransitions)
	return result
}

// InitialStatus determines the initial status for a new ticket based on dependencies.
func InitialStatus(hasOpenDeps bool) models.Status {
	if hasOpenDeps {
		return models.StatusBlocked
	}
	return models.StatusReady
}

// ActionForTransition returns the appropriate Action for logging a transition.
func ActionForTransition(from, to models.Status, transType TransitionType) models.Action {
	switch to {
	case models.StatusReady:
		switch from {
		case models.StatusDraft:
			return models.ActionPromoted
		case models.StatusBlocked:
			return models.ActionUnblocked
		case models.StatusInProgress:
			if transType == TransitionTypeExpire {
				return models.ActionExpired
			}
			return models.ActionReleased
		case models.StatusReview:
			return models.ActionRejected
		case models.StatusClosed:
			return models.ActionReopened
		}
	case models.StatusBlocked:
		if from == models.StatusClosed {
			return models.ActionReopened
		}
		if from == models.StatusDraft {
			return models.ActionBlocked
		}
		return models.ActionBlocked
	case models.StatusDraft:
		if from == models.StatusClosed {
			return models.ActionReopened
		}
		return models.ActionFieldChanged
	case models.StatusInProgress:
		if from == models.StatusHuman {
			return models.ActionHumanResponded
		}
		return models.ActionClaimed
	case models.StatusHuman:
		return models.ActionEscalated
	case models.StatusReview:
		return models.ActionCompleted
	case models.StatusClosed:
		if from == models.StatusReview {
			return models.ActionAccepted
		}
		return models.ActionClosed
	}

	// Fallback - use a generic field changed action
	return models.ActionFieldChanged
}

// IsActiveState returns true if the status represents an active (non-terminal) state.
func IsActiveState(status models.Status) bool {
	return !status.IsTerminal()
}

// CanBeEscalated returns true if tickets in this status can be escalated to human.
func CanBeEscalated(status models.Status) bool {
	switch status {
	case models.StatusReady, models.StatusInProgress:
		return true
	}
	return false
}

// CanBeClosed returns true if tickets in this status can be closed.
func CanBeClosed(status models.Status) bool {
	switch status {
	case models.StatusDraft, models.StatusBlocked, models.StatusReady, models.StatusInProgress,
		models.StatusHuman, models.StatusReview:
		return true
	}
	return false
}

// CanBeReopened returns true if tickets in this status can be reopened.
func CanBeReopened(status models.Status) bool {
	return status == models.StatusClosed
}

// CanModifyDependencies returns true if dependencies can be modified in this status.
func CanModifyDependencies(status models.Status) bool {
	return status.CanModifyDependencies()
}

// CanBePromoted returns true if tickets in this status can be promoted.
// Only draft tickets can be promoted to ready.
func CanBePromoted(status models.Status) bool {
	return status == models.StatusDraft
}
