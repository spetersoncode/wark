package tasks

import (
	"testing"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyResolver_OnTicketCompleted_UnblocksDependents(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create ticket1 (dependency) and ticket2 (depends on ticket1)
	ticket1 := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Dependency",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket1))

	ticket2 := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Dependent",
		Status:    models.StatusBlocked,
	}
	require.NoError(t, ticketRepo.Create(ticket2))

	// Add dependency
	require.NoError(t, depRepo.Add(ticket2.ID, ticket1.ID))

	// Complete ticket1
	ticket1.Status = models.StatusDone
	now := time.Now()
	ticket1.CompletedAt = &now
	require.NoError(t, ticketRepo.Update(ticket1))

	// Run dependency resolution
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.OnTicketCompleted(ticket1.ID, false)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Unblocked)
	assert.Equal(t, 0, result.Errors)

	// Verify ticket2 was unblocked
	updated, err := ticketRepo.GetByID(ticket2.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, updated.Status)
}

func TestDependencyResolver_OnTicketCompleted_MultipleDependencies(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create ticket1 and ticket2 (dependencies)
	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Dep 1", Status: models.StatusDone}
	require.NoError(t, ticketRepo.Create(ticket1))

	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Dep 2", Status: models.StatusInProgress}
	require.NoError(t, ticketRepo.Create(ticket2))

	// Create ticket3 that depends on both
	ticket3 := &models.Ticket{ProjectID: project.ID, Title: "Blocked", Status: models.StatusBlocked}
	require.NoError(t, ticketRepo.Create(ticket3))

	require.NoError(t, depRepo.Add(ticket3.ID, ticket1.ID))
	require.NoError(t, depRepo.Add(ticket3.ID, ticket2.ID))

	// Complete ticket2
	ticket2.Status = models.StatusDone
	require.NoError(t, ticketRepo.Update(ticket2))

	// Run resolution
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.OnTicketCompleted(ticket2.ID, false)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Unblocked)

	// Verify ticket3 was unblocked
	updated, err := ticketRepo.GetByID(ticket3.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, updated.Status)
}

func TestDependencyResolver_OnTicketCompleted_DoesNotUnblockWithRemainingDeps(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create ticket1 (done) and ticket2 (not done)
	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Dep 1", Status: models.StatusInProgress}
	require.NoError(t, ticketRepo.Create(ticket1))

	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Dep 2", Status: models.StatusReady}
	require.NoError(t, ticketRepo.Create(ticket2))

	// Create ticket3 that depends on both
	ticket3 := &models.Ticket{ProjectID: project.ID, Title: "Blocked", Status: models.StatusBlocked}
	require.NoError(t, ticketRepo.Create(ticket3))

	require.NoError(t, depRepo.Add(ticket3.ID, ticket1.ID))
	require.NoError(t, depRepo.Add(ticket3.ID, ticket2.ID))

	// Complete only ticket1
	ticket1.Status = models.StatusDone
	require.NoError(t, ticketRepo.Update(ticket1))

	// Run resolution
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.OnTicketCompleted(ticket1.ID, false)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Unblocked) // ticket2 is still not done

	// Verify ticket3 is still blocked
	updated, err := ticketRepo.GetByID(ticket3.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusBlocked, updated.Status)
}

func TestDependencyResolver_OnTicketCompleted_UpdatesParent(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create parent ticket
	parent := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Parent",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(parent))

	// Create child tickets
	child1 := &models.Ticket{
		ProjectID:      project.ID,
		Title:          "Child 1",
		Status:         models.StatusDone,
		ParentTicketID: &parent.ID,
	}
	require.NoError(t, ticketRepo.Create(child1))

	child2 := &models.Ticket{
		ProjectID:      project.ID,
		Title:          "Child 2",
		Status:         models.StatusInProgress,
		ParentTicketID: &parent.ID,
	}
	require.NoError(t, ticketRepo.Create(child2))

	// Complete child2
	child2.Status = models.StatusDone
	require.NoError(t, ticketRepo.Update(child2))

	// Run resolution (without auto-accept)
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.OnTicketCompleted(child2.ID, false)
	require.NoError(t, err)

	assert.Equal(t, 1, result.ParentsUpdated)
	assert.Len(t, result.ParentResults, 1)
	assert.Equal(t, "review", result.ParentResults[0].NewStatus)
	assert.False(t, result.ParentResults[0].AutoAccepted)

	// Verify parent moved to review
	updated, err := ticketRepo.GetByID(parent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusReview, updated.Status)
}

func TestDependencyResolver_OnTicketCompleted_ParentAutoAccept(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create parent
	parent := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Parent",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(parent))

	// Create single child
	child := &models.Ticket{
		ProjectID:      project.ID,
		Title:          "Child",
		Status:         models.StatusInProgress,
		ParentTicketID: &parent.ID,
	}
	require.NoError(t, ticketRepo.Create(child))

	// Complete child
	child.Status = models.StatusDone
	require.NoError(t, ticketRepo.Update(child))

	// Run resolution with auto-accept
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.OnTicketCompleted(child.ID, true)
	require.NoError(t, err)

	assert.Equal(t, 1, result.ParentsUpdated)
	assert.True(t, result.ParentResults[0].AutoAccepted)

	// Verify parent is done
	updated, err := ticketRepo.GetByID(parent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusDone, updated.Status)
	assert.NotNil(t, updated.CompletedAt)
}

func TestDependencyResolver_OnTicketCompleted_ParentNotUpdatedWithIncompleteChildren(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create parent
	parent := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Parent",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(parent))

	// Create children
	child1 := &models.Ticket{
		ProjectID:      project.ID,
		Title:          "Child 1",
		Status:         models.StatusInProgress,
		ParentTicketID: &parent.ID,
	}
	require.NoError(t, ticketRepo.Create(child1))

	child2 := &models.Ticket{
		ProjectID:      project.ID,
		Title:          "Child 2",
		Status:         models.StatusReady, // Not done
		ParentTicketID: &parent.ID,
	}
	require.NoError(t, ticketRepo.Create(child2))

	// Complete child1
	child1.Status = models.StatusDone
	require.NoError(t, ticketRepo.Update(child1))

	// Run resolution
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.OnTicketCompleted(child1.ID, false)
	require.NoError(t, err)

	assert.Equal(t, 0, result.ParentsUpdated) // Parent not updated

	// Verify parent is still in progress
	updated, err := ticketRepo.GetByID(parent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusInProgress, updated.Status)
}

func TestDependencyResolver_ResolveAll(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create dependency (already done)
	dep := &models.Ticket{ProjectID: project.ID, Title: "Dependency", Status: models.StatusDone}
	require.NoError(t, ticketRepo.Create(dep))

	// Create blocked tickets whose dependencies are resolved
	blocked1 := &models.Ticket{ProjectID: project.ID, Title: "Blocked 1", Status: models.StatusBlocked}
	require.NoError(t, ticketRepo.Create(blocked1))
	require.NoError(t, depRepo.Add(blocked1.ID, dep.ID))

	blocked2 := &models.Ticket{ProjectID: project.ID, Title: "Blocked 2", Status: models.StatusBlocked}
	require.NoError(t, ticketRepo.Create(blocked2))
	require.NoError(t, depRepo.Add(blocked2.ID, dep.ID))

	// Run batch resolution
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.ResolveAll()
	require.NoError(t, err)

	assert.Equal(t, 2, result.Unblocked)
	assert.Equal(t, 0, result.Errors)

	// Verify both were unblocked
	updated1, _ := ticketRepo.GetByID(blocked1.ID)
	assert.Equal(t, models.StatusReady, updated1.Status)

	updated2, _ := ticketRepo.GetByID(blocked2.ID)
	assert.Equal(t, models.StatusReady, updated2.Status)
}

func TestDependencyResolver_SkipsNonBlockedTickets(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	// Create dependency (done)
	dep := &models.Ticket{ProjectID: project.ID, Title: "Dependency", Status: models.StatusDone}
	require.NoError(t, ticketRepo.Create(dep))

	// Create ready ticket with resolved dependency
	ready := &models.Ticket{ProjectID: project.ID, Title: "Ready", Status: models.StatusReady}
	require.NoError(t, ticketRepo.Create(ready))
	require.NoError(t, depRepo.Add(ready.ID, dep.ID))

	// Run resolution
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.OnTicketCompleted(dep.ID, false)
	require.NoError(t, err)

	// Should not count non-blocked ticket
	assert.Equal(t, 0, result.Unblocked)
}

func TestDependencyResolver_NoParentUpdate(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	require.NoError(t, projectRepo.Create(project))

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create standalone ticket (no parent)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Standalone",
		Status:    models.StatusInProgress,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Complete ticket
	ticket.Status = models.StatusDone
	require.NoError(t, ticketRepo.Update(ticket))

	// Run resolution
	resolver := NewDependencyResolver(database.DB)
	result, err := resolver.OnTicketCompleted(ticket.ID, false)
	require.NoError(t, err)

	assert.Equal(t, 0, result.ParentsUpdated)
	assert.Len(t, result.ParentResults, 0)
}
