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
		name       string
		from       models.Status
		to         models.Status
		transType  TransitionType
		reason     string
		resolution *models.Resolution
		wantErr    bool
		errMsg     string
	}{
		// Auto-transitions (dependency-triggered)
		{
			name:      "blocked to ready (auto)",
			from:      models.StatusBlocked,
			to:        models.StatusReady,
			transType: TransitionTypeAuto,
			wantErr:   false,
		},
		{
			name:      "ready to blocked (auto)",
			from:      models.StatusReady,
			to:        models.StatusBlocked,
			transType: TransitionTypeAuto,
			wantErr:   false,
		},

		// Manual transitions
		{
			name:      "ready to in_progress (claim)",
			from:      models.StatusReady,
			to:        models.StatusInProgress,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "ready to human with reason",
			from:      models.StatusReady,
			to:        models.StatusHuman,
			transType: TransitionTypeManual,
			reason:    "Need clarification",
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
			name:      "in_progress to human with reason",
			from:      models.StatusInProgress,
			to:        models.StatusHuman,
			transType: TransitionTypeManual,
			reason:    "Max retries exceeded",
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
			name:      "human to in_progress (resume)",
			from:      models.StatusHuman,
			to:        models.StatusInProgress,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:       "human to closed (resolved)",
			from:       models.StatusHuman,
			to:         models.StatusClosed,
			transType:  TransitionTypeManual,
			resolution: ptrResolution(models.ResolutionCompleted),
			wantErr:    false,
		},
		{
			name:       "review to closed (accept)",
			from:       models.StatusReview,
			to:         models.StatusClosed,
			transType:  TransitionTypeManual,
			resolution: ptrResolution(models.ResolutionCompleted),
			wantErr:    false,
		},
		{
			name:      "review to in_progress (reject with reason)",
			from:      models.StatusReview,
			to:        models.StatusInProgress,
			transType: TransitionTypeManual,
			reason:    "Needs more tests",
			wantErr:   false,
		},
		{
			name:      "closed to ready (reopen)",
			from:      models.StatusClosed,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			wantErr:   false,
		},
		{
			name:      "closed to blocked (reopen with deps)",
			from:      models.StatusClosed,
			to:        models.StatusBlocked,
			transType: TransitionTypeManual,
			wantErr:   false,
		},

		// Close from various states
		{
			name:       "blocked to closed",
			from:       models.StatusBlocked,
			to:         models.StatusClosed,
			transType:  TransitionTypeManual,
			resolution: ptrResolution(models.ResolutionWontDo),
			wantErr:    false,
		},
		{
			name:       "ready to closed",
			from:       models.StatusReady,
			to:         models.StatusClosed,
			transType:  TransitionTypeManual,
			resolution: ptrResolution(models.ResolutionDuplicate),
			wantErr:    false,
		},
		{
			name:       "in_progress to closed",
			from:       models.StatusInProgress,
			to:         models.StatusClosed,
			transType:  TransitionTypeManual,
			resolution: ptrResolution(models.ResolutionObsolete),
			wantErr:    false,
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
			name:      "ready to review (skip in_progress)",
			from:      models.StatusReady,
			to:        models.StatusReview,
			transType: TransitionTypeManual,
			wantErr:   true,
			errMsg:    "not allowed",
		},
		{
			name:      "blocked to in_progress (must go through ready)",
			from:      models.StatusBlocked,
			to:        models.StatusInProgress,
			transType: TransitionTypeManual,
			wantErr:   true,
			errMsg:    "not allowed",
		},
		{
			name:      "wrong transition type for ready to in_progress",
			from:      models.StatusReady,
			to:        models.StatusInProgress,
			transType: TransitionTypeAuto,
			wantErr:   true,
			errMsg:    "not allowed",
		},
		{
			name:      "wrong transition type for blocked to ready",
			from:      models.StatusBlocked,
			to:        models.StatusReady,
			transType: TransitionTypeManual,
			wantErr:   true,
			errMsg:    "not allowed",
		},
		{
			name:      "review to in_progress without reason",
			from:      models.StatusReview,
			to:        models.StatusInProgress,
			transType: TransitionTypeManual,
			reason:    "",
			wantErr:   true,
			errMsg:    "reason is required",
		},
		{
			name:      "human without reason",
			from:      models.StatusReady,
			to:        models.StatusHuman,
			transType: TransitionTypeManual,
			reason:    "",
			wantErr:   true,
			errMsg:    "reason is required",
		},
		{
			name:      "close without resolution",
			from:      models.StatusReady,
			to:        models.StatusClosed,
			transType: TransitionTypeManual,
			wantErr:   true,
			errMsg:    "resolution is required",
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

			err := m.CanTransition(ticket, tt.to, tt.transType, tt.reason, tt.resolution)

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

func ptrResolution(r models.Resolution) *models.Resolution {
	return &r
}

func TestMachine_NilTicket(t *testing.T) {
	m := NewMachine()

	err := m.CanTransition(nil, models.StatusReady, TransitionTypeAuto, "", nil)
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
		trans := NewTransition(models.StatusBlocked, models.StatusReady,
			TransitionTypeAuto, models.ActorTypeSystem, "", "")
		err := m.ValidateTransition(ticket, trans)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status is ready")
	})

	t.Run("close transition with resolution", func(t *testing.T) {
		trans := NewCloseTransition(models.StatusReady, models.ResolutionWontDo,
			TransitionTypeManual, models.ActorTypeHuman, "user-1", "Not needed")
		err := m.ValidateTransition(ticket, trans)
		require.NoError(t, err)
	})
}

func TestMachine_GetValidTransitions(t *testing.T) {
	m := NewMachine()

	t.Run("from ready", func(t *testing.T) {
		transitions := m.GetValidTransitions(models.StatusReady)
		require.NotEmpty(t, transitions)

		// Should include blocked, in_progress, human, closed
		toStates := make(map[models.Status]bool)
		for _, tr := range transitions {
			toStates[tr.To] = true
		}

		assert.True(t, toStates[models.StatusBlocked])
		assert.True(t, toStates[models.StatusInProgress])
		assert.True(t, toStates[models.StatusHuman])
		assert.True(t, toStates[models.StatusClosed])
	})

	t.Run("from blocked", func(t *testing.T) {
		transitions := m.GetValidTransitions(models.StatusBlocked)
		require.NotEmpty(t, transitions)

		toStates := make(map[models.Status]bool)
		for _, tr := range transitions {
			toStates[tr.To] = true
		}

		assert.True(t, toStates[models.StatusReady])
		assert.True(t, toStates[models.StatusClosed])
		assert.False(t, toStates[models.StatusInProgress]) // Must go through ready
	})

	t.Run("from closed", func(t *testing.T) {
		transitions := m.GetValidTransitions(models.StatusClosed)
		require.Len(t, transitions, 2) // Can reopen to ready or blocked
	})

	t.Run("from in_progress", func(t *testing.T) {
		transitions := m.GetValidTransitions(models.StatusInProgress)
		require.NotEmpty(t, transitions)

		toStates := make(map[models.Status]bool)
		for _, tr := range transitions {
			toStates[tr.To] = true
		}

		assert.True(t, toStates[models.StatusReady])   // release
		assert.True(t, toStates[models.StatusHuman])   // escalate
		assert.True(t, toStates[models.StatusReview])  // complete
		assert.True(t, toStates[models.StatusClosed])  // cancel
		assert.False(t, toStates[models.StatusBlocked]) // cannot block from in_progress
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
		rule := m.GetTransitionRule(models.StatusBlocked, models.StatusInProgress)
		assert.Nil(t, rule)
	})
}

func TestInitialStatus(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		status := InitialStatus(false)
		assert.Equal(t, models.StatusReady, status)
	})

	t.Run("has open dependencies", func(t *testing.T) {
		status := InitialStatus(true)
		assert.Equal(t, models.StatusBlocked, status)
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
		{"blocked to ready", models.StatusBlocked, models.StatusReady, TransitionTypeAuto, models.ActionUnblocked},
		{"ready to blocked", models.StatusReady, models.StatusBlocked, TransitionTypeAuto, models.ActionBlocked},
		{"in_progress to ready (release)", models.StatusInProgress, models.StatusReady, TransitionTypeManual, models.ActionReleased},
		{"in_progress to ready (expire)", models.StatusInProgress, models.StatusReady, TransitionTypeExpire, models.ActionExpired},
		{"closed to ready", models.StatusClosed, models.StatusReady, TransitionTypeManual, models.ActionReopened},
		{"closed to blocked", models.StatusClosed, models.StatusBlocked, TransitionTypeManual, models.ActionReopened},
		{"ready to in_progress", models.StatusReady, models.StatusInProgress, TransitionTypeManual, models.ActionClaimed},
		{"human to in_progress", models.StatusHuman, models.StatusInProgress, TransitionTypeManual, models.ActionHumanResponded},
		{"review to in_progress", models.StatusReview, models.StatusInProgress, TransitionTypeManual, models.ActionRejected},
		{"any to human", models.StatusInProgress, models.StatusHuman, TransitionTypeAuto, models.ActionEscalated},
		{"in_progress to review", models.StatusInProgress, models.StatusReview, TransitionTypeManual, models.ActionCompleted},
		{"review to closed", models.StatusReview, models.StatusClosed, TransitionTypeManual, models.ActionAccepted},
		{"ready to closed", models.StatusReady, models.StatusClosed, TransitionTypeManual, models.ActionClosed},
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
		assert.True(t, IsActiveState(models.StatusBlocked))
		assert.True(t, IsActiveState(models.StatusReady))
		assert.True(t, IsActiveState(models.StatusInProgress))
		assert.True(t, IsActiveState(models.StatusHuman))
		assert.True(t, IsActiveState(models.StatusReview))
		assert.False(t, IsActiveState(models.StatusClosed))
	})

	t.Run("CanBeEscalated", func(t *testing.T) {
		assert.True(t, CanBeEscalated(models.StatusReady))
		assert.True(t, CanBeEscalated(models.StatusInProgress))
		assert.False(t, CanBeEscalated(models.StatusBlocked))
		assert.False(t, CanBeEscalated(models.StatusHuman))
		assert.False(t, CanBeEscalated(models.StatusReview))
		assert.False(t, CanBeEscalated(models.StatusClosed))
	})

	t.Run("CanBeClosed", func(t *testing.T) {
		assert.True(t, CanBeClosed(models.StatusBlocked))
		assert.True(t, CanBeClosed(models.StatusReady))
		assert.True(t, CanBeClosed(models.StatusInProgress))
		assert.True(t, CanBeClosed(models.StatusHuman))
		assert.True(t, CanBeClosed(models.StatusReview))
		assert.False(t, CanBeClosed(models.StatusClosed))
	})

	t.Run("CanBeReopened", func(t *testing.T) {
		assert.True(t, CanBeReopened(models.StatusClosed))
		assert.False(t, CanBeReopened(models.StatusBlocked))
		assert.False(t, CanBeReopened(models.StatusReady))
		assert.False(t, CanBeReopened(models.StatusInProgress))
		assert.False(t, CanBeReopened(models.StatusHuman))
		assert.False(t, CanBeReopened(models.StatusReview))
	})

	t.Run("CanModifyDependencies", func(t *testing.T) {
		assert.True(t, CanModifyDependencies(models.StatusBlocked))
		assert.True(t, CanModifyDependencies(models.StatusReady))
		assert.False(t, CanModifyDependencies(models.StatusInProgress))
		assert.False(t, CanModifyDependencies(models.StatusHuman))
		assert.False(t, CanModifyDependencies(models.StatusReview))
		assert.False(t, CanModifyDependencies(models.StatusClosed))
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

	t.Run("blocked with unresolved deps should block", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusBlocked}
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		shouldBlock, err := logic.ShouldBlock(ticket)
		require.NoError(t, err)
		assert.True(t, shouldBlock)
	})

	t.Run("closed ticket should not block", func(t *testing.T) {
		res := models.ResolutionCompleted
		ticket := &models.Ticket{ID: 1, Status: models.StatusClosed, Resolution: &res}
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		shouldBlock, err := logic.ShouldBlock(ticket)
		require.NoError(t, err)
		assert.False(t, shouldBlock)
	})

	t.Run("human should not block", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusHuman}
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		shouldBlock, err := logic.ShouldBlock(ticket)
		require.NoError(t, err)
		assert.False(t, shouldBlock)
	})

	t.Run("in_progress should not block", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusInProgress}
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
	t.Run("all children closed", func(t *testing.T) {
		res := models.ResolutionCompleted
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusClosed, Resolution: &res},
			{ID: 3, Status: models.StatusClosed, Resolution: &res},
		}
		logic := NewLogic(nil, &mockTicketFetcher{children: children}, nil)
		parent := &models.Ticket{ID: 1}
		complete, err := logic.CheckParentCompletion(parent)
		require.NoError(t, err)
		assert.True(t, complete)
	})

	t.Run("incomplete children", func(t *testing.T) {
		res := models.ResolutionCompleted
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusClosed, Resolution: &res},
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

func TestLogic_AllChildrenClosedSuccessfully(t *testing.T) {
	t.Run("all children completed", func(t *testing.T) {
		res := models.ResolutionCompleted
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusClosed, Resolution: &res},
			{ID: 3, Status: models.StatusClosed, Resolution: &res},
		}
		logic := NewLogic(nil, &mockTicketFetcher{children: children}, nil)
		parent := &models.Ticket{ID: 1}
		allDone, err := logic.AllChildrenClosedSuccessfully(parent)
		require.NoError(t, err)
		assert.True(t, allDone)
	})

	t.Run("some wont_do", func(t *testing.T) {
		completed := models.ResolutionCompleted
		wontdo := models.ResolutionWontDo
		children := []*models.Ticket{
			{ID: 2, Status: models.StatusClosed, Resolution: &completed},
			{ID: 3, Status: models.StatusClosed, Resolution: &wontdo},
		}
		logic := NewLogic(nil, &mockTicketFetcher{children: children}, nil)
		parent := &models.Ticket{ID: 1}
		allDone, err := logic.AllChildrenClosedSuccessfully(parent)
		require.NoError(t, err)
		assert.False(t, allDone) // wont_do != completed
	})
}

func TestLogic_GetNextStatus(t *testing.T) {
	t.Run("dependency added to ready ticket", func(t *testing.T) {
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusReady}
		next, res, changed := logic.GetNextStatus(ticket, EventDependencyAdded)
		assert.True(t, changed)
		assert.Equal(t, models.StatusBlocked, next)
		assert.Nil(t, res)
	})

	t.Run("dependency resolved for blocked ticket", func(t *testing.T) {
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: false}, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusBlocked}
		next, res, changed := logic.GetNextStatus(ticket, EventDependencyResolved)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReady, next)
		assert.Nil(t, res)
	})

	t.Run("claim expired with escalation", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusInProgress, RetryCount: 3, MaxRetries: 3}
		next, res, changed := logic.GetNextStatus(ticket, EventClaimExpired)
		assert.True(t, changed)
		assert.Equal(t, models.StatusHuman, next)
		assert.Nil(t, res)
	})

	t.Run("claim expired without escalation", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusInProgress, RetryCount: 1, MaxRetries: 3}
		next, res, changed := logic.GetNextStatus(ticket, EventClaimExpired)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReady, next)
		assert.Nil(t, res)
	})

	t.Run("work completed", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusInProgress}
		next, res, changed := logic.GetNextStatus(ticket, EventWorkCompleted)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReview, next)
		assert.Nil(t, res)
	})

	t.Run("accepted", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusReview}
		next, res, changed := logic.GetNextStatus(ticket, EventAccepted)
		assert.True(t, changed)
		assert.Equal(t, models.StatusClosed, next)
		require.NotNil(t, res)
		assert.Equal(t, models.ResolutionCompleted, *res)
	})

	t.Run("rejected", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusReview}
		next, res, changed := logic.GetNextStatus(ticket, EventRejected)
		assert.True(t, changed)
		assert.Equal(t, models.StatusInProgress, next)
		assert.Nil(t, res)
	})

	t.Run("human responded", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusHuman}
		next, res, changed := logic.GetNextStatus(ticket, EventHumanResponded)
		assert.True(t, changed)
		assert.Equal(t, models.StatusInProgress, next)
		assert.Nil(t, res)
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

	t.Run("cannot claim blocked ticket", func(t *testing.T) {
		ticket := &models.Ticket{ID: 1, Status: models.StatusBlocked}
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

func TestLogic_CanReopenClose(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("can reopen closed", func(t *testing.T) {
		res := models.ResolutionCompleted
		ticket := &models.Ticket{Status: models.StatusClosed, Resolution: &res}
		can, _ := logic.CanReopen(ticket)
		assert.True(t, can)
	})

	t.Run("cannot reopen ready", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReady}
		can, reason := logic.CanReopen(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "closed")
	})

	t.Run("can close ready", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReady}
		can, _ := logic.CanClose(ticket)
		assert.True(t, can)
	})

	t.Run("cannot close closed", func(t *testing.T) {
		res := models.ResolutionCompleted
		ticket := &models.Ticket{Status: models.StatusClosed, Resolution: &res}
		can, reason := logic.CanClose(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "cannot be closed")
	})
}

func TestLogic_CanEscalate(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("can escalate ready", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReady}
		can, _ := logic.CanEscalate(ticket)
		assert.True(t, can)
	})

	t.Run("can escalate in_progress", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusInProgress}
		can, _ := logic.CanEscalate(ticket)
		assert.True(t, can)
	})

	t.Run("cannot escalate human", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusHuman}
		can, reason := logic.CanEscalate(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "cannot be escalated")
	})

	t.Run("cannot escalate closed", func(t *testing.T) {
		res := models.ResolutionCompleted
		ticket := &models.Ticket{Status: models.StatusClosed, Resolution: &res}
		can, reason := logic.CanEscalate(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "cannot be escalated")
	})
}

func TestLogic_CanModifyDependencies(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("can modify blocked", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusBlocked}
		can, _ := logic.CanAddDependency(ticket)
		assert.True(t, can)
	})

	t.Run("can modify ready", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusReady}
		can, _ := logic.CanRemoveDependency(ticket)
		assert.True(t, can)
	})

	t.Run("cannot modify in_progress", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusInProgress}
		can, reason := logic.CanAddDependency(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "blocked or ready")
	})

	t.Run("cannot modify human", func(t *testing.T) {
		ticket := &models.Ticket{Status: models.StatusHuman}
		can, reason := logic.CanRemoveDependency(ticket)
		assert.False(t, can)
		assert.Contains(t, reason, "blocked or ready")
	})
}

func TestLogic_DependencyTriggers(t *testing.T) {
	t.Run("OnDependencyCompleted unblocks", func(t *testing.T) {
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: false}, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusBlocked}
		newStatus, changed := logic.OnDependencyCompleted(ticket)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReady, newStatus)
	})

	t.Run("OnDependencyCompleted stays blocked if deps remain", func(t *testing.T) {
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: true}, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusBlocked}
		newStatus, changed := logic.OnDependencyCompleted(ticket)
		assert.False(t, changed)
		assert.Equal(t, models.StatusBlocked, newStatus)
	})

	t.Run("OnDependencyAdded blocks ready ticket", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusReady}
		newStatus, changed := logic.OnDependencyAdded(ticket, false) // dep not resolved
		assert.True(t, changed)
		assert.Equal(t, models.StatusBlocked, newStatus)
	})

	t.Run("OnDependencyAdded does not block if dep resolved", func(t *testing.T) {
		logic := NewLogic(nil, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusReady}
		newStatus, changed := logic.OnDependencyAdded(ticket, true) // dep is resolved
		assert.False(t, changed)
		assert.Equal(t, models.StatusReady, newStatus)
	})

	t.Run("OnDependencyRemoved unblocks", func(t *testing.T) {
		logic := NewLogic(&mockDependencyChecker{hasUnresolved: false}, nil, nil)
		ticket := &models.Ticket{ID: 1, Status: models.StatusBlocked}
		newStatus, changed := logic.OnDependencyRemoved(ticket)
		assert.True(t, changed)
		assert.Equal(t, models.StatusReady, newStatus)
	})
}

func TestLogic_DetermineInitialStatus(t *testing.T) {
	logic := NewLogic(nil, nil, nil)

	t.Run("no deps", func(t *testing.T) {
		status := logic.DetermineInitialStatus(false)
		assert.Equal(t, models.StatusReady, status)
	})

	t.Run("has deps", func(t *testing.T) {
		status := logic.DetermineInitialStatus(true)
		assert.Equal(t, models.StatusBlocked, status)
	})
}
