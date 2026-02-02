// Package state implements the ticket state machine for wark.
package state

import (
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// TransitionType describes the kind of transition being performed.
type TransitionType string

const (
	TransitionTypeAuto    TransitionType = "auto"    // System-triggered transition
	TransitionTypeManual  TransitionType = "manual"  // User/agent-triggered transition
	TransitionTypeExpire  TransitionType = "expire"  // Expiration-triggered transition
	TransitionTypeResolve TransitionType = "resolve" // Dependency resolution triggered
)

// Transition represents a state transition request.
type Transition struct {
	From      models.Status
	To        models.Status
	Type      TransitionType
	Actor     models.ActorType
	ActorID   string
	Reason    string
	Timestamp time.Time
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
	// created → ready (auto, on validation)
	{
		From:         models.StatusCreated,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeAuto, TransitionTypeManual},
		Description:  "Ticket validated and ready for work",
	},

	// ready → blocked (auto, based on dependencies)
	{
		From:         models.StatusReady,
		To:           models.StatusBlocked,
		AllowedTypes: []TransitionType{TransitionTypeAuto, TransitionTypeManual},
		Description:  "Ticket blocked by unresolved dependencies",
	},

	// blocked → ready (auto, on dependency resolution)
	{
		From:         models.StatusBlocked,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeAuto, TransitionTypeResolve},
		Description:  "Dependencies resolved, ticket unblocked",
	},

	// ready → in_progress (claim)
	{
		From:         models.StatusReady,
		To:           models.StatusInProgress,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket claimed by worker",
	},

	// in_progress → ready (release/expire)
	{
		From:         models.StatusInProgress,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeManual, TransitionTypeExpire},
		Description:  "Ticket released or claim expired",
	},

	// in_progress → review (complete)
	{
		From:         models.StatusInProgress,
		To:           models.StatusReview,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Work completed, pending review",
	},

	// in_progress → blocked (add dependency while working)
	{
		From:         models.StatusInProgress,
		To:           models.StatusBlocked,
		AllowedTypes: []TransitionType{TransitionTypeAuto, TransitionTypeManual},
		Description:  "Ticket blocked by new dependency",
	},

	// * → needs_human (flag from most states)
	{
		From:          models.StatusCreated,
		To:            models.StatusNeedsHuman,
		AllowedTypes:  []TransitionType{TransitionTypeManual, TransitionTypeAuto},
		RequireReason: true,
		Description:   "Flagged for human attention",
	},
	{
		From:          models.StatusReady,
		To:            models.StatusNeedsHuman,
		AllowedTypes:  []TransitionType{TransitionTypeManual, TransitionTypeAuto},
		RequireReason: true,
		Description:   "Flagged for human attention",
	},
	{
		From:          models.StatusBlocked,
		To:            models.StatusNeedsHuman,
		AllowedTypes:  []TransitionType{TransitionTypeManual, TransitionTypeAuto},
		RequireReason: true,
		Description:   "Flagged for human attention",
	},
	{
		From:          models.StatusInProgress,
		To:            models.StatusNeedsHuman,
		AllowedTypes:  []TransitionType{TransitionTypeManual, TransitionTypeAuto},
		RequireReason: true,
		Description:   "Flagged for human attention",
	},
	{
		From:          models.StatusReview,
		To:            models.StatusNeedsHuman,
		AllowedTypes:  []TransitionType{TransitionTypeManual, TransitionTypeAuto},
		RequireReason: true,
		Description:   "Flagged for human attention",
	},

	// needs_human → ready (human respond)
	{
		From:         models.StatusNeedsHuman,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Human responded, returning to ready",
	},

	// needs_human → in_progress (human respond while work continues)
	{
		From:         models.StatusNeedsHuman,
		To:           models.StatusInProgress,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Human responded, resuming work",
	},

	// review → done (accept)
	{
		From:         models.StatusReview,
		To:           models.StatusDone,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Work accepted and completed",
	},

	// review → ready (reject)
	{
		From:          models.StatusReview,
		To:            models.StatusReady,
		AllowedTypes:  []TransitionType{TransitionTypeManual},
		RequireReason: true,
		Description:   "Work rejected, needs revision",
	},

	// * → cancelled (cancel from most states)
	{
		From:         models.StatusCreated,
		To:           models.StatusCancelled,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket cancelled",
	},
	{
		From:         models.StatusReady,
		To:           models.StatusCancelled,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket cancelled",
	},
	{
		From:         models.StatusBlocked,
		To:           models.StatusCancelled,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket cancelled",
	},
	{
		From:         models.StatusInProgress,
		To:           models.StatusCancelled,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket cancelled",
	},
	{
		From:         models.StatusNeedsHuman,
		To:           models.StatusCancelled,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket cancelled",
	},
	{
		From:         models.StatusReview,
		To:           models.StatusCancelled,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket cancelled",
	},

	// done/cancelled → ready (reopen)
	{
		From:         models.StatusDone,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket reopened",
	},
	{
		From:         models.StatusCancelled,
		To:           models.StatusReady,
		AllowedTypes: []TransitionType{TransitionTypeManual},
		Description:  "Ticket reopened",
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
func (m *Machine) CanTransition(ticket *models.Ticket, to models.Status, transType TransitionType, reason string) error {
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

	return m.CanTransition(ticket, t.To, t.Type, t.Reason)
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

// ActionForTransition returns the appropriate Action for logging a transition.
func ActionForTransition(from, to models.Status, transType TransitionType) models.Action {
	switch to {
	case models.StatusReady:
		switch from {
		case models.StatusCreated:
			return models.ActionVetted
		case models.StatusBlocked:
			return models.ActionUnblocked
		case models.StatusInProgress:
			if transType == TransitionTypeExpire {
				return models.ActionExpired
			}
			return models.ActionReleased
		case models.StatusNeedsHuman:
			return models.ActionHumanResponded
		case models.StatusReview:
			return models.ActionRejected
		case models.StatusDone, models.StatusCancelled:
			return models.ActionReopened
		}
	case models.StatusBlocked:
		return models.ActionBlocked
	case models.StatusInProgress:
		if from == models.StatusNeedsHuman {
			return models.ActionHumanResponded
		}
		return models.ActionClaimed
	case models.StatusNeedsHuman:
		return models.ActionFlaggedHuman
	case models.StatusReview:
		return models.ActionCompleted
	case models.StatusDone:
		return models.ActionAccepted
	case models.StatusCancelled:
		return models.ActionCancelled
	}

	// Fallback - use a generic field changed action
	return models.ActionFieldChanged
}

// IsActiveState returns true if the status represents an active (non-terminal) state.
func IsActiveState(status models.Status) bool {
	return !status.IsTerminal()
}

// CanBeFlagged returns true if tickets in this status can be flagged for human attention.
func CanBeFlagged(status models.Status) bool {
	switch status {
	case models.StatusCreated, models.StatusReady, models.StatusBlocked,
		models.StatusInProgress, models.StatusReview:
		return true
	}
	return false
}

// CanBeCancelled returns true if tickets in this status can be cancelled.
func CanBeCancelled(status models.Status) bool {
	switch status {
	case models.StatusCreated, models.StatusReady, models.StatusBlocked,
		models.StatusInProgress, models.StatusNeedsHuman, models.StatusReview:
		return true
	}
	return false
}

// CanBeReopened returns true if tickets in this status can be reopened.
func CanBeReopened(status models.Status) bool {
	return status == models.StatusDone || status == models.StatusCancelled
}
