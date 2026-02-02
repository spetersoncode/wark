// Package models defines the domain models for wark.
package models

// Status represents the state of a ticket in its lifecycle.
// State machine redesign (WARK-12):
// - blocked: has open dependencies, cannot be worked
// - ready: no blockers, available for work
// - in_progress: actively being worked
// - human: needs human decision/input (escalation)
// - review: work complete, awaiting approval
// - closed: terminal state (with resolution)
type Status string

const (
	StatusBlocked    Status = "blocked"
	StatusReady      Status = "ready"
	StatusInProgress Status = "in_progress"
	StatusHuman      Status = "human"
	StatusReview     Status = "review"
	StatusClosed     Status = "closed"
)

// IsValid returns true if the status is a valid ticket status.
func (s Status) IsValid() bool {
	switch s {
	case StatusBlocked, StatusReady, StatusInProgress, StatusHuman, StatusReview, StatusClosed:
		return true
	}
	return false
}

// IsTerminal returns true if the status is a terminal state.
func (s Status) IsTerminal() bool {
	return s == StatusClosed
}

// IsWorkable returns true if the status allows the ticket to be worked on.
func (s Status) IsWorkable() bool {
	return s == StatusReady
}

// CanModifyDependencies returns true if dependencies can be modified in this state.
// Dependencies can only be modified when ticket is blocked or ready.
func (s Status) CanModifyDependencies() bool {
	return s == StatusBlocked || s == StatusReady
}

// Resolution represents why a ticket was closed.
type Resolution string

const (
	ResolutionCompleted Resolution = "completed"
	ResolutionWontDo    Resolution = "wont_do"
	ResolutionDuplicate Resolution = "duplicate"
	ResolutionInvalid   Resolution = "invalid"
	ResolutionObsolete  Resolution = "obsolete"
)

// IsValid returns true if the resolution is valid.
func (r Resolution) IsValid() bool {
	switch r {
	case ResolutionCompleted, ResolutionWontDo, ResolutionDuplicate, ResolutionInvalid, ResolutionObsolete:
		return true
	}
	return false
}

// IsSuccessful returns true if this resolution indicates successful completion.
func (r Resolution) IsSuccessful() bool {
	return r == ResolutionCompleted
}

// Priority represents the importance of a ticket.
type Priority string

const (
	PriorityHighest Priority = "highest"
	PriorityHigh    Priority = "high"
	PriorityMedium  Priority = "medium"
	PriorityLow     Priority = "low"
	PriorityLowest  Priority = "lowest"
)

// IsValid returns true if the priority is a valid ticket priority.
func (p Priority) IsValid() bool {
	switch p {
	case PriorityHighest, PriorityHigh, PriorityMedium, PriorityLow, PriorityLowest:
		return true
	}
	return false
}

// Order returns the sort order for the priority (lower is more important).
func (p Priority) Order() int {
	switch p {
	case PriorityHighest:
		return 1
	case PriorityHigh:
		return 2
	case PriorityMedium:
		return 3
	case PriorityLow:
		return 4
	case PriorityLowest:
		return 5
	default:
		return 99
	}
}

// Complexity represents the estimated size/effort of a ticket.
type Complexity string

const (
	ComplexityTrivial Complexity = "trivial"
	ComplexitySmall   Complexity = "small"
	ComplexityMedium  Complexity = "medium"
	ComplexityLarge   Complexity = "large"
	ComplexityXLarge  Complexity = "xlarge"
)

// IsValid returns true if the complexity is a valid ticket complexity.
func (c Complexity) IsValid() bool {
	switch c {
	case ComplexityTrivial, ComplexitySmall, ComplexityMedium, ComplexityLarge, ComplexityXLarge:
		return true
	}
	return false
}

// ShouldDecompose returns true if tickets of this complexity should be decomposed.
func (c Complexity) ShouldDecompose() bool {
	return c == ComplexityLarge || c == ComplexityXLarge
}

// ClaimStatus represents the state of a claim on a ticket.
type ClaimStatus string

const (
	ClaimStatusActive    ClaimStatus = "active"
	ClaimStatusCompleted ClaimStatus = "completed"
	ClaimStatusExpired   ClaimStatus = "expired"
	ClaimStatusReleased  ClaimStatus = "released"
)

// IsValid returns true if the claim status is valid.
func (cs ClaimStatus) IsValid() bool {
	switch cs {
	case ClaimStatusActive, ClaimStatusCompleted, ClaimStatusExpired, ClaimStatusReleased:
		return true
	}
	return false
}

// IsTerminal returns true if the claim status represents a terminated claim.
func (cs ClaimStatus) IsTerminal() bool {
	return cs != ClaimStatusActive
}

// MessageType represents the type of an inbox message.
type MessageType string

const (
	MessageTypeQuestion   MessageType = "question"
	MessageTypeDecision   MessageType = "decision"
	MessageTypeReview     MessageType = "review"
	MessageTypeEscalation MessageType = "escalation"
	MessageTypeInfo       MessageType = "info"
)

// IsValid returns true if the message type is valid.
func (mt MessageType) IsValid() bool {
	switch mt {
	case MessageTypeQuestion, MessageTypeDecision, MessageTypeReview, MessageTypeEscalation, MessageTypeInfo:
		return true
	}
	return false
}

// RequiresResponse returns true if this message type typically requires a human response.
func (mt MessageType) RequiresResponse() bool {
	return mt == MessageTypeQuestion || mt == MessageTypeDecision || mt == MessageTypeEscalation
}

// ActorType represents who performed an action in the activity log.
type ActorType string

const (
	ActorTypeHuman  ActorType = "human"
	ActorTypeAgent  ActorType = "agent"
	ActorTypeSystem ActorType = "system"
)

// IsValid returns true if the actor type is valid.
func (at ActorType) IsValid() bool {
	switch at {
	case ActorTypeHuman, ActorTypeAgent, ActorTypeSystem:
		return true
	}
	return false
}

// Action represents the type of action logged in the activity log.
type Action string

const (
	// Lifecycle actions
	ActionCreated   Action = "created"
	ActionClaimed   Action = "claimed"
	ActionReleased  Action = "released"
	ActionExpired   Action = "expired"
	ActionCompleted Action = "completed"
	ActionAccepted  Action = "accepted"
	ActionRejected  Action = "rejected"
	ActionClosed    Action = "closed"
	ActionReopened  Action = "reopened"

	// Dependency actions
	ActionDependencyAdded   Action = "dependency_added"
	ActionDependencyRemoved Action = "dependency_removed"
	ActionBlocked           Action = "blocked"
	ActionUnblocked         Action = "unblocked"

	// Decomposition
	ActionDecomposed   Action = "decomposed"
	ActionChildCreated Action = "child_created"

	// Human interaction
	ActionEscalated      Action = "escalated"
	ActionHumanResponded Action = "human_responded"

	// Field changes
	ActionFieldChanged Action = "field_changed"

	// Comments/notes
	ActionComment Action = "comment"
)

// IsValid returns true if the action is valid.
func (a Action) IsValid() bool {
	switch a {
	case ActionCreated, ActionClaimed, ActionReleased, ActionExpired,
		ActionCompleted, ActionAccepted, ActionRejected, ActionClosed, ActionReopened,
		ActionDependencyAdded, ActionDependencyRemoved, ActionBlocked, ActionUnblocked,
		ActionDecomposed, ActionChildCreated, ActionEscalated, ActionHumanResponded,
		ActionFieldChanged, ActionComment:
		return true
	}
	return false
}
