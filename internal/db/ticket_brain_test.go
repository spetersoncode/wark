package db

import (
	"testing"

	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketBrain_CreateAndRetrieve(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	ticketRepo := NewTicketRepo(db)

	// Create ticket with brain set
	brainValue := "sonnet"
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket with brain",
		Brain:     &brainValue,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Retrieve and verify brain
	retrieved, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Brain)
	assert.Equal(t, "sonnet", *retrieved.Brain)
}

func TestTicketBrain_CreateWithoutBrain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	ticketRepo := NewTicketRepo(db)

	// Create ticket without brain
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket without brain",
		Brain:     nil,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Retrieve and verify no brain
	retrieved, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved.Brain)
}

func TestTicketBrain_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	ticketRepo := NewTicketRepo(db)

	// Create ticket without brain
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket",
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Update to add brain
	brainValue := "claude-code"
	ticket.Brain = &brainValue
	require.NoError(t, ticketRepo.Update(ticket))

	// Verify brain was set
	retrieved, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Brain)
	assert.Equal(t, "claude-code", *retrieved.Brain)
}

func TestTicketBrain_UpdateToClear(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	ticketRepo := NewTicketRepo(db)

	// Create ticket with brain
	brainValue := "opus"
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket",
		Brain:     &brainValue,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Update to clear brain
	ticket.Brain = nil
	require.NoError(t, ticketRepo.Update(ticket))

	// Verify brain was cleared
	retrieved, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved.Brain)
}

func TestTicketBrain_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	ticketRepo := NewTicketRepo(db)

	// Create tickets with different brains
	sonnetBrain := "sonnet"
	claudeCodeBrain := "claude-code"
	tickets := []*models.Ticket{
		{
			ProjectID: projectID,
			Title:     "Ticket with sonnet",
			Brain:     &sonnetBrain,
		},
		{
			ProjectID: projectID,
			Title:     "Ticket with claude-code",
			Brain:     &claudeCodeBrain,
		},
		{
			ProjectID: projectID,
			Title:     "Ticket without brain",
			Brain:     nil,
		},
	}

	for _, ticket := range tickets {
		require.NoError(t, ticketRepo.Create(ticket))
	}

	// List all tickets
	filter := TicketFilter{
		ProjectID: &projectID,
	}
	retrieved, err := ticketRepo.List(filter)
	require.NoError(t, err)
	require.Len(t, retrieved, 3)

	// Verify brains are correctly retrieved
	assert.NotNil(t, retrieved[0].Brain)
	assert.Equal(t, "sonnet", *retrieved[0].Brain)

	assert.NotNil(t, retrieved[1].Brain)
	assert.Equal(t, "claude-code", *retrieved[1].Brain)

	assert.Nil(t, retrieved[2].Brain)
}

func TestTicketBrain_GetByKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	projectRepo := NewProjectRepo(db)
	project, err := projectRepo.GetByID(projectID)
	require.NoError(t, err)

	ticketRepo := NewTicketRepo(db)

	// Create ticket with brain
	brainValue := "qwen"
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket",
		Brain:     &brainValue,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Retrieve by key and verify brain
	retrieved, err := ticketRepo.GetByKey(project.Key, ticket.Number)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Brain)
	assert.Equal(t, "qwen", *retrieved.Brain)
}
