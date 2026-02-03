package db

import (
	"testing"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoReleaseExpiredClaims_NoExpiredClaims(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTicketRepo(db)

	// No claims exist - should return 0
	released, err := repo.AutoReleaseExpiredClaims()
	require.NoError(t, err)
	assert.Equal(t, int64(0), released)
}

func TestAutoReleaseExpiredClaims_ReleasesExpiredClaim(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	// Create a ticket in in_progress status
	ticketRepo := NewTicketRepo(db)
	ticket := &models.Ticket{
		ProjectID:  projectID,
		Title:      "Test ticket with expired claim",
		Status:     models.StatusInProgress,
		MaxRetries: 3,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Create an expired claim
	claimRepo := NewClaimRepo(db)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "test-worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	// Run auto-release
	released, err := ticketRepo.AutoReleaseExpiredClaims()
	require.NoError(t, err)
	assert.Equal(t, int64(1), released)

	// Verify ticket is back to ready
	updated, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, updated.Status)
	assert.Equal(t, 1, updated.RetryCount)

	// Verify claim is expired
	updatedClaim, err := claimRepo.GetByID(claim.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ClaimStatusExpired, updatedClaim.Status)
	assert.NotNil(t, updatedClaim.ReleasedAt)
}

func TestAutoReleaseExpiredClaims_EscalatesToHumanOnMaxRetries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	// Create a ticket that's already at max retries - 1
	ticketRepo := NewTicketRepo(db)
	ticket := &models.Ticket{
		ProjectID:  projectID,
		Title:      "Test ticket at max retries",
		Status:     models.StatusInProgress,
		RetryCount: 2, // At 2, max is 3, so next failure escalates
		MaxRetries: 3,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Create an expired claim
	claimRepo := NewClaimRepo(db)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "test-worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	// Run auto-release
	released, err := ticketRepo.AutoReleaseExpiredClaims()
	require.NoError(t, err)
	assert.Equal(t, int64(1), released)

	// Verify ticket is escalated to human
	updated, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusHuman, updated.Status)
	assert.Equal(t, 3, updated.RetryCount)
	assert.Equal(t, "max_retries_exceeded", updated.HumanFlagReason)
}

func TestAutoReleaseExpiredClaims_IgnoresActiveClaims(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	// Create a ticket in in_progress status
	ticketRepo := NewTicketRepo(db)
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket with active claim",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Create a still-active claim (not expired)
	claimRepo := NewClaimRepo(db)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "test-worker",
		ClaimedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour), // Expires in 1 hour
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	// Run auto-release - should not affect active claims
	released, err := ticketRepo.AutoReleaseExpiredClaims()
	require.NoError(t, err)
	assert.Equal(t, int64(0), released)

	// Verify ticket is still in progress
	updated, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusInProgress, updated.Status)
}

func TestAutoReleaseExpiredClaims_IgnoresAlreadyExpiredClaims(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	// Create a ticket in ready status (already released previously)
	ticketRepo := NewTicketRepo(db)
	ticket := &models.Ticket{
		ProjectID:  projectID,
		Title:      "Test ticket already released",
		Status:     models.StatusReady,
		RetryCount: 1,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Create an already-expired claim (status = expired)
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO claims (ticket_id, worker_id, claimed_at, expires_at, released_at, status)
		VALUES (?, 'old-worker', ?, ?, ?, 'expired')
	`, ticket.ID, now.Add(-3*time.Hour), now.Add(-2*time.Hour), now.Add(-2*time.Hour))
	require.NoError(t, err)

	// Run auto-release - should not affect already-expired claims
	released, err := ticketRepo.AutoReleaseExpiredClaims()
	require.NoError(t, err)
	assert.Equal(t, int64(0), released)
}

func TestAutoReleaseExpiredClaims_MultipleExpiredClaims(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticketRepo := NewTicketRepo(db)
	claimRepo := NewClaimRepo(db)

	// Create 3 tickets with expired claims
	for i := 0; i < 3; i++ {
		ticket := &models.Ticket{
			ProjectID:  projectID,
			Title:      "Test ticket",
			Status:     models.StatusInProgress,
			MaxRetries: 3,
		}
		require.NoError(t, ticketRepo.Create(ticket))

		claim := &models.Claim{
			TicketID:  ticket.ID,
			WorkerID:  "test-worker",
			ClaimedAt: time.Now().Add(-2 * time.Hour),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			Status:    models.ClaimStatusActive,
		}
		require.NoError(t, claimRepo.Create(claim))
	}

	// Run auto-release
	released, err := ticketRepo.AutoReleaseExpiredClaims()
	require.NoError(t, err)
	assert.Equal(t, int64(3), released)

	// Verify all tickets are back to ready
	tickets, err := ticketRepo.List(TicketFilter{})
	require.NoError(t, err)
	for _, ticket := range tickets {
		assert.Equal(t, models.StatusReady, ticket.Status)
		assert.Equal(t, 1, ticket.RetryCount)
	}
}

func TestList_AutoReleasesExpiredClaims(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	// Create a ticket in in_progress status with expired claim
	ticketRepo := NewTicketRepo(db)
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	claimRepo := NewClaimRepo(db)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "test-worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	// Call List - should auto-release the expired claim
	tickets, err := ticketRepo.List(TicketFilter{})
	require.NoError(t, err)
	require.Len(t, tickets, 1)

	// Ticket should now be ready (auto-released by the List call)
	assert.Equal(t, models.StatusReady, tickets[0].Status)
}

func TestListWorkable_AutoReleasesAndIncludesTicket(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	// Create a ticket in in_progress status with expired claim
	ticketRepo := NewTicketRepo(db)
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	claimRepo := NewClaimRepo(db)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "test-worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	// ListWorkable should auto-release and then include the ticket
	// (since it's now ready with no unresolved dependencies)
	workable, err := ticketRepo.ListWorkable(TicketFilter{})
	require.NoError(t, err)
	require.Len(t, workable, 1)

	assert.Equal(t, models.StatusReady, workable[0].Status)
	assert.Equal(t, ticket.ID, workable[0].ID)
}

func TestListWorkable_DoesNotIncludeActiveClaimedTickets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	// Create two tickets:
	// 1. One with active (non-expired) claim - should NOT be workable
	// 2. One with expired claim - SHOULD be workable after auto-release
	ticketRepo := NewTicketRepo(db)
	claimRepo := NewClaimRepo(db)

	// Ticket 1: Active claim
	ticket1 := &models.Ticket{
		ProjectID: projectID,
		Title:     "Ticket with active claim",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket1))

	activeClaim := &models.Claim{
		TicketID:  ticket1.ID,
		WorkerID:  "worker1",
		ClaimedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(activeClaim))

	// Ticket 2: Expired claim
	ticket2 := &models.Ticket{
		ProjectID: projectID,
		Title:     "Ticket with expired claim",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket2))

	expiredClaim := &models.Claim{
		TicketID:  ticket2.ID,
		WorkerID:  "worker2",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(expiredClaim))

	// ListWorkable should only return ticket2 (the one with expired claim)
	workable, err := ticketRepo.ListWorkable(TicketFilter{})
	require.NoError(t, err)
	require.Len(t, workable, 1)
	assert.Equal(t, ticket2.ID, workable[0].ID)
}

func TestAutoReleaseExpiredClaims_LogsActivity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	// Create a ticket with expired claim
	ticketRepo := NewTicketRepo(db)
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	claimRepo := NewClaimRepo(db)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "test-worker",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Status:    models.ClaimStatusActive,
	}
	require.NoError(t, claimRepo.Create(claim))

	// Run auto-release
	_, err := ticketRepo.AutoReleaseExpiredClaims()
	require.NoError(t, err)

	// Verify activity was logged
	activityRepo := NewActivityRepo(db)
	logs, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)

	// Find the expired action
	found := false
	for _, log := range logs {
		if log.Action == models.ActionExpired {
			found = true
			assert.Equal(t, models.ActorTypeSystem, log.ActorType)
			assert.Contains(t, log.Summary, "auto-expired")
		}
	}
	assert.True(t, found, "expected to find 'expired' activity log")
}
