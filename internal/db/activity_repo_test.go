package db

import (
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityRepo_SortOrder(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	// Create test project and ticket
	projectRepo := NewProjectRepo(db.DB)
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := NewTicketRepo(db.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusReady,
		Priority:  models.PriorityMedium,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	activityRepo := NewActivityRepo(db.DB)

	// Create activities in specific order with same timestamp
	now := time.Now()

	// Create multiple activities with the same timestamp
	// We use explicit IDs by inserting directly since the auto-increment will give us sequential IDs
	activities := []*models.ActivityLog{
		{
			TicketID:  ticket.ID,
			Action:    models.ActionCreated,
			ActorType: models.ActorTypeSystem,
			Summary:   "First activity",
			CreatedAt: now,
		},
		{
			TicketID:  ticket.ID,
			Action:    models.ActionComment,
			ActorType: models.ActorTypeAgent,
			ActorID:   "agent-1",
			Summary:   "Second activity",
			CreatedAt: now,
		},
		{
			TicketID:  ticket.ID,
			Action:    models.ActionClaimed,
			ActorType: models.ActorTypeAgent,
			ActorID:   "agent-2",
			Summary:   "Third activity",
			CreatedAt: now,
		},
	}

	for _, activity := range activities {
		err := activityRepo.Create(activity)
		require.NoError(t, err)
	}

	// List activities - should be sorted by created_at DESC, then id DESC
	// The trigger creates a "created" activity, so we have 4 total (1 from trigger + 3 we created)
	results, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)
	require.Len(t, results, 4)

	// Verify descending order (newest first)
	// Since all have the same timestamp, they should be ordered by ID descending
	// The trigger-created activity has the earliest ID, so it should be last
	assert.Equal(t, models.ActionClaimed, results[0].Action, "First result should be the claimed action (highest ID)")
	assert.Equal(t, models.ActionComment, results[1].Action, "Second result should be the comment action")
	assert.Equal(t, models.ActionCreated, results[2].Action, "Third result should be the created action (manual)")
	assert.Equal(t, models.ActionCreated, results[3].Action, "Fourth result should be the trigger-created activity")

	// Verify timestamps are descending
	for i := 0; i < len(results)-1; i++ {
		assert.True(t, results[i].CreatedAt.Equal(results[i+1].CreatedAt) || results[i].CreatedAt.After(results[i+1].CreatedAt),
			"Activity %d should have timestamp >= activity %d", i, i+1)
	}
}

func TestActivityRepo_ListByTicket_MostRecentFirst(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	// Create test project and ticket
	projectRepo := NewProjectRepo(db.DB)
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := NewTicketRepo(db.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusReady,
		Priority:  models.PriorityMedium,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	activityRepo := NewActivityRepo(db.DB)

	// Create activities with different timestamps
	baseTime := time.Now()

	// Create in chronological order: old -> new
	oldActivity := &models.ActivityLog{
		TicketID:  ticket.ID,
		Action:    models.ActionCreated,
		ActorType: models.ActorTypeSystem,
		Summary:   "Old activity",
		CreatedAt: baseTime.Add(-2 * time.Hour),
	}
	require.NoError(t, activityRepo.Create(oldActivity))

	middleActivity := &models.ActivityLog{
		TicketID:  ticket.ID,
		Action:    models.ActionComment,
		ActorType: models.ActorTypeAgent,
		ActorID:   "agent-1",
		Summary:   "Middle activity",
		CreatedAt: baseTime.Add(-1 * time.Hour),
	}
	require.NoError(t, activityRepo.Create(middleActivity))

	newActivity := &models.ActivityLog{
		TicketID:  ticket.ID,
		Action:    models.ActionClaimed,
		ActorType: models.ActorTypeAgent,
		ActorID:   "agent-2",
		Summary:   "New activity",
		CreatedAt: baseTime,
	}
	require.NoError(t, activityRepo.Create(newActivity))

	// List activities - should be sorted by created_at DESC (newest first)
	// The trigger creates a "created" activity with the earliest timestamp
	results, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)
	require.Len(t, results, 4)

	// Verify order: new -> middle -> old (trigger-created activity has earliest timestamp)
	assert.Equal(t, models.ActionClaimed, results[0].Action, "First result should be newest (claimed)")
	assert.Equal(t, models.ActionComment, results[1].Action, "Second result should be middle (comment)")
	assert.Equal(t, models.ActionCreated, results[2].Action, "Third result should be the manual created activity")
	assert.Equal(t, models.ActionCreated, results[3].Action, "Fourth result should be trigger-created activity (oldest)")

	// Verify timestamps are descending (or equal within same second)
	for i := 0; i < len(results)-1; i++ {
		// Allow for equal timestamps (within same second) or strictly after
		assert.True(t, results[i].CreatedAt.Equal(results[i+1].CreatedAt) || results[i].CreatedAt.After(results[i+1].CreatedAt),
			"Activity %d should have timestamp >= activity %d", i, i+1)
	}
}

func TestActivityRepo_GetLatestByTicket(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	// Create test project and ticket
	projectRepo := NewProjectRepo(db.DB)
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := NewTicketRepo(db.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusReady,
		Priority:  models.PriorityMedium,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	activityRepo := NewActivityRepo(db.DB)

	// Create activities with different timestamps
	baseTime := time.Now()

	// Create older activity first
	oldActivity := &models.ActivityLog{
		TicketID:  ticket.ID,
		Action:    models.ActionCreated,
		ActorType: models.ActorTypeSystem,
		Summary:   "Old activity",
		CreatedAt: baseTime.Add(-1 * time.Hour),
	}
	require.NoError(t, activityRepo.Create(oldActivity))

	// Create newer activity
	newActivity := &models.ActivityLog{
		TicketID:  ticket.ID,
		Action:    models.ActionClaimed,
		ActorType: models.ActorTypeAgent,
		ActorID:   "agent-1",
		Summary:   "New activity",
		CreatedAt: baseTime,
	}
	require.NoError(t, activityRepo.Create(newActivity))

	// Get latest - should return the claimed activity (not the trigger-created one)
	latest, err := activityRepo.GetLatestByTicket(ticket.ID)
	require.NoError(t, err)
	require.NotNil(t, latest)

	assert.Equal(t, models.ActionClaimed, latest.Action)
	assert.Equal(t, "New activity", latest.Summary)
}
