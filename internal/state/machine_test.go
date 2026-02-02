package state

import (
	"testing"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMachine_CanTransition(t *testing.T) {
	m := NewMachine()

	tests := []struct {
		name      string
		from      models.Status
		to        models.Status
		transType TransitionType
		reason    string
		wantErr   bool
		errMsg    string
	}{
		// Valid transitions
		{
			name:      "created to ready (auto)",
			from:      models.StatusCreated,
			to:        models.StatusReady,
			transType: TransitionTypeAuto,
			wantErr:   false,
		},
		{
			name:      "created to ready (manual)",
			from:      models.StatusCreated,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "ready to blocked",
			from:      models.StatusReady,
			to:        models.StatusBlocked,
			transType: TransitionTypeAuto,
			wantErr:   false,
		},
		{
			name:      "blocked to ready",
			from:      models.StatusBlocked,
			to:        models.StatusReady,
			transType: TransitionTypeResolve,
			wantErr:   false,
		},
		{
			name:      "ready to in_progress",
			from:      models.StatusReady,
			to:        models.StatusInProgress,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "in_progress to ready (release)",
			from:      models.StatusInProgress,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "in_progress to ready (expire)",
			from:      models.StatusInProgress,
			to:        models.StatusReady,
			transType: TransitionTypeExpire,
			wantErr:   false,
		},
		{
			name:      "in_progress to review",
			from:      models.StatusInProgress,
			to:        models.StatusReview,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "in_progress to blocked",
			from:      models.StatusInProgress,
			to:        models.StatusBlocked,
			transType: TransitionTypeAuto,
			wantErr:   false,
		},
		{
			name:      "review to done",
			from:      models.StatusReview,
			to:        models.StatusDone,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "review to ready (reject with reason)",
			from:      models.StatusReview,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			reason:    "Needs more tests",
			wantErr:   false,
		},
		{
			name:      "done to ready (reopen)",
			from:      models.StatusDone,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "cancelled to ready (reopen)",
			from:      models.StatusCancelled,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "needs_human to ready",
			from:      models.StatusNeedsHuman,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "needs_human to in_progress",
			from:      models.StatusNeedsHuman,
			to:        models.StatusInProgress,
			transType: TransitionTypeManual,
			wantErr:   false,
		},

		// Transitions to needs_human (require reason)
		{
			name:      "ready to needs_human with reason",
			from:      models.StatusReady,
			to:        models.StatusNeedsHuman,
			transType: TransitionTypeManual,
			reason:    "Need clarification on requirements",
			wantErr:   false,
		},
		{
			name:      "in_progress to needs_human with reason",
			from:      models.StatusInProgress,
			to:        models.StatusNeedsHuman,
			transType: TransitionTypeAuto,
			reason:    "Max retries exceeded",
			wantErr:   false,
		},

		// Transitions to cancelled
		{
			name:      "ready to cancelled",
			from:      models.StatusReady,
			to:        models.StatusCancelled,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "in_progress to cancelled",
			from:      models.StatusInProgress,
			to:        models.StatusCancelled,
			transType: TransitionTypeManual,
			wantErr:   false,
		},

		// Invalid transitions
		{
			name:      "same state",
			from:      models.StatusReady,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			wantErr:   true,
			errMsg:    "already in status",
		},
		{
			name:      "ready to done (skip review)",
			from:      models.StatusReady,
			to:        models.StatusDone,
			transType: TransitionTypeManual,
			wantErr:   true,
			errMsg:    "not allowed",
		},
		{
			name:      "done to in_progress (invalid)",
			from:      models.StatusDone,
			to:        models.StatusInProgress,
			transType: TransitionTypeManual,
			wantErr:   true,
			errMsg:    "not allowed",
		},
		{
			name:      "wrong transition type",
			from:      models.StatusReady,
			to:        models.StatusInProgress,
			transType: TransitionTypeAuto,
			wantErr:   true,
			errMsg:    "not allowed",
		},
		{
			name:      "review to ready without reason",
			from:      models.StatusReview,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			reason:    "",
			wantErr:   true,
			errMsg:    "reason is required",
		},
		{
			name:      "needs_human without reason",
			from:      models.StatusReady,
			to:        models.StatusNeedsHuman,
			transType: TransitionTypeManual,
			reason:    "",
			wantErr:   true,
			errMsg:    "reason is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket := &models.Ticket{
				ID:         1,
				ProjectID:  1,
				Number:     1,
				Title:      "Test Ticket",
				Status:     tt.from,
				Priority:   models.PriorityMedium,
				Complexity: models.ComplexityMedium,
			}

			err := m.CanTransition(ticket, tt.to, tt.transType, tt.reason)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMachine_NilTicket(t *testing.T) {
	m := NewMachine()

	err := m.CanTransition(nil, models.StatusReady, TransitionTypeAuto, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestMachine_ValidateTransition(t *testing.T) {
	m := NewMachine()

	ticket := &models.Ticket{
		ID:         1,
		ProjectID:  1,
		Number:     1,
		Title:      "Test Ticket",
		Status:     models.StatusReady,
		Priority:   models.PriorityMedium,
		Complexity: models.ComplexityMedium,
	}

	t.Run("valid transition", func(t *testing.T) {
		trans := NewTransition(models.StatusReady, models.StatusInProgress,
			TransitionTypeManual, models.ActorTypeAgent, "worker-1", "")
		err := m.ValidateTransition(ticket, trans)
		require.NoError(t, err)
	})

	t.Run("nil transition", func(t *testing.T) {
		err := m.ValidateTransition(ticket, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("mismatched from state", func(t *testing.T) {
		trans := NewTransition(models.StatusCreated, models.StatusReady,
			TransitionTypeAuto, models.ActorTypeSystem, "", "")
		err := m.ValidateTransition(ticket, trans)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status is ready")
	})
}

func TestMachine_GetValidTransitions(t *testing.T) {
	m := NewMachine()

	t.Run("from ready", func(t *testing.T) {
		transitions := m.GetValidTransitions(models.StatusReady)
		require.NotEmpty(t, transitions)

		// Should include blocked, in_progress, needs_human, cancelled
		toStates := make(map[models.Status]bool)
		for _, tr := range transitions {
			toStates[tr.To] = true
		}

		assert.True(t, toStates[models.StatusBlocked])
		assert.True(t, toStates[models.StatusInProgress])
		assert.True(t, toStates[models.StatusNeedsHuman])
		assert.True(t, toStates[models.StatusCancelled])
	})

	t.Run("from done", func(t *testing.T) {
		transitions := m.GetValidTransitions(models.StatusDone)
		require.Len(t, transitions, 1)
		assert.Equal(t, models.StatusReady, transitions[0].To)
	})
}

func TestMachine_GetTransitionRule(t *testing.T) {
	m := NewMachine()

	t.Run("valid rule", func(t *testing.T) {
		rule := m.GetTransitionRule(models.StatusReady, models.StatusInProgress)
		require.NotNil(t, rule)
		assert.Equal(t, models.StatusReady, rule.From)
		assert.Equal(t, models.StatusInProgress, rule.To)
	})

	t.Run("invalid rule", func(t *testing.T) {
		rule := m.GetTransitionRule(models.StatusReady, models.StatusDone)
		assert.Nil(t, rule)
	})
}

func TestActionForTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     models.Status
		to       models.Status
		transTyp TransitionType
		want     models.Action
	}{
		{"created to ready", models.StatusCreated, models.StatusReady, TransitionTypeAuto, models.ActionVetted},
		{"blocked to ready", models.StatusBlocked, models.StatusReady, TransitionTypeResolve, models.ActionUnblocked},
		{"in_progress to ready (release)", models.StatusInProgress, models.StatusReady, TransitionTypeManual, models.ActionReleased},
		{"in_progress to ready (expire)", models.StatusInProgress, models.StatusReady, TransitionTypeExpire, models.ActionExpired},
		{"needs_human to ready", models.StatusNeedsHuman, models.StatusReady, TransitionTypeManual, models.ActionHumanResponded},
		{"review to ready", models.StatusReview, models.StatusReady, TransitionTypeManual, models.ActionRejected},
		{"done to ready", models.StatusDone, models.StatusReady, TransitionTypeManual, models.ActionReopened},
		{"cancelled to ready", models.StatusCancelled, models.StatusReady, TransitionTypeManual, models.ActionReopened},
		{"any to blocked", models.StatusReady, models.StatusBlocked, TransitionTypeAuto, models.ActionBlocked},
		{"ready to in_progress", models.StatusReady, models.StatusInProgress, TransitionTypeManual, models.ActionClaimed},
		{"needs_human to in_progress", models.StatusNeedsHuman, models.StatusInProgress, TransitionTypeManual, models.ActionHumanResponded},
		{"any to needs_human", models.StatusInProgress, models.StatusNeedsHuman, TransitionTypeAuto, models.ActionFlaggedHuman},
		{"in_progress to review", models.StatusInProgress, models.StatusReview, TransitionTypeManual, models.ActionCompleted},
		{"review to done", models.StatusReview, models.StatusDone, TransitionTypeManual, models.ActionAccepted},
		{"any to cancelled", models.StatusReady, models.StatusCancelled, TransitionTypeManual, models.ActionCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ActionForTransition(tt.from, tt.to, tt.transTyp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("IsActiveState", func(t *testing.T) {
		assert.True(t, IsActiveState(models.StatusCreated))
		assert.True(t, IsActiveState(models.StatusReady))
		assert.True(t, IsActiveState(models.StatusInProgress))
		assert.True(t, IsActiveState(models.StatusBlocked))
		assert.True(t, IsActiveState(models.StatusNeedsHuman))
		assert.True(t, IsActiveState(models.StatusReview))
		assert.False(t, IsActiveState(models.StatusDone))
		assert.False(t, IsActiveState(models.StatusCancelled))
	})

	t.Run("CanBeFlagged", func(t *testing.T) {
		assert.True(t, CanBeFlagged(models.StatusCreated))
		assert.True(t, CanBeFlagged(models.StatusReady))
		assert.True(t, CanBeFlagged(models.StatusInProgress))
		assert.True(t, CanBeFlagged(models.StatusBlocked))
		assert.True(t, CanBeFlagged(models.StatusReview))
		assert.False(t, CanBeFlagged(models.StatusNeedsHuman))
		assert.False(t, CanBeFlagged(models.StatusDone))
		assert.False(t, CanBeFlagged(models.StatusCancelled))
	})

	t.Run("CanBeCancelled", func(t *testing.T) {
		assert.True(t, CanBeCancelled(models.StatusCreated))
		assert.True(t, CanBeCancelled(models.StatusReady))
		assert.True(t, CanBeCancelled(models.StatusInProgress))
		assert.True(t, CanBeCancelled(models.StatusBlocked))
		assert.True(t, CanBeCancelled(models.StatusNeedsHuman))
		assert.True(t, CanBeCancelled(models.StatusReview))
		assert.False(t, CanBeCancelled(models.StatusDone))
		assert.False(t, CanBeCancelled(models.StatusCancelled))
	})

	t.Run("CanBeReopened", func(t *testing.T) {
		assert.True(t, CanBeReopened(models.StatusDone))
		assert.True(t, CanBeReopened(models.StatusCancelled))
		assert.False(t, CanBeReopened(models.StatusCreated))
		assert.False(t, CanBeReopened(models.StatusReady))
		assert.False(t, CanBeReopened(models.StatusInProgress))
		assert.False(t, CanBeReopened(models.StatusBlocked))
		assert.False(t, CanBeReopened(models.StatusNeedsHuman))
		assert.False(t, CanBeReopened(models.StatusReview))
	})
}

// Mock implementations for testing Logic
type mockDependencyChecker struct {
	hasUnresolved bool
	unresolved    []*models.Ticket
	err           error
}

func (m *mockDependencyChecker) HasUnresolvedDependencies(ticketID int64) (bool, error) {
	return m.hasUnresolved, m.err
}

func (m *mockDependencyChecker) GetUnresolvedDependencies(ticketID int64) ([]*models.Ticket, error) {
	return m.unresolved, m.err
}

type mockTicketFetcher struct {
	ticket   *models.Ticket
	children []*models.Ticket
	err      error
}

func (m *mockTicketFetcher) GetByID(id int64) (*models.Ticket, error) {
	return m.ticket, m.err
}

func (m *mockTicketFetcher) GetChildren(parentID int64) ([]*models.Ticket, error) {
	return m.children, m.err
}

type mockClaimChecker struct {
	hasActive bool
	expired   []*models.Claim
	err       error
}

func (m *mockClaimChecker) HasActiveClaim(ticketID int64) (bool, error) {
	return m.hasActive, m.err
}

func (m *mockClaimChecker) ListExpired() ([]*models.Claim, error) {
	return m.expired, m.err
}

func TestLogic_CheckDependencies(t *testing.T) {
	ticket := &models.Ticket{ID: 1, Status: models.StatusReady}

	t.Run("no dependencies", func(t *testing.T) {
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: false}, nil, nil)
		resolved, err := logic.CheckDependencies(ticket)
		require.NoError(t, err)
		assert.True(t, resolved)
	})

	t.Run("has unresolved dependencies", func(t *testing.T) {
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		resolved, err := logic.CheckDependencies(ticket)
		require.NoError(t, err)
		assert.False(t, resolved)
	})

	t.Run("nil dependency checker", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		resolved, err := logic.CheckDependencies(ticket)
		require.NoError(t, err)
		assert.True(t, resolved) // No checker means no dependencies
	})
}

func TestLogic_ShouldBlock(t *testing.T) {
	t.Run("ready with unresolved deps should block", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusReady}
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		shouldBlock, err := logic.ShouldBlock(ticket)
		require.NoError(t, err)
		assert.True(t, shouldBlock)
	})

	t.Run("ready with resolved deps should not block", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusReady}
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: false}, nil, nil)
		shouldBlock, err := logic.ShouldBlock(ticket)
		require.NoError(t, err)
		assert.False(t, shouldBlock)
	})

	t.Run("done ticket should not block", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusDone}
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		shouldBlock, err := logic.ShouldBlock(ticket)
		require.NoError(t, err)
		assert.False(t, shouldBlock)
	})

	t.Run("needs_human should not block", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusNeedsHuman}
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		shouldBlock, err := logic.ShouldBlock(ticket)
		require.NoError(t, err)
		assert.False(t, shouldBlock)
	})
}

func TestLogic_CheckClaimExpiration(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("nil claim", func(t *testing.T) {
		assert.False(t, logic.CheckClaimExpiration(nil))
	})

	t.Run("expired claim", func(t *testing.T) {
		claim := &models.Claim{
			Status:    models.ClaimStatusActive,
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		assert.True(t, logic.CheckClaimExpiration(claim))
	})

	t.Run("active claim", func(t *testing.T) {
		claim := &models.Claim{
			Status:    models.ClaimStatusActive,
			ExpiresAt: time.Now().Add(time.Hour),
		}
		assert.False(t, logic.CheckClaimExpiration(claim))
	})
}

func TestLogic_ShouldEscalateToHuman(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("nil ticket", func(t *testing.T) {
		assert.False(t, logic.ShouldEscalateToHuman(nil))
	})

	t.Run("exceeded retries", func(t *testing.T) {
		ticket := &models.Ticket{RetryCount: 3, MaxRetries: 3}
		assert.True(t, logic.ShouldEscalateToHuman(ticket))
	})

	t.Run("under max retries", func(t *testing.T) {
		ticket := &models.Ticket{RetryCount: 2, MaxRetries: 3}
		assert.False(t, logic.ShouldEscalateToHuman(ticket))
	})
}

func TestLogic_CheckParentCompletion(t *testing.T) {
	t.Run("all children done", func(t *testing.T) {
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusDone},
			{ID: 3, Status: models.StatusDone},
		}
		logic := NewLogic(nil, &mockTicketFetcher{children: children}, nil)
		parent := &models.Ticket{ID: 1}
		complete, err := logic.CheckParentCompletion(parent)
		require.NoError(t, err)
		assert.True(t, complete)
	})

	t.Run("mixed terminal states", func(t *testing.T) {
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusDone},
			{ID: 3, Status: models.StatusCancelled},
		}
		logic := NewLogic(nil, &mockTicketFetcher{children: children}, nil)
		parent := &models.Ticket{ID: 1}
		complete, err := logic.CheckParentCompletion(parent)
		require.NoError(t, err)
		assert.True(t, complete) // All terminal
	})

	t.Run("incomplete children", func(t *testing.T) {
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusDone},
			{ID: 3, Status: models.StatusInProgress},
		}
		logic := NewLogic(nil, &mockTicketFetcher{children: children}, nil)
		parent := &models.Ticket{ID: 1}
		complete, err := logic.CheckParentCompletion(parent)
		require.NoError(t, err)
		assert.False(t, complete)
	})

	t.Run("no children", func(t *testing.T) {
		logic := NewLogic(nil, &mockTicketFetcher{children: []*models.Ticket{}}, nil)
		parent := &models.Ticket{ID: 1}
		complete, err := logic.CheckParentCompletion(parent)
		require.NoError(t, err)
		assert.False(t, complete) // No children = not complete
	})

	t.Run("nil parent", func(t *testing.T) {
		logic := NewLogic(nil, &mockTicketFetcher{}, nil)
		complete, err := logic.CheckParentCompletion(nil)
		require.NoError(t, err)
		assert.False(t, complete)
	})
}

func TestLogic_AllChildrenDone(t *testing.T) {
	t.Run("all children done", func(t *testing.T) {
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusDone},
			{ID: 3, Status: models.StatusDone},
		}
		logic := NewLogic(nil, &mockTicketFetcher{children: children}, nil)
		parent := &models.Ticket{ID: 1}
		allDone, err := logic.AllChildrenDone(parent)
		require.NoError(t, err)
		assert.True(t, allDone)
	})

	t.Run("some cancelled", func(t *testing.T) {
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusDone},
			{ID: 3, Status: models.StatusCancelled},
		}
		logic := NewLogic(nil, &mockTicketFetcher{children: children}, nil)
		parent := &models.Ticket{ID: 1}
		allDone, err := logic.AllChildrenDone(parent)
		require.NoError(t, err)
		assert.False(t, allDone) // Cancelled != done
	})
}

func TestLogic_GetNextStatus(t *testing.T) {
	t.Run("validated event", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusCreated}
		next, changed := logic.GetNextStatus(ticket, EventValidated)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReady, next)
	})

	t.Run("dependency resolved event", func(t *testing.T) {
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: false}, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusBlocked}
		next, changed := logic.GetNextStatus(ticket, EventDependencyResolved)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReady, next)
	})

	t.Run("claim expired with escalation", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusInProgress, RetryCount: 3, MaxRetries: 3}
		next, changed := logic.GetNextStatus(ticket, EventClaimExpired)
		assert.True(t, changed)
		assert.Equal(t, models.StatusNeedsHuman, next)
	})

	t.Run("claim expired without escalation", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusInProgress, RetryCount: 1, MaxRetries: 3}
		next, changed := logic.GetNextStatus(ticket, EventClaimExpired)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReady, next)
	})

	t.Run("work completed", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusInProgress}
		next, changed := logic.GetNextStatus(ticket, EventWorkCompleted)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReview, next)
	})

	t.Run("accepted", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusReview}
		next, changed := logic.GetNextStatus(ticket, EventAccepted)
		assert.True(t, changed)
		assert.Equal(t, models.StatusDone, next)
	})

	t.Run("rejected", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusReview}
		next, changed := logic.GetNextStatus(ticket, EventRejected)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReady, next)
	})
}

func TestLogic_CanClaim(t *testing.T) {
	t.Run("can claim ready ticket", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusReady}
		logic := NewLogic(
			&mockDependencyChecker{hasUnresolved: false},
			nil,
			&mockClaimChecker{hasActive: false},
		)
		canClaim, reason := logic.CanClaim(ticket)
		assert.True(t, canClaim)
		assert.Empty(t, reason)
	})

	t.Run("cannot claim non-ready ticket", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusInProgress}
		logic := NewLogic(nil, nil, nil)
		canClaim, reason := logic.CanClaim(ticket)
		assert.False(t, canClaim)
		assert.Contains(t, reason, "ready")
	})

	t.Run("cannot claim with active claim", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusReady}
		logic := NewLogic(
			&mockDependencyChecker{hasUnresolved: false},
			nil,
			&mockClaimChecker{hasActive: true},
		)
		canClaim, reason := logic.CanClaim(ticket)
		assert.False(t, canClaim)
		assert.Contains(t, reason, "already has an active claim")
	})

	t.Run("cannot claim with unresolved deps", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusReady}
		logic := NewLogic(
			&mockDependencyChecker{hasUnresolved: true},
			nil,
			&mockClaimChecker{hasActive: false},
		)
		canClaim, reason := logic.CanClaim(ticket)
		assert.False(t, canClaim)
		assert.Contains(t, reason, "unresolved dependencies")
	})
}

func TestLogic_CanComplete(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("can complete in_progress", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusInProgress}
		canComplete, _ := logic.CanComplete(ticket)
		assert.True(t, canComplete)
	})

	t.Run("cannot complete ready", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReady}
		canComplete, reason := logic.CanComplete(ticket)
		assert.False(t, canComplete)
		assert.Contains(t, reason, "in_progress")
	})
}

func TestLogic_CanAcceptReject(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("can accept in review", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReview}
		can, _ := logic.CanAccept(ticket)
		assert.True(t, can)
	})

	t.Run("can reject in review", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReview}
		can, _ := logic.CanReject(ticket)
		assert.True(t, can)
	})

	t.Run("cannot accept non-review", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusInProgress}
		can, reason := logic.CanAccept(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "review")
	})
}

func TestLogic_CanReopenCancel(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("can reopen done", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusDone}
		can, _ := logic.CanReopen(ticket)
		assert.True(t, can)
	})

	t.Run("can reopen cancelled", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusCancelled}
		can, _ := logic.CanReopen(ticket)
		assert.True(t, can)
	})

	t.Run("cannot reopen ready", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReady}
		can, reason := logic.CanReopen(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "done or cancelled")
	})

	t.Run("can cancel ready", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReady}
		can, _ := logic.CanCancel(ticket)
		assert.True(t, can)
	})

	t.Run("cannot cancel done", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusDone}
		can, reason := logic.CanCancel(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "cannot be cancelled")
	})
}

func TestLogic_CanFlag(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("can flag ready", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReady}
		can, _ := logic.CanFlag(ticket)
		assert.True(t, can)
	})

	t.Run("cannot flag needs_human", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusNeedsHuman}
		can, reason := logic.CanFlag(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "cannot be flagged")
	})

	t.Run("cannot flag done", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusDone}
		can, reason := logic.CanFlag(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "cannot be flagged")
	})
}
