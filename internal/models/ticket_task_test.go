package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTicketTask(t *testing.T) {
	task := NewTicketTask(1, 0, "Implement feature")

	assert.Equal(t, int64(1), task.TicketID)
	assert.Equal(t, 0, task.Position)
	assert.Equal(t, "Implement feature", task.Description)
	assert.False(t, task.Complete)
	assert.False(t, task.CreatedAt.IsZero())
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTicketTaskValidate_Valid(t *testing.T) {
	tests := []struct {
		name string
		task *TicketTask
	}{
		{
			name: "valid task at position 0",
			task: &TicketTask{
				TicketID:    1,
				Position:    0,
				Description: "First task",
			},
		},
		{
			name: "valid task at position 5",
			task: &TicketTask{
				TicketID:    42,
				Position:    5,
				Description: "Another task",
			},
		},
		{
			name: "valid completed task",
			task: &TicketTask{
				TicketID:    1,
				Position:    0,
				Description: "Done task",
				Complete:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestTicketTaskValidate_EmptyDescription(t *testing.T) {
	task := &TicketTask{
		TicketID:    1,
		Position:    0,
		Description: "",
	}

	err := task.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "description cannot be empty")
}

func TestTicketTaskValidate_NegativePosition(t *testing.T) {
	task := &TicketTask{
		TicketID:    1,
		Position:    -1,
		Description: "Some task",
	}

	err := task.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "position cannot be negative")
}

func TestTicketTaskValidate_MissingTicketID(t *testing.T) {
	task := &TicketTask{
		TicketID:    0,
		Position:    0,
		Description: "Some task",
	}

	err := task.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket_id is required")
}

func TestTicketTaskValidate_NegativeTicketID(t *testing.T) {
	task := &TicketTask{
		TicketID:    -5,
		Position:    0,
		Description: "Some task",
	}

	err := task.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket_id is required")
}

func TestNewTicketTaskTimestamps(t *testing.T) {
	before := time.Now()
	task := NewTicketTask(1, 0, "Test task")
	after := time.Now()

	// Timestamps should be between before and after
	assert.True(t, task.CreatedAt.After(before) || task.CreatedAt.Equal(before))
	assert.True(t, task.CreatedAt.Before(after) || task.CreatedAt.Equal(after))
	assert.Equal(t, task.CreatedAt, task.UpdatedAt)
}
