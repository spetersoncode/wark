package service

import (
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/errors"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboxService_Respond(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	// Setup repos
	projectRepo := db.NewProjectRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create project
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create ticket in human status
	ticket := &models.Ticket{
		ProjectID:       project.ID,
		Title:           "Test Ticket",
		Status:          models.StatusHuman,
		HumanFlagReason: "test_reason",
		RetryCount:      2,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create inbox message
	msg := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "What should I do?", "agent-1")
	err = inboxRepo.Create(msg)
	require.NoError(t, err)

	// Create InboxService
	service := NewInboxService(inboxRepo, ticketRepo, claimRepo, activityRepo)

	t.Run("successful response transitions ticket to ready", func(t *testing.T) {
		result, err := service.Respond(msg.ID, "Do this!")
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Message)
		assert.NotNil(t, result.Message.RespondedAt)
		assert.True(t, result.TicketUpdated)
		assert.Equal(t, models.StatusHuman, result.PreviousStatus)
		assert.Equal(t, models.StatusReady, result.NewStatus)

		// Verify ticket was updated
		updatedTicket, err := ticketRepo.GetByID(ticket.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusReady, updatedTicket.Status)
		assert.Equal(t, 0, updatedTicket.RetryCount)       // Reset to 0
		assert.Empty(t, updatedTicket.HumanFlagReason)     // Cleared

		// Verify activity was logged
		activities, err := activityRepo.ListByTicket(ticket.ID, 10)
		require.NoError(t, err)
		assert.True(t, len(activities) >= 1)
		assert.Equal(t, models.ActionHumanResponded, activities[0].Action)
	})

	t.Run("responding to already responded message fails", func(t *testing.T) {
		_, err := service.Respond(msg.ID, "Another response")
		require.Error(t, err)

		// Verify it's a state error
		sharedErr, ok := err.(*errors.Error)
		require.True(t, ok)
		assert.Equal(t, errors.KindStateError, sharedErr.Kind)
	})

	t.Run("responding to non-existent message fails", func(t *testing.T) {
		_, err := service.Respond(99999, "Response")
		require.Error(t, err)

		// Verify it's a not found error
		sharedErr, ok := err.(*errors.Error)
		require.True(t, ok)
		assert.Equal(t, errors.KindNotFound, sharedErr.Kind)
	})

	t.Run("empty response fails", func(t *testing.T) {
		_, err := service.Respond(msg.ID, "")
		require.Error(t, err)

		sharedErr, ok := err.(*errors.Error)
		require.True(t, ok)
		assert.Equal(t, errors.KindInvalidArgs, sharedErr.Kind)
	})
}

func TestInboxService_Respond_NonHumanStatus(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	// Setup repos
	projectRepo := db.NewProjectRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create project
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create ticket in ready status (not human)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create inbox message
	msg := models.NewInboxMessage(ticket.ID, models.MessageTypeInfo, "Info message", "agent-1")
	err = inboxRepo.Create(msg)
	require.NoError(t, err)

	// Create InboxService
	service := NewInboxService(inboxRepo, ticketRepo, claimRepo, activityRepo)

	// Respond - should not change ticket status
	result, err := service.Respond(msg.ID, "Thanks for the info")
	require.NoError(t, err)

	assert.NotNil(t, result)
	assert.False(t, result.TicketUpdated) // Ticket was not in human status

	// Verify ticket status unchanged
	updatedTicket, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, updatedTicket.Status)
}

func TestInboxService_Send(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	// Setup repos
	projectRepo := db.NewProjectRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create project
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create InboxService
	service := NewInboxService(inboxRepo, ticketRepo, claimRepo, activityRepo)

	t.Run("successful send transitions ticket to human status", func(t *testing.T) {
		// Create ticket in in_progress status
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Ticket for Send Test",
			Status:    models.StatusInProgress,
		}
		err = ticketRepo.Create(ticket)
		require.NoError(t, err)

		result, err := service.Send(ticket.ID, models.MessageTypeQuestion, "What should I do?", "agent-1")
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Message)
		assert.Equal(t, ticket.ID, result.Message.TicketID)
		assert.Equal(t, models.MessageTypeQuestion, result.Message.MessageType)
		assert.True(t, result.StatusChanged)
		assert.Equal(t, models.StatusInProgress, result.PreviousStatus)
		assert.Equal(t, models.StatusHuman, result.NewStatus)

		// Verify ticket was updated
		updatedTicket, err := ticketRepo.GetByID(ticket.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusHuman, updatedTicket.Status)

		// Verify activity was logged
		activities, err := activityRepo.ListByTicket(ticket.ID, 10)
		require.NoError(t, err)
		assert.True(t, len(activities) >= 1)
		assert.Equal(t, models.ActionEscalated, activities[0].Action)
	})

	t.Run("send to already human ticket does not change status", func(t *testing.T) {
		// Create ticket already in human status
		ticket := &models.Ticket{
			ProjectID:       project.ID,
			Title:           "Already Human Ticket",
			Status:          models.StatusHuman,
			HumanFlagReason: "existing_reason",
		}
		err = ticketRepo.Create(ticket)
		require.NoError(t, err)

		result, err := service.Send(ticket.ID, models.MessageTypeQuestion, "Question", "agent-1")
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.False(t, result.StatusChanged)
		assert.Equal(t, models.StatusHuman, result.PreviousStatus)
		assert.Equal(t, models.StatusHuman, result.NewStatus)
	})

	t.Run("info type does not change status", func(t *testing.T) {
		// Create ticket in in_progress status
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Ticket for Info Test",
			Status:    models.StatusInProgress,
		}
		err = ticketRepo.Create(ticket)
		require.NoError(t, err)

		result, err := service.Send(ticket.ID, models.MessageTypeInfo, "FYI: some info", "agent-1")
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.False(t, result.StatusChanged, "info type should NOT change status")
		assert.Equal(t, models.StatusInProgress, result.PreviousStatus)
		assert.Equal(t, models.StatusInProgress, result.NewStatus)

		// Verify ticket status unchanged
		updatedTicket, err := ticketRepo.GetByID(ticket.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusInProgress, updatedTicket.Status)
	})

	t.Run("review type does not change status", func(t *testing.T) {
		// Create ticket in in_progress status
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Ticket for Review Test",
			Status:    models.StatusInProgress,
		}
		err = ticketRepo.Create(ticket)
		require.NoError(t, err)

		result, err := service.Send(ticket.ID, models.MessageTypeReview, "Please review", "agent-1")
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.False(t, result.StatusChanged, "review type should NOT change status")
		assert.Equal(t, models.StatusInProgress, result.PreviousStatus)
		assert.Equal(t, models.StatusInProgress, result.NewStatus)
	})

	t.Run("send to closed ticket does not change status", func(t *testing.T) {
		// Create closed ticket
		completedRes := models.ResolutionCompleted
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Title:      "Closed Ticket",
			Status:     models.StatusClosed,
			Resolution: &completedRes,
		}
		err = ticketRepo.Create(ticket)
		require.NoError(t, err)

		result, err := service.Send(ticket.ID, models.MessageTypeInfo, "Post-close info", "agent-1")
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.False(t, result.StatusChanged)
		assert.Equal(t, models.StatusClosed, result.PreviousStatus)
	})

	t.Run("send to non-existent ticket fails", func(t *testing.T) {
		_, err := service.Send(99999, models.MessageTypeQuestion, "Question", "agent-1")
		require.Error(t, err)

		sharedErr, ok := err.(*errors.Error)
		require.True(t, ok)
		assert.Equal(t, errors.KindNotFound, sharedErr.Kind)
	})

	t.Run("empty content fails", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Ticket for Empty Content",
			Status:    models.StatusReady,
		}
		err = ticketRepo.Create(ticket)
		require.NoError(t, err)

		_, err := service.Send(ticket.ID, models.MessageTypeQuestion, "", "agent-1")
		require.Error(t, err)

		sharedErr, ok := err.(*errors.Error)
		require.True(t, ok)
		assert.Equal(t, errors.KindInvalidArgs, sharedErr.Kind)
	})

	t.Run("invalid message type fails", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Ticket for Invalid Type",
			Status:    models.StatusReady,
		}
		err = ticketRepo.Create(ticket)
		require.NoError(t, err)

		_, err := service.Send(ticket.ID, models.MessageType("invalid"), "Content", "agent-1")
		require.Error(t, err)

		sharedErr, ok := err.(*errors.Error)
		require.True(t, ok)
		assert.Equal(t, errors.KindInvalidArgs, sharedErr.Kind)
	})
}

func TestInboxService_Send_ReleasesActiveClaim(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	// Setup repos
	projectRepo := db.NewProjectRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create project
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create ticket in in_progress status
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Ticket with Claim",
		Status:    models.StatusInProgress,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create active claim
	claim := models.NewClaim(ticket.ID, "worker-123", 60*time.Minute)
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// Verify claim is active
	activeClaim, err := claimRepo.GetActiveByTicketID(ticket.ID)
	require.NoError(t, err)
	require.NotNil(t, activeClaim)

	// Create InboxService
	service := NewInboxService(inboxRepo, ticketRepo, claimRepo, activityRepo)

	// Send message
	result, err := service.Send(ticket.ID, models.MessageTypeQuestion, "Need help!", "")
	require.NoError(t, err)

	assert.True(t, result.ClaimReleased)

	// Verify claim was released
	activeClaim, err = claimRepo.GetActiveByTicketID(ticket.ID)
	require.NoError(t, err)
	assert.Nil(t, activeClaim, "Claim should have been released")

	// Verify released claim status
	releasedClaim, err := claimRepo.GetByID(claim.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ClaimStatusReleased, releasedClaim.Status)
}

func TestInboxService_Send_UsesClaimWorkerID(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	// Setup repos
	projectRepo := db.NewProjectRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create project
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create ticket
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Ticket with Claim",
		Status:    models.StatusInProgress,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create active claim with specific worker ID
	claim := models.NewClaim(ticket.ID, "worker-from-claim", 60*time.Minute)
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// Create InboxService
	service := NewInboxService(inboxRepo, ticketRepo, claimRepo, activityRepo)

	// Send message without specifying worker ID
	result, err := service.Send(ticket.ID, models.MessageTypeQuestion, "Question", "")
	require.NoError(t, err)

	// Verify the message has worker ID from claim
	assert.Equal(t, "worker-from-claim", result.Message.FromAgent)
}
