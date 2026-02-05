package service

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/common"
	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDB creates a temporary test database and returns cleanup function.
func testDB(t *testing.T) (*db.DB, string, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	database, err := db.Open(dbPath)
	require.NoError(t, err)

	err = database.Migrate()
	require.NoError(t, err)

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, dbPath, cleanup
}

func TestStatusService_GetSummary(t *testing.T) {
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

	// Create tickets with various statuses
	completedRes := models.ResolutionCompleted
	ticketData := []struct {
		title  string
		status models.Status
	}{
		{"Ready 1", models.StatusReady},
		{"Ready 2", models.StatusReady},
		{"Working", models.StatusWorking},
		{"Review 1", models.StatusReview},
		{"Blocked", models.StatusBlocked},
		{"Human 1", models.StatusHuman},
		{"Human 2", models.StatusHuman},
		{"Closed", models.StatusClosed},
	}

	var inProgressTicket *models.Ticket
	for _, td := range ticketData {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     td.title,
			Status:    td.status,
		}
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
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Status:    models.ClaimStatusActive,
	}
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// Create activity
	err = activityRepo.LogAction(inProgressTicket.ID, models.ActionClaimed, models.ActorTypeAgent, "agent-1", "Agent claimed ticket")
	require.NoError(t, err)

	// Test the service
	statusService := NewStatusService(ticketRepo, inboxRepo, claimRepo, activityRepo)
	summary, err := statusService.GetSummary("")
	require.NoError(t, err)

	assert.Equal(t, 2, summary.Workable)
	assert.Equal(t, 1, summary.Working)
	assert.Equal(t, 1, summary.Review)
	assert.Equal(t, 1, summary.BlockedDeps)
	assert.Equal(t, 2, summary.BlockedHuman)
	assert.Equal(t, 3, summary.PendingInbox)
	assert.Len(t, summary.ExpiringSoon, 1)
	// RecentActivity includes ticket creations from the trigger + our explicit claim log
	// The activity log limit is 5, so we should have 5 entries (limit)
	assert.Len(t, summary.RecentActivity, 5)
	// Most recent should be the claimed action (added last)
	assert.Equal(t, "claimed", summary.RecentActivity[0].Action)
}

func TestStatusService_GetSummary_ByProject(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	// Setup repos
	projectRepo := db.NewProjectRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create projects
	project1 := &models.Project{Key: "PROJ1", Name: "Project 1"}
	project2 := &models.Project{Key: "PROJ2", Name: "Project 2"}
	err := projectRepo.Create(project1)
	require.NoError(t, err)
	err = projectRepo.Create(project2)
	require.NoError(t, err)

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

	// Test the service
	statusService := NewStatusService(ticketRepo, inboxRepo, claimRepo, activityRepo)

	// Check project 1
	summary1, err := statusService.GetSummary("PROJ1")
	require.NoError(t, err)
	assert.Equal(t, 3, summary1.Workable)
	assert.Equal(t, "PROJ1", summary1.ProjectKey)

	// Check project 2
	summary2, err := statusService.GetSummary("proj2") // Test case insensitivity
	require.NoError(t, err)
	assert.Equal(t, 5, summary2.Workable)
	assert.Equal(t, "PROJ2", summary2.ProjectKey)

	// Check all projects
	summaryAll, err := statusService.GetSummary("")
	require.NoError(t, err)
	assert.Equal(t, 8, summaryAll.Workable)
}

func TestStatusService_ExpiringSoon(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	// Setup repos
	projectRepo := db.NewProjectRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create project
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

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

	// Test the service
	statusService := NewStatusService(ticketRepo, inboxRepo, claimRepo, activityRepo)
	summary, err := statusService.GetSummary("")
	require.NoError(t, err)

	assert.Len(t, summary.ExpiringSoon, 1)
	assert.Equal(t, "worker-1", summary.ExpiringSoon[0].WorkerID)
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"just now", time.Now().Add(-30 * time.Second), "just now"},
		{"minutes ago", time.Now().Add(-5 * time.Minute), "5m ago"},
		{"hours ago", time.Now().Add(-3 * time.Hour), "3h ago"},
		{"days ago", time.Now().Add(-2 * 24 * time.Hour), "2d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.FormatAge(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusService_EmptyDatabase(t *testing.T) {
	database, _, cleanup := testDB(t)
	defer cleanup()

	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	statusService := NewStatusService(ticketRepo, inboxRepo, claimRepo, activityRepo)
	summary, err := statusService.GetSummary("")
	require.NoError(t, err)

	assert.Equal(t, 0, summary.Workable)
	assert.Equal(t, 0, summary.Working)
	assert.Equal(t, 0, summary.Review)
	assert.Equal(t, 0, summary.BlockedDeps)
	assert.Equal(t, 0, summary.BlockedHuman)
	assert.Equal(t, 0, summary.PendingInbox)
	assert.Empty(t, summary.ExpiringSoon)
	assert.Empty(t, summary.RecentActivity)
}

// TestNewStatusService verifies the constructor.
func TestNewStatusService(t *testing.T) {
	// Just verify that NewStatusService doesn't panic with nil
	// In real usage, the repos would not be nil
	var ticketRepo *db.TicketRepo
	var inboxRepo *db.InboxRepo
	var claimRepo *db.ClaimRepo
	var activityRepo *db.ActivityRepo

	service := NewStatusService(ticketRepo, inboxRepo, claimRepo, activityRepo)
	assert.NotNil(t, service)
}

// Helper to get a database handle for parallel tests
func getTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	database, _, cleanup := testDB(t)
	return database.DB, cleanup
}
