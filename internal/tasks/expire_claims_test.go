package tasks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDB creates a temporary database for testing
func testDB(t *testing.T) (*db.DB, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)

	err = database.Migrate()
	require.NoError(t, err)

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

func TestClaimExpirer_ExpireAll_NoExpired(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	expirer := NewClaimExpirer(database.DB)

	// No claims at all
	result, err := expirer.ExpireAll(false)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
	assert.Equal(t, 0, result.Expired)
}

func TestClaimExpirer_ExpireAll_WithExpired(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project and tickets
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	// Create 3 tickets with expired claims
	for i := 0; i < 3; i++ {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Title:      "Test Ticket",
			Status:     models.StatusInProgress,
			MaxRetries: 3,
		}
		require.NoError(t, ticketRepo.Create(ticket))

		claim := &models.Claim{
			TicketID:  ticket.ID,
			WorkerID:  "worker",
			ClaimedAt: time.Now().Add(-2 * time.Hour),
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
			Status:    models.ClaimStatusActive,
		}
		require.NoError(t, claimRepo.Create(claim))
	}

	expirer := NewClaimExpirer(database.DB)
	result, err := expirer.ExpireAll(false)
	require.NoError(t, err)

	assert.Equal(t, 3, result.Processed)
	assert.Equal(t, 3, result.Expired)
	assert.Equal(t, 0, result.Escalated)
	assert.Equal(t, 0, result.Errors)
	assert.False(t, result.DryRun)

	// Verify tickets were updated
	tickets, err := ticketRepo.List(db.TicketFilter{})
	require.NoError(t, err)
	for _, ticket := range tickets {
		assert.Equal(t, models.StatusReady, ticket.Status)
		assert.Equal(t, 1, ticket.RetryCount)
	}
}

func TestClaimExpirer_ExpireAll_MaxRetriesEscalation(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	// Create ticket at max retries
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Title:      "Test Ticket",
		Status:     models.StatusInProgress,
		RetryCount: 2, // Already at 2, max is 3
		MaxRetries: 3,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	expirer := NewClaimExpirer(database.DB)
	result, err := expirer.ExpireAll(false)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 1, result.Expired)
	assert.Equal(t, 1, result.Escalated)

	// Verify ticket was escalated
	updated, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusHuman, updated.Status)
	assert.Equal(t, 3, updated.RetryCount)
	assert.Equal(t, "max_retries_exceeded", updated.HumanFlagReason)
}

func TestClaimExpirer_ExpireAll_DryRun(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	expirer := NewClaimExpirer(database.DB)
	result, err := expirer.ExpireAll(true) // dry run
	require.NoError(t, err)

	assert.True(t, result.DryRun)
	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 1, result.Expired)

	// Verify nothing was actually changed
	updated, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusInProgress, updated.Status)
	assert.Equal(t, 0, updated.RetryCount)

	updatedClaim, err := claimRepo.GetByID(claim.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ClaimStatusActive, updatedClaim.Status)
}

func TestClaimExpirer_ExpireTicket(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	claim := models.NewClaim(ticket.ID, "worker", time.Hour)
	require.NoError(t, claimRepo.Create(claim))

	expirer := NewClaimExpirer(database.DB)
	result, err := expirer.ExpireTicket(ticket.ID, false)
	require.NoError(t, err)

	assert.Equal(t, ticket.ID, result.TicketID)
	assert.Equal(t, "worker", result.WorkerID)
	assert.Equal(t, "ready", result.NewStatus)
	assert.Equal(t, 1, result.RetryCount)
	assert.False(t, result.Escalated)

	// Verify ticket was updated
	updated, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, updated.Status)
}

func TestClaimExpirer_ExpireTicket_NoClaim(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusReady,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	expirer := NewClaimExpirer(database.DB)
	_, err := expirer.ExpireTicket(ticket.ID, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active claim")
}

func TestClaimExpirer_SkipsNonInProgressTickets(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	// Create ticket that's already closed
	completedRes := models.ResolutionCompleted
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Title:      "Test Ticket",
		Status:     models.StatusClosed, // Not in_progress
		Resolution: &completedRes,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Create an "expired" claim (even though ticket is done)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	expirer := NewClaimExpirer(database.DB)
	result, err := expirer.ExpireAll(false)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 0, result.Expired) // Not expired because ticket isn't in_progress
	assert.Equal(t, 1, result.Errors)  // Should count as error
}

func TestClaimExpirer_RunDaemon_ImmediateCancel(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	expirer := NewClaimExpirer(database.DB)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := expirer.RunDaemon(ctx, time.Hour, nil)
	assert.Equal(t, context.Canceled, err)
}

func TestClaimExpirer_RunDaemon_WithCallback(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	expirer := NewClaimExpirer(database.DB)

	ctx, cancel := context.WithCancel(context.Background())

	callbackCalled := false
	callback := func(result *ExpireClaimsResult) {
		callbackCalled = true
		cancel() // Cancel after first callback
	}

	err := expirer.RunDaemon(ctx, 10*time.Millisecond, callback)
	assert.Equal(t, context.Canceled, err)
	assert.True(t, callbackCalled)
}

// TestClaimExpirer_ActivityLogging verifies that claim expiration logs activity entries
func TestClaimExpirer_ActivityLogging(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "EXPLOG", Name: "Expiration Logging Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create ticket with expired claim
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Title:      "Test Expiration Logging",
		Status:     models.StatusInProgress,
		MaxRetries: 3,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "expire-test-worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	// Run expiration
	expirer := NewClaimExpirer(database.DB)
	result, err := expirer.ExpireAll(false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Expired)

	// Verify activity was logged
	logs, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)

	expireFound := false
	for _, log := range logs {
		if log.Action == models.ActionExpired {
			expireFound = true
			assert.Equal(t, models.ActorTypeSystem, log.ActorType)
			assert.Contains(t, log.Summary, "expired")
		}
	}
	assert.True(t, expireFound, "expiration activity should be logged")
}

// TestClaimExpirer_ActivityLogging_Escalation verifies that escalation logs activity entries
func TestClaimExpirer_ActivityLogging_Escalation(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "ESCLOG", Name: "Escalation Logging Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Create ticket at max retries - will escalate on expiration
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Title:      "Test Escalation Logging",
		Status:     models.StatusInProgress,
		RetryCount: 2, // At 2, max is 3, so next expiration escalates
		MaxRetries: 3,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "escalate-test-worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	// Run expiration
	expirer := NewClaimExpirer(database.DB)
	result, err := expirer.ExpireAll(false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Escalated)

	// Verify activity was logged with escalation info
	logs, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)

	expireFound := false
	for _, log := range logs {
		if log.Action == models.ActionExpired {
			expireFound = true
			assert.Equal(t, models.ActorTypeSystem, log.ActorType)
			assert.Contains(t, log.Summary, "escalated")
			assert.Contains(t, log.Summary, "human")
		}
	}
	assert.True(t, expireFound, "expiration with escalation activity should be logged")
}
