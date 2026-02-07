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
	brain := &models.Brain{
		Type:  "model",
		Value: "sonnet",
	}
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket with brain",
		Brain:     brain,
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Retrieve and verify brain
	retrieved, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Brain)
	assert.Equal(t, "model", retrieved.Brain.Type)
	assert.Equal(t, "sonnet", retrieved.Brain.Value)
	assert.Equal(t, "model:sonnet", retrieved.Brain.String())
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
	ticket.Brain = &models.Brain{
		Type:  "tool",
		Value: "claude-code",
	}
	require.NoError(t, ticketRepo.Update(ticket))

	// Verify brain was set
	retrieved, err := ticketRepo.GetByID(ticket.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Brain)
	assert.Equal(t, "tool", retrieved.Brain.Type)
	assert.Equal(t, "claude-code", retrieved.Brain.Value)
	assert.Equal(t, "tool:claude-code", retrieved.Brain.String())
}

func TestTicketBrain_UpdateToClear(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)

	ticketRepo := NewTicketRepo(db)

	// Create ticket with brain
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket",
		Brain: &models.Brain{
			Type:  "model",
			Value: "opus",
		},
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
	tickets := []*models.Ticket{
		{
			ProjectID: projectID,
			Title:     "Ticket with sonnet",
			Brain: &models.Brain{
				Type:  "model",
				Value: "sonnet",
			},
		},
		{
			ProjectID: projectID,
			Title:     "Ticket with claude-code",
			Brain: &models.Brain{
				Type:  "tool",
				Value: "claude-code",
			},
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
	assert.Equal(t, "model:sonnet", retrieved[0].Brain.String())

	assert.NotNil(t, retrieved[1].Brain)
	assert.Equal(t, "tool:claude-code", retrieved[1].Brain.String())

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
	ticket := &models.Ticket{
		ProjectID: projectID,
		Title:     "Test ticket",
		Brain: &models.Brain{
			Type:  "model",
			Value: "qwen",
		},
	}
	require.NoError(t, ticketRepo.Create(ticket))

	// Retrieve by key and verify brain
	retrieved, err := ticketRepo.GetByKey(project.Key, ticket.Number)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Brain)
	assert.Equal(t, "model", retrieved.Brain.Type)
	assert.Equal(t, "qwen", retrieved.Brain.Value)
}

func TestBrain_String(t *testing.T) {
	tests := []struct {
		name     string
		brain    *models.Brain
		expected string
	}{
		{
			name: "model brain",
			brain: &models.Brain{
				Type:  "model",
				Value: "sonnet",
			},
			expected: "model:sonnet",
		},
		{
			name: "tool brain",
			brain: &models.Brain{
				Type:  "tool",
				Value: "claude-code",
			},
			expected: "tool:claude-code",
		},
		{
			name:     "nil brain",
			brain:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.brain.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
