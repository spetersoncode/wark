package cli

import (
	"testing"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboxCreate(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project and ticket
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

	// Create inbox message
	inboxRepo := db.NewInboxRepo(database.DB)
	message := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "What API to use?", "agent-123")
	err = inboxRepo.Create(message)
	require.NoError(t, err)

	assert.Equal(t, int64(1), message.ID)
	assert.Equal(t, models.MessageTypeQuestion, message.MessageType)
	assert.True(t, message.IsPending())
}

func TestInboxList(t *testing.T) {
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

	// Create messages
	inboxRepo := db.NewInboxRepo(database.DB)
	messages := []*models.InboxMessage{
		models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Question 1", "agent-1"),
		models.NewInboxMessage(ticket.ID, models.MessageTypeDecision, "Decision needed", "agent-2"),
		models.NewInboxMessage(ticket.ID, models.MessageTypeInfo, "FYI info", "agent-3"),
	}

	for _, m := range messages {
		err := inboxRepo.Create(m)
		require.NoError(t, err)
	}

	// List all
	allMessages, err := inboxRepo.List(db.InboxFilter{})
	require.NoError(t, err)
	assert.Len(t, allMessages, 3)

	// List pending
	pendingMessages, err := inboxRepo.ListPending()
	require.NoError(t, err)
	assert.Len(t, pendingMessages, 3)

	// List by type
	questionMessages, err := inboxRepo.List(db.InboxFilter{MessageType: func() *models.MessageType { t := models.MessageTypeQuestion; return &t }()})
	require.NoError(t, err)
	assert.Len(t, questionMessages, 1)
}

func TestInboxRespond(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Test Ticket", Status: models.StatusHuman}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create message
	inboxRepo := db.NewInboxRepo(database.DB)
	message := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Which approach?", "agent-1")
	err = inboxRepo.Create(message)
	require.NoError(t, err)

	assert.True(t, message.IsPending())

	// Respond
	err = inboxRepo.Respond(message.ID, "Use approach A")
	require.NoError(t, err)

	// Verify response
	updated, err := inboxRepo.GetByID(message.ID)
	require.NoError(t, err)
	assert.False(t, updated.IsPending())
	assert.Equal(t, "Use approach A", updated.Response)
	assert.NotNil(t, updated.RespondedAt)
}

func TestInboxCountPending(t *testing.T) {
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

	// Create messages
	inboxRepo := db.NewInboxRepo(database.DB)
	for i := 0; i < 5; i++ {
		m := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Question", "agent")
		err := inboxRepo.Create(m)
		require.NoError(t, err)
	}

	// Count pending
	count, err := inboxRepo.CountPending()
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	// Respond to one
	messages, _ := inboxRepo.ListPending()
	err = inboxRepo.Respond(messages[0].ID, "Answer")
	require.NoError(t, err)

	// Count again
	count, err = inboxRepo.CountPending()
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestParseID(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"1", 1, false},
		{"42", 42, false},
		{"999", 999, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestInboxMessageTypes(t *testing.T) {
	// Verify all message types
	types := []models.MessageType{
		models.MessageTypeQuestion,
		models.MessageTypeDecision,
		models.MessageTypeReview,
		models.MessageTypeEscalation,
		models.MessageTypeInfo,
	}

	for _, mt := range types {
		assert.True(t, mt.IsValid(), "message type %s should be valid", mt)
	}

	// Invalid type
	assert.False(t, models.MessageType("invalid").IsValid())

	// Check requires response
	assert.True(t, models.MessageTypeQuestion.RequiresResponse())
	assert.True(t, models.MessageTypeDecision.RequiresResponse())
	assert.True(t, models.MessageTypeEscalation.RequiresResponse())
	assert.False(t, models.MessageTypeInfo.RequiresResponse())
	assert.False(t, models.MessageTypeReview.RequiresResponse())
}

func TestInboxRespondLogsActivityToTicket(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project and ticket
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusHuman,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create inbox message
	inboxRepo := db.NewInboxRepo(database.DB)
	message := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Which API should I use?", "agent-123")
	err = inboxRepo.Create(message)
	require.NoError(t, err)

	// Respond to the message
	response := "Use the REST API for simplicity"
	err = inboxRepo.Respond(message.ID, response)
	require.NoError(t, err)

	// Log activity (simulating what runInboxRespond does)
	activityRepo := db.NewActivityRepo(database.DB)
	err = activityRepo.LogActionWithDetails(ticket.ID, models.ActionHumanResponded, models.ActorTypeHuman, "",
		"Responded to message",
		map[string]interface{}{
			"inbox_message_id": message.ID,
			"response":         response,
		})
	require.NoError(t, err)

	// Verify activity was logged
	activities, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)
	require.NotEmpty(t, activities, "expected at least one activity log entry")

	// Find the human_responded activity
	var foundActivity *models.ActivityLog
	for _, a := range activities {
		if a.Action == models.ActionHumanResponded {
			foundActivity = a
			break
		}
	}

	require.NotNil(t, foundActivity, "expected to find human_responded activity")
	assert.Equal(t, models.ActorTypeHuman, foundActivity.ActorType)
	assert.Equal(t, "Responded to message", foundActivity.Summary)

	// Verify details contain the response
	details, err := foundActivity.GetDetails()
	require.NoError(t, err)
	require.NotNil(t, details)
	assert.Equal(t, float64(message.ID), details["inbox_message_id"]) // JSON numbers are float64
	assert.Equal(t, response, details["response"])
}

func TestInboxRespondActivityIncludesMessageContent(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project and ticket
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusHuman,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create inbox message
	inboxRepo := db.NewInboxRepo(database.DB)
	originalQuestion := "Should I use PostgreSQL or SQLite?"
	message := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, originalQuestion, "agent-456")
	err = inboxRepo.Create(message)
	require.NoError(t, err)

	// Respond to the message
	response := "Use SQLite for simplicity"
	err = inboxRepo.Respond(message.ID, response)
	require.NoError(t, err)

	// Log activity with message content included (enhanced version)
	activityRepo := db.NewActivityRepo(database.DB)
	err = activityRepo.LogActionWithDetails(ticket.ID, models.ActionHumanResponded, models.ActorTypeHuman, "",
		"Responded to message",
		map[string]interface{}{
			"inbox_message_id": message.ID,
			"message_type":     string(message.MessageType),
			"message":          originalQuestion,
			"response":         response,
		})
	require.NoError(t, err)

	// Verify activity was logged with full context
	activities, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)

	var foundActivity *models.ActivityLog
	for _, a := range activities {
		if a.Action == models.ActionHumanResponded {
			foundActivity = a
			break
		}
	}

	require.NotNil(t, foundActivity)
	details, err := foundActivity.GetDetails()
	require.NoError(t, err)

	// Verify all expected fields are present
	assert.Equal(t, originalQuestion, details["message"])
	assert.Equal(t, response, details["response"])
	assert.Equal(t, "question", details["message_type"])
}
