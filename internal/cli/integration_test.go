package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spetersoncode/wark/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Full Workflow Integration Tests
// =============================================================================

// TestFullWorkflowInitToAccept tests the complete CLI workflow:
// init → project create → ticket create → claim → complete → accept
func TestFullWorkflowInitToAccept(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// 1. Database is already initialized by testDB, verify it works
	status, err := database.MigrationStatus()
	require.NoError(t, err)
	assert.Greater(t, status, int64(0), "migration should have run")

	// 2. Create a project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{
		Key:         "WEBAPP",
		Name:        "Web Application",
		Description: "Main web app project",
	}
	err = projectRepo.Create(project)
	require.NoError(t, err)
	assert.Equal(t, int64(1), project.ID)
	assert.Equal(t, "WEBAPP", project.Key)

	// 3. Create a ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:   project.ID,
		Title:       "Implement user login",
		Description: "Add OAuth2 login support",
		Priority:    models.PriorityHigh,
		Complexity:  models.ComplexityMedium,
		Status:      models.StatusReady, // Set to ready so it can be claimed
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)
	assert.Equal(t, 1, ticket.Number)
	assert.Equal(t, models.StatusReady, ticket.Status)

	// Generate ticket key
	ticket.TicketKey = "WEBAPP-1"
	ticket.ProjectKey = "WEBAPP"

	// 4. Claim the ticket
	claimRepo := db.NewClaimRepo(database.DB)
	workerID := "test-agent-001"
	duration := 60 * time.Minute
	claim := models.NewClaim(ticket.ID, workerID, duration)
	err = claimRepo.Create(claim)
	require.NoError(t, err)
	assert.Equal(t, models.ClaimStatusActive, claim.Status)

	// Update ticket status to in_progress
	ticket.Status = models.StatusInProgress
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	err = activityRepo.LogAction(ticket.ID, models.ActionClaimed, models.ActorTypeAgent, workerID, "Claimed for work")
	require.NoError(t, err)

	// 5. Complete the ticket
	err = claimRepo.Release(claim.ID, models.ClaimStatusCompleted)
	require.NoError(t, err)

	ticket.Status = models.StatusReview
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	err = activityRepo.LogAction(ticket.ID, models.ActionCompleted, models.ActorTypeAgent, workerID, "Implementation done")
	require.NoError(t, err)

	// 6. Accept the ticket
	ticket.Status = models.StatusClosed
	completedRes := models.ResolutionCompleted
	ticket.Resolution = &completedRes
	now := time.Now()
	ticket.CompletedAt = &now
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	err = activityRepo.LogAction(ticket.ID, models.ActionAccepted, models.ActorTypeHuman, "", "Work accepted")
	require.NoError(t, err)

	// Verify final state
	finalTicket, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusClosed, finalTicket.Status)
	assert.NotNil(t, finalTicket.CompletedAt)

	// Verify activity log
	logs, err := activityRepo.ListByTicket(ticket.ID, 100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logs), 3) // claimed, completed, accepted

	// Verify claim history
	claims, err := claimRepo.ListByTicketID(ticket.ID)
	require.NoError(t, err)
	assert.Len(t, claims, 1)
	assert.Equal(t, models.ClaimStatusCompleted, claims[0].Status)
}

// TestWorkflowWithRejection tests the workflow when a ticket is rejected
func TestWorkflowWithRejection(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project and ticket
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Title:      "Fix bug",
		Status:     models.StatusReady,
		MaxRetries: 3,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Claim the ticket
	claimRepo := db.NewClaimRepo(database.DB)
	claim := models.NewClaim(ticket.ID, "worker-1", time.Hour)
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	ticket.Status = models.StatusInProgress
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Complete the ticket
	err = claimRepo.Release(claim.ID, models.ClaimStatusCompleted)
	require.NoError(t, err)
	ticket.Status = models.StatusReview
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Reject the ticket - goes back to ready with incremented retry
	ticket.Status = models.StatusReady
	ticket.RetryCount = 1
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Verify state
	rejectedTicket, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, rejectedTicket.Status)
	assert.Equal(t, 1, rejectedTicket.RetryCount)
}

// TestWorkflowWithCancellation tests cancelling a ticket
func TestWorkflowWithCancellation(t *testing.T) {
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
		Title:     "Deprecated feature",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Close the ticket (cancel)
	assert.True(t, state.CanBeClosed(ticket.Status))
	ticket.Status = models.StatusClosed
	wontDoRes := models.ResolutionWontDo
	ticket.Resolution = &wontDoRes
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Verify
	closedTicket, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusClosed, closedTicket.Status)
	assert.True(t, closedTicket.Status.IsTerminal())
}

// TestWorkflowReopen tests reopening a done or cancelled ticket
func TestWorkflowReopen(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)

	// Test reopening a closed (completed) ticket
	t.Run("reopen completed ticket", func(t *testing.T) {
		completedRes := models.ResolutionCompleted
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Title:      "Completed ticket",
			Status:     models.StatusClosed,
			Resolution: &completedRes,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		assert.True(t, state.CanBeReopened(ticket.Status))
		ticket.Status = models.StatusReady
		ticket.Resolution = nil
		ticket.CompletedAt = nil
		err = ticketRepo.Update(ticket)
		require.NoError(t, err)

		reopened, err := ticketRepo.GetByID(ticket.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusReady, reopened.Status)
	})

	// Test reopening a closed (wont_do) ticket
	t.Run("reopen wont_do ticket", func(t *testing.T) {
		wontDoRes := models.ResolutionWontDo
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Title:      "WontDo ticket",
			Status:     models.StatusClosed,
			Resolution: &wontDoRes,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		assert.True(t, state.CanBeReopened(ticket.Status))
		ticket.Status = models.StatusReady
		ticket.Resolution = nil
		err = ticketRepo.Update(ticket)
		require.NoError(t, err)

		reopened, err := ticketRepo.GetByID(ticket.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusReady, reopened.Status)
	})
}

// =============================================================================
// Error Case Tests
// =============================================================================

// TestInvalidProjectKey tests validation of project keys
func TestInvalidProjectKey(t *testing.T) {
	tests := []struct {
		key     string
		wantErr bool
	}{
		{"VALID", false},
		{"AB", false},          // Min length
		{"ABCDEFGHIJ", false},  // Max length
		{"A1B2C3", false},      // Alphanumeric
		{"A", true},            // Too short
		{"ABCDEFGHIJK", true},  // Too long
		{"123ABC", true},       // Must start with letter
		{"abc", true},          // Must be uppercase (but ValidateProjectKey accepts lowercase and uppercases it)
		{"AB-CD", true},        // No special chars
		{"", true},             // Empty
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := models.ValidateProjectKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMissingProject tests creating a ticket with a non-existent project
func TestMissingProject(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)

	// Try to get a non-existent project
	project, err := projectRepo.GetByKey("NONEXISTENT")
	require.NoError(t, err) // GetByKey returns nil, not error
	assert.Nil(t, project)
}

// TestMissingTicket tests operations on non-existent tickets
func TestMissingTicket(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	ticketRepo := db.NewTicketRepo(database.DB)

	// Try to get a non-existent ticket
	ticket, err := ticketRepo.GetByID(999)
	require.NoError(t, err)
	assert.Nil(t, ticket)

	ticket, err = ticketRepo.GetByKey("TEST", 999)
	require.NoError(t, err)
	assert.Nil(t, ticket)
}

// TestDuplicateProject tests creating a project with a duplicate key
func TestDuplicateProject(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)

	// Create first project
	project1 := &models.Project{Key: "DUPE", Name: "First"}
	err := projectRepo.Create(project1)
	require.NoError(t, err)

	// Try to create duplicate
	project2 := &models.Project{Key: "DUPE", Name: "Second"}
	err = projectRepo.Create(project2)
	assert.Error(t, err) // Should fail with unique constraint
}

// TestStateTransitionErrors tests invalid state transitions
func TestStateTransitionErrors(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	machine := state.NewMachine()

	tests := []struct {
		name        string
		fromStatus  models.Status
		toStatus    models.Status
		shouldError bool
	}{
		{"blocked to in_progress", models.StatusBlocked, models.StatusInProgress, true},  // Must be ready first
		{"closed to in_progress", models.StatusClosed, models.StatusInProgress, true},    // Terminal state
		{"ready to in_progress", models.StatusReady, models.StatusInProgress, false},     // Valid
		{"in_progress to review", models.StatusInProgress, models.StatusReview, false},   // Valid
		{"review to closed", models.StatusReview, models.StatusClosed, false},            // Valid (accept)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket := &models.Ticket{
				ProjectID: project.ID,
				Title:     tt.name,
				Status:    tt.fromStatus,
			}
			// For closed status, we need a resolution to create a valid ticket
			if tt.fromStatus == models.StatusClosed {
				res := models.ResolutionCompleted
				ticket.Resolution = &res
			}
			err := ticketRepo.Create(ticket)
			require.NoError(t, err)

			// For transitions to closed, provide a resolution
			var resolution *models.Resolution
			if tt.toStatus == models.StatusClosed {
				res := models.ResolutionCompleted
				resolution = &res
			}
			err = machine.CanTransition(ticket, tt.toStatus, state.TransitionTypeManual, "", resolution)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestClaimAlreadyClaimed tests trying to claim an already claimed ticket
func TestClaimAlreadyClaimed(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test", Status: models.StatusReady}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// First claim
	claimRepo := db.NewClaimRepo(database.DB)
	claim1 := models.NewClaim(ticket.ID, "worker-1", time.Hour)
	err = claimRepo.Create(claim1)
	require.NoError(t, err)

	// Check if there's an active claim
	hasClaim, err := claimRepo.HasActiveClaim(ticket.ID)
	require.NoError(t, err)
	assert.True(t, hasClaim, "should have active claim")

	// Get the active claim to see who has it
	activeClaim, err := claimRepo.GetActiveByTicketID(ticket.ID)
	require.NoError(t, err)
	assert.NotNil(t, activeClaim)
	assert.Equal(t, "worker-1", activeClaim.WorkerID)
}

// TestClaimExpiredCanReClaim tests that an expired claim allows re-claiming
func TestClaimExpiredCanReClaim(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test", Status: models.StatusInProgress}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	claimRepo := db.NewClaimRepo(database.DB)

	// Create an expired claim
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "worker-1",
		ClaimedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		Status:    models.ClaimStatusActive,
	}
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// Expire the claim
	_, err = claimRepo.ExpireAll()
	require.NoError(t, err)

	// Now should be able to claim again
	hasClaim, err := claimRepo.HasActiveClaim(ticket.ID)
	require.NoError(t, err)
	assert.False(t, hasClaim, "should not have active claim after expiry")

	// New claim should work
	claim2 := models.NewClaim(ticket.ID, "worker-2", time.Hour)
	err = claimRepo.Create(claim2)
	require.NoError(t, err)
	assert.Equal(t, models.ClaimStatusActive, claim2.Status)
}

// TestInvalidEnumValues tests validation of invalid enum values
func TestInvalidEnumValues(t *testing.T) {
	// Test invalid priority
	assert.False(t, models.Priority("invalid").IsValid())
	assert.False(t, models.Priority("").IsValid())

	// Test invalid complexity
	assert.False(t, models.Complexity("invalid").IsValid())
	assert.False(t, models.Complexity("").IsValid())

	// Test invalid status
	assert.False(t, models.Status("invalid").IsValid())
	assert.False(t, models.Status("").IsValid())

	// Test invalid message type
	assert.False(t, models.MessageType("invalid").IsValid())

	// Test invalid claim status
	assert.False(t, models.ClaimStatus("invalid").IsValid())
}

// =============================================================================
// JSON Output Tests
// =============================================================================

// TestProjectJSONOutput tests JSON serialization of projects
func TestProjectJSONOutput(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{
		Key:         "TEST",
		Name:        "Test Project",
		Description: "Description",
	}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Marshal to JSON
	data, err := json.Marshal(project)
	require.NoError(t, err)

	// Verify JSON fields
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "TEST", parsed["key"])
	assert.Equal(t, "Test Project", parsed["name"])
	assert.Equal(t, "Description", parsed["description"])
	assert.NotNil(t, parsed["created_at"])
	assert.NotNil(t, parsed["id"])
}

// TestTicketJSONOutput tests JSON serialization of tickets
func TestTicketJSONOutput(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:   project.ID,
		Title:       "Test Ticket",
		Description: "Test Description",
		Priority:    models.PriorityHigh,
		Complexity:  models.ComplexityLarge,
		Status:      models.StatusReady,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Set computed fields
	ticket.TicketKey = "TEST-1"
	ticket.ProjectKey = "TEST"

	// Marshal to JSON
	data, err := json.Marshal(ticket)
	require.NoError(t, err)

	// Verify JSON fields
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Test Ticket", parsed["title"])
	assert.Equal(t, "Test Description", parsed["description"])
	assert.Equal(t, "high", parsed["priority"])
	assert.Equal(t, "large", parsed["complexity"])
	assert.Equal(t, "ready", parsed["status"])
	assert.Equal(t, float64(1), parsed["number"])
	assert.Equal(t, float64(3), parsed["max_retries"])
}

// TestClaimJSONOutput tests JSON serialization of claims
func TestClaimJSONOutput(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test"}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	claimRepo := db.NewClaimRepo(database.DB)
	claim := models.NewClaim(ticket.ID, "test-worker", time.Hour)
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	// Marshal to JSON
	data, err := json.Marshal(claim)
	require.NoError(t, err)

	// Verify JSON fields
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test-worker", parsed["worker_id"])
	assert.Equal(t, "active", parsed["status"])
	assert.NotNil(t, parsed["claimed_at"])
	assert.NotNil(t, parsed["expires_at"])
}

// TestInboxMessageJSONOutput tests JSON serialization of inbox messages
func TestInboxMessageJSONOutput(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test"}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	inboxRepo := db.NewInboxRepo(database.DB)
	message := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "What API to use?", "agent-123")
	err = inboxRepo.Create(message)
	require.NoError(t, err)

	// Marshal to JSON
	data, err := json.Marshal(message)
	require.NoError(t, err)

	// Verify JSON fields
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "question", parsed["message_type"])
	assert.Equal(t, "What API to use?", parsed["content"])
	assert.Equal(t, "agent-123", parsed["from_agent"])
	assert.NotNil(t, parsed["created_at"])
}

// TestStatusResultJSONOutput tests JSON serialization of status results
func TestStatusResultJSONOutput(t *testing.T) {
	result := StatusResult{
		Workable:     5,
		InProgress:   2,
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

	// Marshal to JSON
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Verify JSON fields
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, float64(5), parsed["workable"])
	assert.Equal(t, float64(2), parsed["in_progress"])
	assert.Equal(t, float64(3), parsed["blocked_deps"])
	assert.Equal(t, float64(1), parsed["blocked_human"])
	assert.Equal(t, float64(4), parsed["pending_inbox"])
	assert.Equal(t, "TEST", parsed["project"])

	expiringSoon := parsed["expiring_soon"].([]interface{})
	assert.Len(t, expiringSoon, 1)

	recentActivity := parsed["recent_activity"].([]interface{})
	assert.Len(t, recentActivity, 1)
}

// =============================================================================
// Edge Condition Tests: Dependencies
// =============================================================================

// TestDependencyBlocking tests that dependencies block tickets
func TestDependencyBlocking(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create parent ticket (must be done first)
	parent := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Setup infrastructure",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(parent)
	require.NoError(t, err)

	// Create child ticket (depends on parent)
	child := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Deploy application",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(child)
	require.NoError(t, err)

	// Add dependency
	err = depRepo.Add(child.ID, parent.ID)
	require.NoError(t, err)

	// Verify child has unresolved dependencies
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(child.ID)
	require.NoError(t, err)
	assert.True(t, hasUnresolved)

	// Complete parent
	parent.Status = models.StatusClosed
	completedRes := models.ResolutionCompleted
	parent.Resolution = &completedRes
	err = ticketRepo.Update(parent)
	require.NoError(t, err)

	// Now child should be unblocked
	hasUnresolved, err = depRepo.HasUnresolvedDependencies(child.ID)
	require.NoError(t, err)
	assert.False(t, hasUnresolved)
}

// TestDependencyCancellationDoesNotAutoUnblock tests that cancelling a dependency
// does NOT auto-unblock dependents (only completed resolution does that).
// Non-completed closures require human review.
func TestDependencyCancellationDoesNotAutoUnblock(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create two tickets
	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 1", Status: models.StatusReady}
	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 2", Status: models.StatusReady}
	err = ticketRepo.Create(ticket1)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket2)
	require.NoError(t, err)

	// ticket2 depends on ticket1
	err = depRepo.Add(ticket2.ID, ticket1.ID)
	require.NoError(t, err)

	// ticket2 has unresolved dependencies
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket2.ID)
	require.NoError(t, err)
	assert.True(t, hasUnresolved)

	// Close ticket1 (as wont_do, NOT completed)
	ticket1.Status = models.StatusClosed
	wontDoRes := models.ResolutionWontDo
	ticket1.Resolution = &wontDoRes
	err = ticketRepo.Update(ticket1)
	require.NoError(t, err)

	// ticket2 should STILL have unresolved dependencies
	// (only 'completed' resolution counts as truly resolved)
	hasUnresolved, err = depRepo.HasUnresolvedDependencies(ticket2.ID)
	require.NoError(t, err)
	assert.True(t, hasUnresolved, "Non-completed closure should NOT resolve the dependency")
}

// TestDependencyCompletionUnblocks tests that completing a dependency unblocks dependents
func TestDependencyCompletionUnblocks(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create two tickets
	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 1", Status: models.StatusReady}
	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 2", Status: models.StatusReady}
	err = ticketRepo.Create(ticket1)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket2)
	require.NoError(t, err)

	// ticket2 depends on ticket1
	err = depRepo.Add(ticket2.ID, ticket1.ID)
	require.NoError(t, err)

	// ticket2 has unresolved dependencies
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket2.ID)
	require.NoError(t, err)
	assert.True(t, hasUnresolved)

	// Close ticket1 with 'completed' resolution
	ticket1.Status = models.StatusClosed
	completedRes := models.ResolutionCompleted
	ticket1.Resolution = &completedRes
	err = ticketRepo.Update(ticket1)
	require.NoError(t, err)

	// ticket2 should now have NO unresolved dependencies
	hasUnresolved, err = depRepo.HasUnresolvedDependencies(ticket2.ID)
	require.NoError(t, err)
	assert.False(t, hasUnresolved, "Completed resolution should resolve the dependency")
}

// TestChainedDependencies tests a chain of dependencies
func TestChainedDependencies(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create chain: ticket3 -> ticket2 -> ticket1
	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 1", Status: models.StatusReady}
	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 2", Status: models.StatusReady}
	ticket3 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 3", Status: models.StatusReady}

	err = ticketRepo.Create(ticket1)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket2)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket3)
	require.NoError(t, err)

	err = depRepo.Add(ticket2.ID, ticket1.ID)
	require.NoError(t, err)
	err = depRepo.Add(ticket3.ID, ticket2.ID)
	require.NoError(t, err)

	// All tickets except ticket1 are blocked
	hasUnresolved1, _ := depRepo.HasUnresolvedDependencies(ticket1.ID)
	hasUnresolved2, _ := depRepo.HasUnresolvedDependencies(ticket2.ID)
	hasUnresolved3, _ := depRepo.HasUnresolvedDependencies(ticket3.ID)

	assert.False(t, hasUnresolved1)
	assert.True(t, hasUnresolved2)
	assert.True(t, hasUnresolved3)

	// Complete ticket1
	ticket1.Status = models.StatusClosed
	completedRes := models.ResolutionCompleted
	ticket1.Resolution = &completedRes
	err = ticketRepo.Update(ticket1)
	require.NoError(t, err)

	// ticket2 unblocked, ticket3 still blocked
	hasUnresolved2, _ = depRepo.HasUnresolvedDependencies(ticket2.ID)
	hasUnresolved3, _ = depRepo.HasUnresolvedDependencies(ticket3.ID)
	assert.False(t, hasUnresolved2)
	assert.True(t, hasUnresolved3)

	// Complete ticket2
	ticket2.Status = models.StatusClosed
	ticket2.Resolution = &completedRes
	err = ticketRepo.Update(ticket2)
	require.NoError(t, err)

	// Now ticket3 is unblocked
	hasUnresolved3, _ = depRepo.HasUnresolvedDependencies(ticket3.ID)
	assert.False(t, hasUnresolved3)
}

// TestDependencyRemoval tests removing a dependency
func TestDependencyRemoval(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 1", Status: models.StatusReady}
	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 2", Status: models.StatusReady}
	err = ticketRepo.Create(ticket1)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket2)
	require.NoError(t, err)

	// Add dependency
	err = depRepo.Add(ticket2.ID, ticket1.ID)
	require.NoError(t, err)

	hasUnresolved, _ := depRepo.HasUnresolvedDependencies(ticket2.ID)
	assert.True(t, hasUnresolved)

	// Remove dependency
	err = depRepo.Remove(ticket2.ID, ticket1.ID)
	require.NoError(t, err)

	// Now unblocked
	hasUnresolved, _ = depRepo.HasUnresolvedDependencies(ticket2.ID)
	assert.False(t, hasUnresolved)
}

// =============================================================================
// Edge Condition Tests: Inbox and Human Blocking
// =============================================================================

// TestInboxBlocksTicket tests that flagging a ticket blocks it
func TestInboxBlocksTicket(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test",
		Status:    models.StatusInProgress,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Verify ticket can be flagged
	assert.True(t, state.CanBeEscalated(ticket.Status))

	// Flag the ticket
	ticket.Status = models.StatusHuman
	ticket.HumanFlagReason = "unclear_requirements"
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Create inbox message
	inboxRepo := db.NewInboxRepo(database.DB)
	message := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Need clarification on API design", "worker-1")
	err = inboxRepo.Create(message)
	require.NoError(t, err)

	// Verify ticket is blocked
	blockedTicket, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusHuman, blockedTicket.Status)

	// Verify pending message
	pending, err := inboxRepo.ListPending()
	require.NoError(t, err)
	assert.Len(t, pending, 1)
}

// TestInboxResponseUnblocksTicket tests that responding to inbox unblocks ticket
func TestInboxResponseUnblocksTicket(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:       project.ID,
		Title:           "Test",
		Status:          models.StatusHuman,
		HumanFlagReason: "decision_needed",
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	inboxRepo := db.NewInboxRepo(database.DB)
	message := models.NewInboxMessage(ticket.ID, models.MessageTypeDecision, "Choose between REST and GraphQL", "worker-1")
	err = inboxRepo.Create(message)
	require.NoError(t, err)

	// Respond to message
	err = inboxRepo.Respond(message.ID, "Use REST for simplicity")
	require.NoError(t, err)

	// Simulate CLI behavior: update ticket status
	ticket.Status = models.StatusReady
	ticket.RetryCount = 0 // Reset on human response
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Verify ticket is ready again
	unblockedTicket, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, unblockedTicket.Status)

	// Verify message is no longer pending
	message, err = inboxRepo.GetByID(message.ID)
	require.NoError(t, err)
	assert.False(t, message.IsPending())
	assert.Equal(t, "Use REST for simplicity", message.Response)
}

// TestMultipleInboxMessages tests multiple pending messages for a ticket
func TestMultipleInboxMessages(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test", Status: models.StatusHuman}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	inboxRepo := db.NewInboxRepo(database.DB)

	// Create multiple messages
	for i := 0; i < 3; i++ {
		msg := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Question", "worker")
		err := inboxRepo.Create(msg)
		require.NoError(t, err)
	}

	// Verify all pending
	pending, err := inboxRepo.ListPending()
	require.NoError(t, err)
	assert.Len(t, pending, 3)

	count, err := inboxRepo.CountPending()
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Respond to one
	err = inboxRepo.Respond(pending[0].ID, "Answer")
	require.NoError(t, err)

	count, err = inboxRepo.CountPending()
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// =============================================================================
// Edge Condition Tests: Workable Tickets
// =============================================================================

// TestWorkableExcludesBlocked tests that workable list excludes blocked tickets
func TestWorkableExcludesBlocked(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create a ready ticket (workable)
	workable := &models.Ticket{ProjectID: project.ID, Title: "Workable", Status: models.StatusReady}
	err = ticketRepo.Create(workable)
	require.NoError(t, err)

	// Create a blocked ticket
	blocked := &models.Ticket{ProjectID: project.ID, Title: "Blocked", Status: models.StatusReady}
	err = ticketRepo.Create(blocked)
	require.NoError(t, err)

	// Add dependency to make it blocked
	err = depRepo.Add(blocked.ID, workable.ID)
	require.NoError(t, err)

	// List workable - should only include the non-blocked ticket
	tickets, err := ticketRepo.ListWorkable(db.TicketFilter{})
	require.NoError(t, err)
	assert.Len(t, tickets, 1)
	assert.Equal(t, "Workable", tickets[0].Title)
}

// TestWorkableExcludesMaxRetries tests that tickets at max retries need application-level filtering
// Note: The ListWorkable query returns ready tickets with resolved deps, but retry filtering
// is done at application level (in ticket_util.go's runTicketNext function)
func TestWorkableExcludesMaxRetries(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create a ticket at max retries
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Title:      "At Max Retries",
		Status:     models.StatusReady,
		RetryCount: 3, // Default max is 3
		MaxRetries: 3,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// ListWorkable returns the ticket (it's ready with no unresolved deps)
	tickets, err := ticketRepo.ListWorkable(db.TicketFilter{})
	require.NoError(t, err)
	assert.Len(t, tickets, 1) // Returned from DB

	// Application-level filtering: exclude tickets at max retries
	workable := make([]*models.Ticket, 0)
	for _, t := range tickets {
		if t.RetryCount < t.MaxRetries {
			workable = append(workable, t)
		}
	}
	assert.Len(t, workable, 0) // Filtered out at application level
}

// TestWorkableOnlyReady tests that workable only includes ready status
func TestWorkableOnlyReady(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create tickets with various statuses
	statuses := []models.Status{
		models.StatusBlocked,
		models.StatusReady,
		models.StatusInProgress,
		models.StatusHuman,
		models.StatusReview,
		models.StatusClosed,
	}

	for _, s := range statuses {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     string(s),
			Status:    s,
		}
		// Closed tickets require a resolution
		if s == models.StatusClosed {
			res := models.ResolutionCompleted
			ticket.Resolution = &res
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// List workable - only ready should be included
	tickets, err := ticketRepo.ListWorkable(db.TicketFilter{})
	require.NoError(t, err)
	assert.Len(t, tickets, 1)
	assert.Equal(t, models.StatusReady, tickets[0].Status)
}

// =============================================================================
// Edge Condition Tests: Claim Expiration
// =============================================================================

// TestClaimExpirationEscalation tests escalation when max retries reached
func TestClaimExpirationEscalation(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Title:      "Test",
		Status:     models.StatusInProgress,
		RetryCount: 2, // One more retry will hit max (3)
		MaxRetries: 3,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Simulate what happens on claim expiry when at max-1 retries
	ticket.RetryCount++
	if ticket.RetryCount >= ticket.MaxRetries {
		ticket.Status = models.StatusHuman
		ticket.HumanFlagReason = "max_retries_reached"
	} else {
		ticket.Status = models.StatusReady
	}
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Verify escalation
	escalatedTicket, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusHuman, escalatedTicket.Status)
	assert.Equal(t, 3, escalatedTicket.RetryCount)
}

// TestClaimListByTicket tests listing claims for a specific ticket
func TestClaimListByTicket(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test"}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	claimRepo := db.NewClaimRepo(database.DB)

	// Create multiple claims (simulate claim -> release -> claim cycle)
	claim1 := models.NewClaim(ticket.ID, "worker-1", time.Hour)
	err = claimRepo.Create(claim1)
	require.NoError(t, err)
	err = claimRepo.Release(claim1.ID, models.ClaimStatusReleased)
	require.NoError(t, err)

	claim2 := models.NewClaim(ticket.ID, "worker-2", time.Hour)
	err = claimRepo.Create(claim2)
	require.NoError(t, err)
	err = claimRepo.Release(claim2.ID, models.ClaimStatusCompleted)
	require.NoError(t, err)

	// List claims for ticket
	claims, err := claimRepo.ListByTicketID(ticket.ID)
	require.NoError(t, err)
	assert.Len(t, claims, 2)

	// Should be ordered by claimed_at DESC
	assert.Equal(t, "worker-2", claims[0].WorkerID)
	assert.Equal(t, "worker-1", claims[1].WorkerID)
}

// =============================================================================
// Project and Ticket Filtering Tests
// =============================================================================

// TestTicketFilterByProject tests filtering tickets by project
func TestTicketFilterByProject(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)

	// Create two projects
	proj1 := &models.Project{Key: "PROJ1", Name: "Project 1"}
	proj2 := &models.Project{Key: "PROJ2", Name: "Project 2"}
	err := projectRepo.Create(proj1)
	require.NoError(t, err)
	err = projectRepo.Create(proj2)
	require.NoError(t, err)

	// Create tickets in each project
	for i := 0; i < 3; i++ {
		err := ticketRepo.Create(&models.Ticket{ProjectID: proj1.ID, Title: "P1 Ticket", Status: models.StatusReady})
		require.NoError(t, err)
	}
	for i := 0; i < 5; i++ {
		err := ticketRepo.Create(&models.Ticket{ProjectID: proj2.ID, Title: "P2 Ticket", Status: models.StatusReady})
		require.NoError(t, err)
	}

	// Filter by project
	proj1Tickets, err := ticketRepo.List(db.TicketFilter{ProjectKey: "PROJ1"})
	require.NoError(t, err)
	assert.Len(t, proj1Tickets, 3)

	proj2Tickets, err := ticketRepo.List(db.TicketFilter{ProjectKey: "PROJ2"})
	require.NoError(t, err)
	assert.Len(t, proj2Tickets, 5)

	// All tickets
	allTickets, err := ticketRepo.List(db.TicketFilter{})
	require.NoError(t, err)
	assert.Len(t, allTickets, 8)
}

// TestTicketFilterByStatus tests filtering tickets by status
func TestTicketFilterByStatus(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create tickets with various statuses
	statuses := []models.Status{
		models.StatusReady,
		models.StatusReady,
		models.StatusInProgress,
		models.StatusClosed,
		models.StatusClosed,
		models.StatusClosed,
	}

	completedRes := models.ResolutionCompleted
	for _, s := range statuses {
		ticket := &models.Ticket{ProjectID: project.ID, Title: string(s), Status: s}
		if s == models.StatusClosed {
			ticket.Resolution = &completedRes
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// Filter by status
	readyStatus := models.StatusReady
	readyTickets, err := ticketRepo.List(db.TicketFilter{Status: &readyStatus})
	require.NoError(t, err)
	assert.Len(t, readyTickets, 2)

	closedStatus := models.StatusClosed
	closedTickets, err := ticketRepo.List(db.TicketFilter{Status: &closedStatus})
	require.NoError(t, err)
	assert.Len(t, closedTickets, 3)
}

// TestTicketFilterByPriority tests filtering tickets by priority
func TestTicketFilterByPriority(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create tickets with various priorities
	priorities := []models.Priority{
		models.PriorityHighest,
		models.PriorityHigh,
		models.PriorityHigh,
		models.PriorityMedium,
		models.PriorityLow,
	}

	for _, p := range priorities {
		err := ticketRepo.Create(&models.Ticket{ProjectID: project.ID, Title: string(p), Priority: p})
		require.NoError(t, err)
	}

	// Filter by priority
	highPriority := models.PriorityHigh
	highTickets, err := ticketRepo.List(db.TicketFilter{Priority: &highPriority})
	require.NoError(t, err)
	assert.Len(t, highTickets, 2)
}

// =============================================================================
// Activity Log Tests
// =============================================================================

// TestActivityLogFiltering tests filtering activity logs
func TestActivityLogFiltering(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test"}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	activityRepo := db.NewActivityRepo(database.DB)

	// Log various activities
	actions := []struct {
		action    models.Action
		actorType models.ActorType
	}{
		{models.ActionClaimed, models.ActorTypeAgent},
		{models.ActionCompleted, models.ActorTypeAgent},
		{models.ActionAccepted, models.ActorTypeHuman},
	}

	for _, a := range actions {
		err := activityRepo.LogAction(ticket.ID, a.action, a.actorType, "", "")
		require.NoError(t, err)
	}

	// Filter by action
	claimedAction := models.ActionClaimed
	filter := db.ActivityFilter{
		TicketID: &ticket.ID,
		Action:   &claimedAction,
	}
	logs, err := activityRepo.List(filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logs), 1)

	// Filter by actor type
	humanActor := models.ActorTypeHuman
	filter = db.ActivityFilter{
		TicketID:  &ticket.ID,
		ActorType: &humanActor,
	}
	logs, err = activityRepo.List(filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logs), 1)
}

// TestActivityLogWithDetails tests logging activities with JSON details
func TestActivityLogWithDetails(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test"}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	activityRepo := db.NewActivityRepo(database.DB)

	// Log with details
	details := map[string]interface{}{
		"worker_id":     "agent-123",
		"duration_mins": 60,
		"extra_data":    "test value",
	}
	err = activityRepo.LogActionWithDetails(ticket.ID, models.ActionClaimed, models.ActorTypeAgent, "agent-123", "Claimed for work", details)
	require.NoError(t, err)

	// Retrieve and verify
	logs, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)

	// Find our specific log entry
	var found bool
	for _, log := range logs {
		if log.ActorID == "agent-123" && log.Action == models.ActionClaimed {
			found = true
			assert.Equal(t, "Claimed for work", log.Summary)
			assert.NotEmpty(t, log.Details)

			// Parse and verify details
			var parsedDetails map[string]interface{}
			err = json.Unmarshal([]byte(log.Details), &parsedDetails)
			require.NoError(t, err)
			assert.Equal(t, "agent-123", parsedDetails["worker_id"])
			break
		}
	}
	assert.True(t, found, "should find our logged activity")
}

// =============================================================================
// Auto-Transition Tests (WARK-11)
// =============================================================================

// TestAutoTransitionToBlockedOnCreate tests that tickets with unresolved dependencies
// automatically transition to blocked status on creation
func TestAutoTransitionToBlockedOnCreate(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create a dependency ticket (not completed)
	depTicket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Dependency Ticket",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(depTicket)
	require.NoError(t, err)

	// Create main ticket with dependency
	mainTicket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Main Ticket",
		Status:    models.StatusReady, // Initial status
	}
	err = ticketRepo.Create(mainTicket)
	require.NoError(t, err)

	// Add dependency (simulating what happens in ticket create with --depends-on)
	err = depRepo.Add(mainTicket.ID, depTicket.ID)
	require.NoError(t, err)

	// Check dependencies
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(mainTicket.ID)
	require.NoError(t, err)
	assert.True(t, hasUnresolved, "should have unresolved dependencies")

	// Simulate auto-transition (this is what the CLI does after adding deps)
	if hasUnresolved && mainTicket.Status == models.StatusReady {
		mainTicket.Status = models.StatusBlocked
		err = ticketRepo.Update(mainTicket)
		require.NoError(t, err)
	}

	// Verify final state
	updatedTicket, err := ticketRepo.GetByID(mainTicket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusBlocked, updatedTicket.Status, "ticket should be blocked due to unresolved dependency")
}

// TestAutoTransitionToReadyWhenDependencyResolved tests that blocked tickets
// automatically transition to ready when their dependencies are resolved
func TestAutoTransitionToReadyWhenDependencyResolved(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create dependency ticket
	depTicket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Dependency",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(depTicket)
	require.NoError(t, err)

	// Create blocked ticket
	blockedTicket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Blocked Ticket",
		Status:    models.StatusBlocked,
	}
	err = ticketRepo.Create(blockedTicket)
	require.NoError(t, err)

	// Add dependency
	err = depRepo.Add(blockedTicket.ID, depTicket.ID)
	require.NoError(t, err)

	// Complete the dependency
	depTicket.Status = models.StatusClosed
	completed := models.ResolutionCompleted
	depTicket.Resolution = &completed
	err = ticketRepo.Update(depTicket)
	require.NoError(t, err)

	// Check if dependencies are now resolved
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(blockedTicket.ID)
	require.NoError(t, err)
	assert.False(t, hasUnresolved, "dependencies should be resolved now")

	// Simulate auto-transition (this is what the CLI/tasks do)
	if !hasUnresolved && blockedTicket.Status == models.StatusBlocked {
		blockedTicket.Status = models.StatusReady
		err = ticketRepo.Update(blockedTicket)
		require.NoError(t, err)
	}

	// Verify final state
	updatedTicket, err := ticketRepo.GetByID(blockedTicket.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, updatedTicket.Status, "ticket should be ready after dependency resolved")
}

// TestAutoTransitionOnDependencyEdit tests status transitions when dependencies
// are added or removed via ticket edit
func TestAutoTransitionOnDependencyEdit(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create two tickets
	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 1", Status: models.StatusReady}
	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 2", Status: models.StatusReady}
	err = ticketRepo.Create(ticket1)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket2)
	require.NoError(t, err)

	// Test 1: Adding dependency to ready ticket should block it
	err = depRepo.Add(ticket2.ID, ticket1.ID)
	require.NoError(t, err)

	// Check and transition
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket2.ID)
	require.NoError(t, err)
	assert.True(t, hasUnresolved)

	if hasUnresolved && ticket2.Status == models.StatusReady {
		ticket2.Status = models.StatusBlocked
		err = ticketRepo.Update(ticket2)
		require.NoError(t, err)
	}

	ticket2, _ = ticketRepo.GetByID(ticket2.ID)
	assert.Equal(t, models.StatusBlocked, ticket2.Status, "ticket2 should be blocked after adding dependency")

	// Test 2: Removing dependency from blocked ticket should unblock it
	err = depRepo.Remove(ticket2.ID, ticket1.ID)
	require.NoError(t, err)

	hasUnresolved, err = depRepo.HasUnresolvedDependencies(ticket2.ID)
	require.NoError(t, err)
	assert.False(t, hasUnresolved)

	if !hasUnresolved && ticket2.Status == models.StatusBlocked {
		ticket2.Status = models.StatusReady
		err = ticketRepo.Update(ticket2)
		require.NoError(t, err)
	}

	ticket2, _ = ticketRepo.GetByID(ticket2.ID)
	assert.Equal(t, models.StatusReady, ticket2.Status, "ticket2 should be ready after removing dependency")
}

// TestNoAutoTransitionWithResolvedDependency tests that tickets don't get blocked
// when depending on already-completed tickets
func TestNoAutoTransitionWithResolvedDependency(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create a completed dependency
	completedTicket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Completed Ticket",
		Status:    models.StatusClosed,
	}
	completed := models.ResolutionCompleted
	completedTicket.Resolution = &completed
	err = ticketRepo.Create(completedTicket)
	require.NoError(t, err)

	// Create main ticket
	mainTicket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Main Ticket",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(mainTicket)
	require.NoError(t, err)

	// Add dependency on completed ticket
	err = depRepo.Add(mainTicket.ID, completedTicket.ID)
	require.NoError(t, err)

	// Check dependencies - should be resolved since dep is completed
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(mainTicket.ID)
	require.NoError(t, err)
	assert.False(t, hasUnresolved, "dependency should be resolved")

	// Status should remain ready
	mainTicket, _ = ticketRepo.GetByID(mainTicket.ID)
	assert.Equal(t, models.StatusReady, mainTicket.Status, "ticket should stay ready when depending on completed ticket")
}
