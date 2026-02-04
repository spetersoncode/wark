package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Status
		wantErr bool
	}{
		// Valid cases
		{"blocked lowercase", "blocked", StatusBlocked, false},
		{"ready lowercase", "ready", StatusReady, false},
		{"in_progress underscore", "in_progress", StatusInProgress, false},
		{"in-progress hyphen", "in-progress", StatusInProgress, false},
		{"human lowercase", "human", StatusHuman, false},
		{"review lowercase", "review", StatusReview, false},
		{"closed lowercase", "closed", StatusClosed, false},
		{"uppercase", "BLOCKED", StatusBlocked, false},
		{"mixed case", "In_Progress", StatusInProgress, false},
		{"with whitespace", "  ready  ", StatusReady, false},
		// Invalid cases
		{"invalid status", "invalid_status", "", true},
		{"empty", "", "", true},
		{"partial", "blo", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStatus(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid status")
				assert.Contains(t, err.Error(), "valid:")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseResolution(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Resolution
		wantErr bool
	}{
		// Valid cases
		{"completed", "completed", ResolutionCompleted, false},
		{"wont_do underscore", "wont_do", ResolutionWontDo, false},
		{"wont-do hyphen", "wont-do", ResolutionWontDo, false},
		{"duplicate", "duplicate", ResolutionDuplicate, false},
		{"invalid resolution", "invalid", ResolutionInvalid, false},
		{"obsolete", "obsolete", ResolutionObsolete, false},
		{"uppercase", "COMPLETED", ResolutionCompleted, false},
		{"with whitespace", "  duplicate  ", ResolutionDuplicate, false},
		// Invalid cases
		{"invalid", "unknown", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResolution(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid resolution")
				assert.Contains(t, err.Error(), "valid:")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Priority
		wantErr bool
	}{
		// Valid cases
		{"highest", "highest", PriorityHighest, false},
		{"high", "high", PriorityHigh, false},
		{"medium", "medium", PriorityMedium, false},
		{"low", "low", PriorityLow, false},
		{"lowest", "lowest", PriorityLowest, false},
		{"uppercase", "HIGH", PriorityHigh, false},
		{"mixed case", "Medium", PriorityMedium, false},
		{"with whitespace", "  low  ", PriorityLow, false},
		// Invalid cases
		{"invalid", "urgent", "", true},
		{"empty", "", "", true},
		{"partial", "med", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePriority(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid priority")
				assert.Contains(t, err.Error(), "valid:")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseComplexity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Complexity
		wantErr bool
	}{
		// Valid cases
		{"trivial", "trivial", ComplexityTrivial, false},
		{"small", "small", ComplexitySmall, false},
		{"medium", "medium", ComplexityMedium, false},
		{"large", "large", ComplexityLarge, false},
		{"xlarge", "xlarge", ComplexityXLarge, false},
		{"uppercase", "LARGE", ComplexityLarge, false},
		{"mixed case", "XLarge", ComplexityXLarge, false},
		{"with whitespace", "  small  ", ComplexitySmall, false},
		// Invalid cases
		{"invalid", "huge", "", true},
		{"empty", "", "", true},
		{"xl instead of xlarge", "xl", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseComplexity(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid complexity")
				assert.Contains(t, err.Error(), "valid:")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseMessageType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    MessageType
		wantErr bool
	}{
		// Valid cases
		{"question", "question", MessageTypeQuestion, false},
		{"decision", "decision", MessageTypeDecision, false},
		{"review", "review", MessageTypeReview, false},
		{"escalation", "escalation", MessageTypeEscalation, false},
		{"info", "info", MessageTypeInfo, false},
		{"uppercase", "QUESTION", MessageTypeQuestion, false},
		{"with whitespace", "  info  ", MessageTypeInfo, false},
		// Invalid cases
		{"invalid", "alert", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMessageType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid message type")
				assert.Contains(t, err.Error(), "valid:")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseFlagReason(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    FlagReason
		wantErr bool
	}{
		// Valid cases - underscore form
		{"irreconcilable_conflict", "irreconcilable_conflict", FlagReasonIrreconcilableConflict, false},
		{"unclear_requirements", "unclear_requirements", FlagReasonUnclearRequirements, false},
		{"decision_needed", "decision_needed", FlagReasonDecisionNeeded, false},
		{"access_required", "access_required", FlagReasonAccessRequired, false},
		{"blocked_external", "blocked_external", FlagReasonBlockedExternal, false},
		{"risk_assessment", "risk_assessment", FlagReasonRiskAssessment, false},
		{"out_of_scope", "out_of_scope", FlagReasonOutOfScope, false},
		{"max_retries_exceeded", "max_retries_exceeded", FlagReasonMaxRetriesExceeded, false},
		{"other", "other", FlagReasonOther, false},
		// Valid cases - hyphen form
		{"irreconcilable-conflict hyphen", "irreconcilable-conflict", FlagReasonIrreconcilableConflict, false},
		{"unclear-requirements hyphen", "unclear-requirements", FlagReasonUnclearRequirements, false},
		{"decision-needed hyphen", "decision-needed", FlagReasonDecisionNeeded, false},
		{"out-of-scope hyphen", "out-of-scope", FlagReasonOutOfScope, false},
		// Valid cases - normalization
		{"uppercase", "DECISION_NEEDED", FlagReasonDecisionNeeded, false},
		{"mixed case hyphen", "Risk-Assessment", FlagReasonRiskAssessment, false},
		{"with whitespace", "  other  ", FlagReasonOther, false},
		// Invalid cases
		{"invalid reason", "unknown_reason", "", true},
		{"empty", "", "", true},
		{"partial", "unclear", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFlagReason(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid flag reason")
				assert.Contains(t, err.Error(), "valid:")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestFlagReasonIsValid(t *testing.T) {
	// Test that all defined constants are valid
	validReasons := []FlagReason{
		FlagReasonIrreconcilableConflict,
		FlagReasonUnclearRequirements,
		FlagReasonDecisionNeeded,
		FlagReasonAccessRequired,
		FlagReasonBlockedExternal,
		FlagReasonRiskAssessment,
		FlagReasonOutOfScope,
		FlagReasonMaxRetriesExceeded,
		FlagReasonOther,
	}

	for _, reason := range validReasons {
		assert.True(t, reason.IsValid(), "expected %q to be valid", reason)
	}

	// Test invalid
	assert.False(t, FlagReason("invalid").IsValid())
	assert.False(t, FlagReason("").IsValid())
}
