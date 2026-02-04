package cli

import (
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimListActive(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	// Create tickets and claims
	for i := 0; i < 3; i++ {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Test Ticket",
			Status:    models.StatusInProgress,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		claim := models.NewClaim(ticket.ID, "worker-"+string(rune('A'+i)), time.Hour)
		err = claimRepo.Create(claim)
		require.NoError(t, err)
	}

	// List active claims
	claims, err := claimRepo.ListActive()
	require.NoError(t, err)
	assert.Len(t, claims, 3)
}

func TestClaimExpired(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusInProgress,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create an already-expired claim (negative duration)
	claimRepo := db.NewClaimRepo(database.DB)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "worker-1",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		Status:    models.ClaimStatusActive,
	}
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// List expired
	expired, err := claimRepo.ListExpired()
	require.NoError(t, err)
	assert.Len(t, expired, 1)
	assert.Equal(t, claim.ID, expired[0].ID)
}

func TestClaimExpireAll(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)

	// Create expired claims
	for i := 0; i < 3; i++ {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Test Ticket",
			Status:    models.StatusInProgress,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		claim := &models.Claim{
			TicketID:  ticket.ID,
			WorkerID:  "worker",
			ClaimedAt: time.Now().Add(-2 * time.Hour),
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
			Status:    models.ClaimStatusActive,
		}
		err = claimRepo.Create(claim)
		require.NoError(t, err)
	}

	// Expire all
	count, err := claimRepo.ExpireAll()
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Verify no more expired
	expired, err := claimRepo.ListExpired()
	require.NoError(t, err)
	assert.Len(t, expired, 0)
}

func TestClaimRelease(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusInProgress,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	claimRepo := db.NewClaimRepo(database.DB)
	claim := models.NewClaim(ticket.ID, "worker-1", time.Hour)
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// Release claim
	err = claimRepo.Release(claim.ID, models.ClaimStatusReleased)
	require.NoError(t, err)

	// Verify claim status
	updated, err := claimRepo.GetByID(claim.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ClaimStatusReleased, updated.Status)
	assert.NotNil(t, updated.ReleasedAt)
}

func TestClaimHasActive(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test Ticket"}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	claimRepo := db.NewClaimRepo(database.DB)

	// No claim yet
	hasClaim, err := claimRepo.HasActiveClaim(ticket.ID)
	require.NoError(t, err)
	assert.False(t, hasClaim)

	// Create claim
	claim := models.NewClaim(ticket.ID, "worker-1", time.Hour)
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// Now has claim
	hasClaim, err = claimRepo.HasActiveClaim(ticket.ID)
	require.NoError(t, err)
	assert.True(t, hasClaim)

	// Release claim
	err = claimRepo.Release(claim.ID, models.ClaimStatusReleased)
	require.NoError(t, err)

	// No longer has active claim
	hasClaim, err = claimRepo.HasActiveClaim(ticket.ID)
	require.NoError(t, err)
	assert.False(t, hasClaim)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{0, "0m"},
		{5, "5m"},
		{30, "30m"},
		{60, "1h"},
		{90, "1h30m"},
		{120, "2h"},
		{125, "2h5m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatDuration(tt.minutes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatDurationTime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{0, "0m"},
		{5 * time.Minute, "5m"},
		{30 * time.Minute, "30m"},
		{time.Hour, "1h"},
		{90 * time.Minute, "1h30m"},
		{2 * time.Hour, "2h"},
		{125 * time.Minute, "2h5m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatDurationTime(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaimStatuses(t *testing.T) {
	// Verify all claim statuses
	statuses := []models.ClaimStatus{
		models.ClaimStatusActive,
		models.ClaimStatusCompleted,
		models.ClaimStatusExpired,
		models.ClaimStatusReleased,
	}

	for _, s := range statuses {
		assert.True(t, s.IsValid(), "claim status %s should be valid", s)
	}

	// Invalid status
	assert.False(t, models.ClaimStatus("invalid").IsValid())

	// Check terminal
	assert.False(t, models.ClaimStatusActive.IsTerminal())
	assert.True(t, models.ClaimStatusCompleted.IsTerminal())
	assert.True(t, models.ClaimStatusExpired.IsTerminal())
	assert.True(t, models.ClaimStatusReleased.IsTerminal())
}

func TestClaimTimeRemaining(t *testing.T) {
	// Active claim with time remaining
	claim := &models.Claim{
		ExpiresAt: time.Now().Add(30 * time.Minute),
		Status:    models.ClaimStatusActive,
	}
	remaining := claim.TimeRemaining()
	assert.True(t, remaining > 29*time.Minute)
	assert.True(t, remaining <= 30*time.Minute)

	// Expired claim
	claim.ExpiresAt = time.Now().Add(-10 * time.Minute)
	remaining = claim.TimeRemaining()
	assert.Equal(t, time.Duration(0), remaining)

	// Released claim
	claim.Status = models.ClaimStatusReleased
	claim.ExpiresAt = time.Now().Add(30 * time.Minute)
	remaining = claim.TimeRemaining()
	assert.Equal(t, time.Duration(0), remaining)
}

// TestClaimRaceConditionPrevented verifies that the unique index prevents concurrent claims
// at the database level. This is the real safety net - the CLI pre-check is for UX.
func TestClaimRaceConditionPrevented(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project and ticket
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "RACE", Name: "Race Condition Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Race Condition",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	claimRepo := db.NewClaimRepo(database.DB)

	// First agent claims the ticket
	claim1 := models.NewClaim(ticket.ID, "agent-1", time.Hour)
	err = claimRepo.Create(claim1)
	require.NoError(t, err)

	// Second agent tries to claim the same ticket - should fail due to unique index
	claim2 := models.NewClaim(ticket.ID, "agent-2", time.Hour)
	err = claimRepo.Create(claim2)

	// The unique index should prevent this
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UNIQUE constraint failed")

	// Verify only one active claim exists
	claims, err := claimRepo.ListByTicketID(ticket.ID)
	require.NoError(t, err)

	activeCount := 0
	for _, c := range claims {
		if c.Status == models.ClaimStatusActive {
			activeCount++
		}
	}
	assert.Equal(t, 1, activeCount, "only one active claim should exist")
}

// TestClaimReleaseActivityLog verifies that releasing a claim logs an activity entry
func TestClaimReleaseActivityLog(t *testing.T) {
	database, dbPath, cleanup := testDBWithPath(t)
	defer cleanup()

	// Setup project and ticket
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "ACTLOG", Name: "Activity Log Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Activity Logging",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Claim the ticket via CLI (which logs activity)
	_, err = runCmd(t, dbPath, "ticket", "claim", "ACTLOG-1", "--worker-id", "test-worker")
	require.NoError(t, err)

	// Verify claim activity was logged
	activityRepo := db.NewActivityRepo(database.DB)
	logs, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)

	claimFound := false
	for _, log := range logs {
		if log.Action == models.ActionClaimed {
			claimFound = true
			assert.Equal(t, "test-worker", log.ActorID)
			assert.Equal(t, models.ActorTypeAgent, log.ActorType)
		}
	}
	assert.True(t, claimFound, "claim activity should be logged")

	// Release the ticket via CLI
	_, err = runCmd(t, dbPath, "ticket", "release", "ACTLOG-1", "--reason", "Testing release logging")
	require.NoError(t, err)

	// Verify release activity was logged
	logs, err = activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)

	releaseFound := false
	for _, log := range logs {
		if log.Action == models.ActionReleased {
			releaseFound = true
			assert.Equal(t, "test-worker", log.ActorID)
			assert.Equal(t, models.ActorTypeAgent, log.ActorType)
			assert.Contains(t, log.Summary, "Testing release logging")
		}
	}
	assert.True(t, releaseFound, "release activity should be logged")
}
