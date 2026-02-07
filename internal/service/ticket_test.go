package service

import (
	"context"
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTicketTestProject creates a test project and returns its ID
func createTicketTestProject(t *testing.T, database *db.DB, key string) *models.Project {
	t.Helper()

	repo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: key, Name: "Test Project"}
	err := repo.Create(project)
	require.NoError(t, err)

	return project
}

// createTicketTestTicket creates a test ticket and returns it
func createTicketTestTicket(t *testing.T, database *db.DB, projectID int64, number int, status models.Status) *models.Ticket {
	t.Helper()

	ticket := &models.Ticket{
		ProjectID:  projectID,
		Number:     number,
		Title:      "Test Ticket",
		Status:     status,
		Priority:   models.PriorityMedium,
		Complexity: models.ComplexityMedium,
		MaxRetries: 3,
	}

	repo := db.NewTicketRepo(database.DB)
	err := repo.Create(ticket)
	require.NoError(t, err)

	return ticket
}

func TestTicketService_Claim(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReady)

	svc := NewTicketService(database.DB)

	t.Run("successful claim", func(t *testing.T) {
		result, err := svc.Claim(ticket.ID, "worker-123", 60*time.Minute)
		require.NoError(t, err)

		assert.Equal(t, models.StatusWorking, result.Ticket.Status)
		assert.NotNil(t, result.Claim)
		assert.Equal(t, "worker-123", result.Claim.WorkerID)
		assert.NotEmpty(t, result.Branch)
	})

	t.Run("claim in-progress ticket fails", func(t *testing.T) {
		// After first claim, ticket is working, so state check fails first
		_, err := svc.Claim(ticket.ID, "worker-456", 60*time.Minute)
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeInvalidState, svcErr.Code)
	})

	t.Run("claim ticket with existing active claim", func(t *testing.T) {
		// Create a new ticket and claim it (via review path - doesn't change status)
		ticket2 := createTicketTestTicket(t, database, project.ID, 2, models.StatusReview)
		_, err := svc.Claim(ticket2.ID, "worker-123", 60*time.Minute)
		require.NoError(t, err)

		// Try to claim again - should fail with ALREADY_CLAIMED
		_, err = svc.Claim(ticket2.ID, "worker-456", 60*time.Minute)
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeAlreadyClaimed, svcErr.Code)
	})

	t.Run("claim non-existent ticket", func(t *testing.T) {
		_, err := svc.Claim(9999, "worker-123", 60*time.Minute)
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeNotFound, svcErr.Code)
	})
}

func TestTicketService_Release(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReady)

	svc := NewTicketService(database.DB)

	// First claim the ticket
	_, err := svc.Claim(ticket.ID, "worker-123", 60*time.Minute)
	require.NoError(t, err)

	t.Run("successful release", func(t *testing.T) {
		err := svc.Release(ticket.ID, "testing release")
		require.NoError(t, err)

		// Verify ticket is back to ready
		updatedTicket, _ := svc.GetTicketByID(ticket.ID)
		assert.Equal(t, models.StatusReady, updatedTicket.Status)
		assert.Equal(t, 1, updatedTicket.RetryCount)
	})

	t.Run("release non-in-progress ticket", func(t *testing.T) {
		err := svc.Release(ticket.ID, "should fail")
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeInvalidState, svcErr.Code)
	})
}

func TestTicketService_Complete(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReady)

	svc := NewTicketService(database.DB)

	// First claim the ticket
	_, err := svc.Claim(ticket.ID, "worker-123", 60*time.Minute)
	require.NoError(t, err)

	t.Run("successful complete to review", func(t *testing.T) {
		result, err := svc.Complete(ticket.ID, "work done", false)
		require.NoError(t, err)

		assert.Equal(t, models.StatusReview, result.Ticket.Status)
		assert.False(t, result.AutoAccepted)
	})
}

func TestTicketService_CompleteWithAutoAccept(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReady)

	svc := NewTicketService(database.DB)

	// First claim the ticket
	_, err := svc.Claim(ticket.ID, "worker-123", 60*time.Minute)
	require.NoError(t, err)

	t.Run("successful complete with auto-accept", func(t *testing.T) {
		result, err := svc.Complete(ticket.ID, "work done", true)
		require.NoError(t, err)

		assert.Equal(t, models.StatusClosed, result.Ticket.Status)
		assert.True(t, result.AutoAccepted)
		require.NotNil(t, result.Ticket.Resolution)
		assert.Equal(t, models.ResolutionCompleted, *result.Ticket.Resolution)
	})
}

func TestTicketService_CompleteWithIncompleteTasks(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReady)

	// Add an incomplete task
	tasksRepo := db.NewTasksRepo(database.DB)
	_, err := tasksRepo.CreateTask(context.Background(), ticket.ID, "Test task")
	require.NoError(t, err)

	svc := NewTicketService(database.DB)

	// First claim the ticket
	_, err = svc.Claim(ticket.ID, "worker-123", 60*time.Minute)
	require.NoError(t, err)

	t.Run("complete blocked by incomplete tasks", func(t *testing.T) {
		_, err := svc.Complete(ticket.ID, "work done", false)
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeIncompleteTasks, svcErr.Code)
	})
}

func TestTicketService_Accept(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReview)

	svc := NewTicketService(database.DB)

	t.Run("successful accept", func(t *testing.T) {
		result, err := svc.Accept(ticket.ID)
		require.NoError(t, err)

		assert.Equal(t, models.StatusClosed, result.Ticket.Status)
		require.NotNil(t, result.Ticket.Resolution)
		assert.Equal(t, models.ResolutionCompleted, *result.Ticket.Resolution)
	})
}

func TestTicketService_Reject(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReview)

	svc := NewTicketService(database.DB)

	t.Run("successful reject", func(t *testing.T) {
		err := svc.Reject(ticket.ID, "tests failing")
		require.NoError(t, err)

		updatedTicket, _ := svc.GetTicketByID(ticket.ID)
		assert.Equal(t, models.StatusReady, updatedTicket.Status)
		assert.Equal(t, 1, updatedTicket.RetryCount)
	})

	t.Run("reject without reason", func(t *testing.T) {
		// Create a new ticket in review
		ticket2 := createTicketTestTicket(t, database, project.ID, 2, models.StatusReview)

		err := svc.Reject(ticket2.ID, "")
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeInvalidReason, svcErr.Code)
	})
}

func TestTicketService_Release_EscalatesToHumanOnMaxRetries(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")

	// Create ticket with retry count at max-1 (so next release will trigger escalation)
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Number:     1,
		Title:      "Test Ticket",
		Status:     models.StatusReady,
		Priority:   models.PriorityMedium,
		Complexity: models.ComplexityMedium,
		MaxRetries: 3,
		RetryCount: 2, // At 2, next failure (3) will hit max
	}
	err := ticketRepo.Create(ticket)
	require.NoError(t, err)

	svc := NewTicketService(database.DB)

	// Claim the ticket
	_, err = svc.Claim(ticket.ID, "worker-123", 60*time.Minute)
	require.NoError(t, err)

	// Release should escalate to human
	err = svc.Release(ticket.ID, "still failing")
	require.NoError(t, err)

	// Verify ticket is escalated to human, not ready
	updatedTicket, _ := svc.GetTicketByID(ticket.ID)
	assert.Equal(t, models.StatusHuman, updatedTicket.Status)
	assert.Equal(t, 3, updatedTicket.RetryCount)
	assert.Equal(t, string(models.FlagReasonMaxRetriesExceeded), updatedTicket.HumanFlagReason)

	// Verify inbox message was created
	inboxRepo := db.NewInboxRepo(database.DB)
	messages, err := inboxRepo.List(db.InboxFilter{TicketID: &ticket.ID})
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, models.MessageTypeEscalation, messages[0].MessageType)
}

func TestTicketService_Reject_EscalatesToHumanOnMaxRetries(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")

	// Create ticket in review with retry count at max-1
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Number:     1,
		Title:      "Test Ticket",
		Status:     models.StatusReview,
		Priority:   models.PriorityMedium,
		Complexity: models.ComplexityMedium,
		MaxRetries: 3,
		RetryCount: 2, // At 2, next failure (3) will hit max
	}
	err := ticketRepo.Create(ticket)
	require.NoError(t, err)

	svc := NewTicketService(database.DB)

	// Reject should escalate to human
	err = svc.Reject(ticket.ID, "tests still failing")
	require.NoError(t, err)

	// Verify ticket is escalated to human, not ready
	updatedTicket, _ := svc.GetTicketByID(ticket.ID)
	assert.Equal(t, models.StatusHuman, updatedTicket.Status)
	assert.Equal(t, 3, updatedTicket.RetryCount)
	assert.Equal(t, string(models.FlagReasonMaxRetriesExceeded), updatedTicket.HumanFlagReason)

	// Verify inbox message was created
	inboxRepo := db.NewInboxRepo(database.DB)
	messages, err := inboxRepo.List(db.InboxFilter{TicketID: &ticket.ID})
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, models.MessageTypeEscalation, messages[0].MessageType)
}

func TestTicketService_Flag(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReady)

	svc := NewTicketService(database.DB)

	t.Run("successful flag", func(t *testing.T) {
		err := svc.Flag(ticket.ID, models.FlagReasonUnclearRequirements, "need clarification", "worker-123")
		require.NoError(t, err)

		updatedTicket, _ := svc.GetTicketByID(ticket.ID)
		assert.Equal(t, models.StatusHuman, updatedTicket.Status)
		assert.Equal(t, string(models.FlagReasonUnclearRequirements), updatedTicket.HumanFlagReason)
	})

	t.Run("flag without message", func(t *testing.T) {
		ticket2 := createTicketTestTicket(t, database, project.ID, 2, models.StatusReady)

		err := svc.Flag(ticket2.ID, models.FlagReasonUnclearRequirements, "", "worker-123")
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeInvalidReason, svcErr.Code)
	})
}

func TestTicketService_Close(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReady)

	svc := NewTicketService(database.DB)

	t.Run("successful close", func(t *testing.T) {
		err := svc.Close(ticket.ID, models.ResolutionWontDo, "no longer needed")
		require.NoError(t, err)

		updatedTicket, _ := svc.GetTicketByID(ticket.ID)
		assert.Equal(t, models.StatusClosed, updatedTicket.Status)
		require.NotNil(t, updatedTicket.Resolution)
		assert.Equal(t, models.ResolutionWontDo, *updatedTicket.Resolution)
	})

	t.Run("close with invalid resolution", func(t *testing.T) {
		ticket2 := createTicketTestTicket(t, database, project.ID, 2, models.StatusReady)

		// Use a resolution value that definitely doesn't exist
		err := svc.Close(ticket2.ID, models.Resolution("not_a_real_resolution"), "reason")
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeInvalidResolution, svcErr.Code)
	})
}

func TestTicketService_Reopen(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReady)

	svc := NewTicketService(database.DB)

	// First close the ticket
	err := svc.Close(ticket.ID, models.ResolutionWontDo, "testing")
	require.NoError(t, err)

	t.Run("successful reopen", func(t *testing.T) {
		err := svc.Reopen(ticket.ID)
		require.NoError(t, err)

		updatedTicket, _ := svc.GetTicketByID(ticket.ID)
		assert.Equal(t, models.StatusReady, updatedTicket.Status)
		assert.Nil(t, updatedTicket.Resolution)
	})

	t.Run("reopen non-closed ticket", func(t *testing.T) {
		ticket2 := createTicketTestTicket(t, database, project.ID, 2, models.StatusReady)

		err := svc.Reopen(ticket2.ID)
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeInvalidState, svcErr.Code)
	})
}

func TestTicketService_ClaimReviewTicket(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	ticket := createTicketTestTicket(t, database, project.ID, 1, models.StatusReview)

	svc := NewTicketService(database.DB)

	t.Run("claim review ticket stays in review", func(t *testing.T) {
		result, err := svc.Claim(ticket.ID, "reviewer-123", 60*time.Minute)
		require.NoError(t, err)

		// Review claims should not change status
		assert.Equal(t, models.StatusReview, result.Ticket.Status)
		assert.NotNil(t, result.Claim)
	})
}

// NOTE: TestTicketService_PromoteWithDependencies skipped until draft status is added to database schema (WARK-21)

func TestTicketService_GetExecutionContext(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	project := createTicketTestProject(t, database, "TEST")
	svc := NewTicketService(database.DB)

	t.Run("fast capability - trivial complexity", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Number:     1,
			Title:      "Fast Ticket",
			Status:     models.StatusReady,
			Priority:   models.PriorityMedium,
			Complexity: models.ComplexityTrivial,
			MaxRetries: 3,
		}
		ticketRepo := db.NewTicketRepo(database.DB)
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		ctx, err := svc.GetExecutionContext(ticket.ID)
		require.NoError(t, err)

		assert.Equal(t, "fast", ctx.Capability)
		assert.NotEmpty(t, ctx.Model)
		assert.Equal(t, "", ctx.Role)
	})

	t.Run("fast capability - small complexity", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Number:     2,
			Title:      "Fast Ticket",
			Status:     models.StatusReady,
			Priority:   models.PriorityMedium,
			Complexity: models.ComplexitySmall,
			MaxRetries: 3,
		}
		ticketRepo := db.NewTicketRepo(database.DB)
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		ctx, err := svc.GetExecutionContext(ticket.ID)
		require.NoError(t, err)

		assert.Equal(t, "fast", ctx.Capability)
		assert.NotEmpty(t, ctx.Model)
	})

	t.Run("standard capability - medium complexity", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Number:     3,
			Title:      "Standard Ticket",
			Status:     models.StatusReady,
			Priority:   models.PriorityMedium,
			Complexity: models.ComplexityMedium,
			MaxRetries: 3,
		}
		ticketRepo := db.NewTicketRepo(database.DB)
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		ctx, err := svc.GetExecutionContext(ticket.ID)
		require.NoError(t, err)

		assert.Equal(t, "standard", ctx.Capability)
		assert.NotEmpty(t, ctx.Model)
	})

	t.Run("standard capability - large complexity", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Number:     4,
			Title:      "Standard Ticket Large",
			Status:     models.StatusReady,
			Priority:   models.PriorityMedium,
			Complexity: models.ComplexityLarge,
			MaxRetries: 3,
		}
		ticketRepo := db.NewTicketRepo(database.DB)
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		ctx, err := svc.GetExecutionContext(ticket.ID)
		require.NoError(t, err)

		assert.Equal(t, "standard", ctx.Capability)
		assert.NotEmpty(t, ctx.Model)
	})

	t.Run("powerful capability - xlarge complexity", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Number:     5,
			Title:      "Powerful Ticket",
			Status:     models.StatusReady,
			Priority:   models.PriorityMedium,
			Complexity: models.ComplexityXLarge,
			MaxRetries: 3,
		}
		ticketRepo := db.NewTicketRepo(database.DB)
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		ctx, err := svc.GetExecutionContext(ticket.ID)
		require.NoError(t, err)

		assert.Equal(t, "powerful", ctx.Capability)
		assert.NotEmpty(t, ctx.Model)
	})

	t.Run("with role instructions", func(t *testing.T) {
		// Create a role
		roleRepo := db.NewRoleRepo(database.DB)
		role := &models.Role{
			Name:         "test-engineer",
			Description:  "Test engineer role",
			Instructions: "You are a test engineer. Write comprehensive tests.",
		}
		err := roleRepo.Create(role)
		require.NoError(t, err)

		// Create ticket with role
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Number:     6,
			Title:      "Ticket With Role",
			Status:     models.StatusReady,
			Priority:   models.PriorityMedium,
			Complexity: models.ComplexityMedium,
			MaxRetries: 3,
			RoleID:     &role.ID,
		}
		ticketRepo := db.NewTicketRepo(database.DB)
		err = ticketRepo.Create(ticket)
		require.NoError(t, err)

		ctx, err := svc.GetExecutionContext(ticket.ID)
		require.NoError(t, err)

		assert.Equal(t, "test-engineer", ctx.Role)
		assert.Equal(t, "You are a test engineer. Write comprehensive tests.", ctx.Instructions)
		assert.Equal(t, "standard", ctx.Capability)
	})

	t.Run("non-existent ticket", func(t *testing.T) {
		_, err := svc.GetExecutionContext(99999)
		require.Error(t, err)

		svcErr, ok := err.(*TicketError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeNotFound, svcErr.Code)
	})
}

func TestGenerateWorktreeName(t *testing.T) {
	tests := []struct {
		name       string
		projectKey string
		number     int
		title      string
		expected   string
	}{
		{
			name:       "basic worktree name",
			projectKey: "WEBAPP",
			number:     42,
			title:      "Add login page",
			expected:   "WEBAPP-42-add-login-page",
		},
		{
			name:       "special characters",
			projectKey: "TEST",
			number:     1,
			title:      "Fix bug #123",
			expected:   "TEST-1-fix-bug-123",
		},
		{
			name:       "long title truncated",
			projectKey: "API",
			number:     99,
			title:      "A very long title that should be truncated because it exceeds the maximum allowed length for branch names",
			expected:   "API-99-a-very-long-title-that-should-be-truncated-because",
		},
		{
			name:       "multiple dashes collapsed",
			projectKey: "PROJ",
			number:     5,
			title:      "Title--with---multiple----dashes",
			expected:   "PROJ-5-title-with-multiple-dashes",
		},
		{
			name:       "special chars removed",
			projectKey: "X",
			number:     1,
			title:      "Special chars !@#$%^&*() here",
			expected:   "X-1-special-chars-here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateWorktreeName(tt.projectKey, tt.number, tt.title)
			assert.Equal(t, tt.expected, result)
		})
	}
}
