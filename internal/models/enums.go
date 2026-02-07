// Package models defines the domain models for wark.
package models

import (
	"fmt"
	"strings"
)

// Status represents the state of a ticket in its lifecycle.
// State machine (WARK-12):
// - blocked: has open dependencies, cannot be worked
// - ready: no blockers, available for work
// - working: actively being worked
// - human: needs human decision/input (escalation)
// - review: work complete, awaiting approval
// - closed: terminal state (with resolution)
type Status string

const (
	StatusBlocked Status = "blocked"
	StatusReady   Status = "ready"
	StatusWorking Status = "working"
	StatusHuman   Status = "human"
	StatusReview  Status = "review"
	StatusClosed  Status = "closed"
)

// IsValid returns true if the status is a valid ticket status.
func (s Status) IsValid() bool {
	switch s {
	case StatusBlocked, StatusReady, StatusWorking, StatusHuman, StatusReview, StatusClosed:
		return true
	}
	return false
}

// ParseStatus parses a string into a Status, normalizing input.
func ParseStatus(s string) (Status, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	status := Status(normalized)
	if !status.IsValid() {
		return "", fmt.Errorf("invalid status %q (valid: blocked, ready, working, human, review, closed)", s)
	}
	return status, nil
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

// ParseResolution parses a string into a Resolution, normalizing input.
// Accepts both hyphenated (wont-do) and underscored (wont_do) forms.
func ParseResolution(s string) (Resolution, error) {
	// Normalize: lowercase, trim whitespace, convert hyphens to underscores
	normalized := strings.ToLower(strings.TrimSpace(s))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	resolution := Resolution(normalized)
	if !resolution.IsValid() {
		return "", fmt.Errorf("invalid resolution %q (valid: completed, wont_do, duplicate, invalid, obsolete)", s)
	}
	return resolution, nil
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

// ParsePriority parses a string into a Priority, normalizing input.
func ParsePriority(s string) (Priority, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	priority := Priority(normalized)
	if !priority.IsValid() {
		return "", fmt.Errorf("invalid priority %q (valid: highest, high, medium, low, lowest)", s)
	}
	return priority, nil
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

// ParseComplexity parses a string into a Complexity, normalizing input.
func ParseComplexity(s string) (Complexity, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	complexity := Complexity(normalized)
	if !complexity.IsValid() {
		return "", fmt.Errorf("invalid complexity %q (valid: trivial, small, medium, large, xlarge)", s)
	}
	return complexity, nil
}

// ShouldDecompose returns true if tickets of this complexity should be decomposed.
func (c Complexity) ShouldDecompose() bool {
	return c == ComplexityLarge || c == ComplexityXLarge
}

// Capability returns the capability level for this complexity.
// Maps complexity to execution capability: fast, standard, or powerful.
func (c Complexity) Capability() string {
	switch c {
	case ComplexityTrivial, ComplexitySmall:
		return "fast"
	case ComplexityMedium:
		return "standard"
	case ComplexityLarge, ComplexityXLarge:
		return "powerful"
	default:
		return "standard"
	}
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

// ParseMessageType parses a string into a MessageType, normalizing input.
func ParseMessageType(s string) (MessageType, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	msgType := MessageType(normalized)
	if !msgType.IsValid() {
		return "", fmt.Errorf("invalid message type %q (valid: question, decision, review, escalation, info)", s)
	}
	return msgType, nil
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

	// Task actions
	ActionTaskCompleted Action = "task_completed"

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

// TicketType represents the type of a ticket (task or epic).
type TicketType string

const (
	TicketTypeTask TicketType = "task"
	TicketTypeEpic TicketType = "epic"
)

// IsValid returns true if the ticket type is valid.
func (tt TicketType) IsValid() bool {
	switch tt {
	case TicketTypeTask, TicketTypeEpic:
		return true
	}
	return false
}

// ParseTicketType parses a string into a TicketType, normalizing input.
func ParseTicketType(s string) (TicketType, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	ticketType := TicketType(normalized)
	if !ticketType.IsValid() {
		return "", fmt.Errorf("invalid ticket type %q (valid: task, epic)", s)
	}
	return ticketType, nil
}

// FlagReason represents the reason for flagging a ticket for human attention.
type FlagReason string

const (
	FlagReasonIrreconcilableConflict FlagReason = "irreconcilable_conflict"
	FlagReasonUnclearRequirements    FlagReason = "unclear_requirements"
	FlagReasonDecisionNeeded         FlagReason = "decision_needed"
	FlagReasonAccessRequired         FlagReason = "access_required"
	FlagReasonBlockedExternal        FlagReason = "blocked_external"
	FlagReasonRiskAssessment         FlagReason = "risk_assessment"
	FlagReasonOutOfScope             FlagReason = "out_of_scope"
	FlagReasonMaxRetriesExceeded     FlagReason = "max_retries_exceeded"
	FlagReasonOther                  FlagReason = "other"
)

// IsValid returns true if the flag reason is valid.
func (fr FlagReason) IsValid() bool {
	switch fr {
	case FlagReasonIrreconcilableConflict, FlagReasonUnclearRequirements,
		FlagReasonDecisionNeeded, FlagReasonAccessRequired,
		FlagReasonBlockedExternal, FlagReasonRiskAssessment,
		FlagReasonOutOfScope, FlagReasonMaxRetriesExceeded, FlagReasonOther:
		return true
	}
	return false
}

// ParseFlagReason parses a string into a FlagReason, normalizing input.
// Accepts both hyphenated and underscored forms.
func ParseFlagReason(s string) (FlagReason, error) {
	// Normalize: lowercase, trim whitespace, convert hyphens to underscores
	normalized := strings.ToLower(strings.TrimSpace(s))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	reason := FlagReason(normalized)
	if !reason.IsValid() {
		return "", fmt.Errorf("invalid flag reason %q (valid: irreconcilable_conflict, unclear_requirements, decision_needed, access_required, blocked_external, risk_assessment, out_of_scope, max_retries_exceeded, other)", s)
	}
	return reason, nil
}
