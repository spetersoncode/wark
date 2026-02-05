package cli

import (
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusOverview(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	// Create tickets with various statuses
	ticketData := []struct {
		title  string
		status models.Status
	}{
		{"Ready 1", models.StatusReady},
		{"Ready 2", models.StatusReady},
		{"Working", models.StatusWorking},
		{"Blocked", models.StatusBlocked},
		{"Human 1", models.StatusHuman},
		{"Human 2", models.StatusHuman},
		{"Closed", models.StatusClosed},
	}

	completedRes := models.ResolutionCompleted
	var inProgressTicket *models.Ticket
	for _, td := range ticketData {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     td.title,
			Status:    td.status,
		}
		// Closed tickets require a resolution
		if td.status == models.StatusClosed {
			ticket.Resolution = &completedRes
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		if td.status == models.StatusWorking {
			inProgressTicket = ticket
		}
	}

	// Create inbox messages
	for i := 0; i < 3; i++ {
		msg := models.NewInboxMessage(inProgressTicket.ID, models.MessageTypeQuestion, "Question", "agent")
		err := inboxRepo.Create(msg)
		require.NoError(t, err)
	}

	// Create a claim expiring soon
	claim := &models.Claim{
		TicketID:  inProgressTicket.ID,
		WorkerID:  "worker-1",
		ClaimedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute), // Expires in 15 minutes
		Status:    models.ClaimStatusActive,
	}
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// Verify counts via repos
	workable, err := ticketRepo.ListWorkable(db.TicketFilter{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(workable))

	inProgressStatus := models.StatusWorking
	inProgress, err := ticketRepo.List(db.TicketFilter{Status: &inProgressStatus})
	require.NoError(t, err)
	assert.Equal(t, 1, len(inProgress))

	blockedStatus := models.StatusBlocked
	blocked, err := ticketRepo.List(db.TicketFilter{Status: &blockedStatus})
	require.NoError(t, err)
	assert.Equal(t, 1, len(blocked))

	humanStatus := models.StatusHuman
	humanTickets, err := ticketRepo.List(db.TicketFilter{Status: &humanStatus})
	require.NoError(t, err)
	assert.Equal(t, 2, len(humanTickets))

	pendingCount, err := inboxRepo.CountPending()
	require.NoError(t, err)
	assert.Equal(t, 3, pendingCount)

	activeClaims, err := claimRepo.ListActive()
	require.NoError(t, err)
	assert.Equal(t, 1, len(activeClaims))
}

func TestStatusByProject(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create multiple projects
	projectRepo := db.NewProjectRepo(database.DB)
	project1 := &models.Project{Key: "PROJ1", Name: "Project 1"}
	project2 := &models.Project{Key: "PROJ2", Name: "Project 2"}
	err := projectRepo.Create(project1)
	require.NoError(t, err)
	err = projectRepo.Create(project2)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create tickets in project 1
	for i := 0; i < 3; i++ {
		ticket := &models.Ticket{
			ProjectID: project1.ID,
			Title:     "Project 1 Ticket",
			Status:    models.StatusReady,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// Create tickets in project 2
	for i := 0; i < 5; i++ {
		ticket := &models.Ticket{
			ProjectID: project2.ID,
			Title:     "Project 2 Ticket",
			Status:    models.StatusReady,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// Count for project 1
	proj1Workable, err := ticketRepo.ListWorkable(db.TicketFilter{ProjectKey: "PROJ1"})
	require.NoError(t, err)
	assert.Equal(t, 3, len(proj1Workable))

	// Count for project 2
	proj2Workable, err := ticketRepo.ListWorkable(db.TicketFilter{ProjectKey: "PROJ2"})
	require.NoError(t, err)
	assert.Equal(t, 5, len(proj2Workable))

	// Count all
	allWorkable, err := ticketRepo.ListWorkable(db.TicketFilter{})
	require.NoError(t, err)
	assert.Equal(t, 8, len(allWorkable))
}

func TestStatusResultStruct(t *testing.T) {
	result := StatusResult{
		Workable:     5,
		Working:   2,
		Review:       1,
		BlockedDeps:  3,
		BlockedHuman: 1,
		PendingInbox: 4,
		ExpiringSoon: []*ExpiringSoon{
			{TicketKey: "TEST-1", WorkerID: "worker-1", MinutesLeft: 15},
		},
		RecentActivity: []*ActivitySummary{
			{TicketKey: "TEST-2", Action: "completed", Age: "5m ago", Summary: "Work done"},
		},
		Project: "TEST",
	}

	assert.Equal(t, 5, result.Workable)
	assert.Equal(t, 2, result.Working)
	assert.Equal(t, 1, result.Review)
	assert.Equal(t, 3, result.BlockedDeps)
	assert.Equal(t, 1, result.BlockedHuman)
	assert.Equal(t, 4, result.PendingInbox)
	assert.Len(t, result.ExpiringSoon, 1)
	assert.Len(t, result.RecentActivity, 1)
	assert.Equal(t, "TEST", result.Project)
}

func TestExpiringSoonDetection(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	// Create ticket and claim expiring in 20 minutes (within 30 minute threshold)
	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 1", Status: models.StatusWorking}
	err = ticketRepo.Create(ticket1)
	require.NoError(t, err)

	claim1 := &models.Claim{
		TicketID:  ticket1.ID,
		WorkerID:  "worker-1",
		ClaimedAt: time.Now(),
		ExpiresAt: time.Now().Add(20 * time.Minute),
		Status:    models.ClaimStatusActive,
	}
	err = claimRepo.Create(claim1)
	require.NoError(t, err)

	// Create ticket and claim expiring in 60 minutes (not within threshold)
	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 2", Status: models.StatusWorking}
	err = ticketRepo.Create(ticket2)
	require.NoError(t, err)

	claim2 := &models.Claim{
		TicketID:  ticket2.ID,
		WorkerID:  "worker-2",
		ClaimedAt: time.Now(),
		ExpiresAt: time.Now().Add(60 * time.Minute),
		Status:    models.ClaimStatusActive,
	}
	err = claimRepo.Create(claim2)
	require.NoError(t, err)

	// Get active claims and check which are expiring soon using model methods
	// (SQL-calculated MinutesRemaining may have timezone issues with julianday)
	claims, err := claimRepo.ListActive()
	require.NoError(t, err)
	assert.Len(t, claims, 2)

	// Use the model's TimeRemaining method for reliable calculation
	expiringSoonCount := 0
	for _, claim := range claims {
		remaining := claim.TimeRemaining()
		if remaining > 0 && remaining <= 30*time.Minute {
			expiringSoonCount++
		}
	}
	assert.Equal(t, 1, expiringSoonCount)
}
