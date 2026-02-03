package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// TasksRepo provides database operations for ticket tasks.
type TasksRepo struct {
	db *sql.DB
}

// NewTasksRepo creates a new TasksRepo.
func NewTasksRepo(db *sql.DB) *TasksRepo {
	return &TasksRepo{db: db}
}

// CreateTask adds a new task to a ticket at the next position.
func (r *TasksRepo) CreateTask(ctx context.Context, ticketID int64, description string) (*models.TicketTask, error) {
	if ticketID <= 0 {
		return nil, fmt.Errorf("ticket_id is required")
	}
	if description == "" {
		return nil, fmt.Errorf("description cannot be empty")
	}

	// Get the next position for this ticket
	var maxPos sql.NullInt64
	err := r.db.QueryRowContext(ctx, "SELECT MAX(position) FROM ticket_tasks WHERE ticket_id = ?", ticketID).Scan(&maxPos)
	if err != nil {
		return nil, fmt.Errorf("failed to get max position: %w", err)
	}

	nextPos := 0
	if maxPos.Valid {
		nextPos = int(maxPos.Int64) + 1
	}

	now := time.Now()
	query := `
		INSERT INTO ticket_tasks (ticket_id, position, description, complete, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query, ticketID, nextPos, description, false, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get task id: %w", err)
	}

	return &models.TicketTask{
		ID:          id,
		TicketID:    ticketID,
		Position:    nextPos,
		Description: description,
		Complete:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// ListTasks retrieves all tasks for a ticket ordered by position.
func (r *TasksRepo) ListTasks(ctx context.Context, ticketID int64) ([]*models.TicketTask, error) {
	query := `
		SELECT id, ticket_id, position, description, complete, created_at, updated_at
		FROM ticket_tasks
		WHERE ticket_id = ?
		ORDER BY position
	`
	rows, err := r.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// GetNextIncompleteTask returns the first incomplete task for a ticket.
func (r *TasksRepo) GetNextIncompleteTask(ctx context.Context, ticketID int64) (*models.TicketTask, error) {
	query := `
		SELECT id, ticket_id, position, description, complete, created_at, updated_at
		FROM ticket_tasks
		WHERE ticket_id = ? AND complete = FALSE
		ORDER BY position
		LIMIT 1
	`
	return r.scanOne(r.db.QueryRowContext(ctx, query, ticketID))
}

// CompleteTask marks a task as complete.
func (r *TasksRepo) CompleteTask(ctx context.Context, taskID int64) error {
	if taskID <= 0 {
		return fmt.Errorf("task_id is required")
	}

	now := time.Now()
	query := `UPDATE ticket_tasks SET complete = TRUE, updated_at = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, now, taskID)
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// RemoveTask removes a task and reorders remaining tasks.
func (r *TasksRepo) RemoveTask(ctx context.Context, taskID int64) error {
	if taskID <= 0 {
		return fmt.Errorf("task_id is required")
	}

	// Get the task to find its ticket_id and position
	var ticketID int64
	var position int
	err := r.db.QueryRowContext(ctx, "SELECT ticket_id, position FROM ticket_tasks WHERE id = ?", taskID).Scan(&ticketID, &position)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Delete the task
	_, err = r.db.ExecContext(ctx, "DELETE FROM ticket_tasks WHERE id = ?", taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// Reorder remaining tasks - decrement positions of all tasks after the removed one
	now := time.Now()
	_, err = r.db.ExecContext(ctx, `
		UPDATE ticket_tasks 
		SET position = position - 1, updated_at = ?
		WHERE ticket_id = ? AND position > ?
	`, now, ticketID, position)
	if err != nil {
		return fmt.Errorf("failed to reorder tasks: %w", err)
	}

	return nil
}

// HasIncompleteTasks checks if a ticket has any incomplete tasks.
func (r *TasksRepo) HasIncompleteTasks(ctx context.Context, ticketID int64) (bool, error) {
	query := `SELECT 1 FROM ticket_tasks WHERE ticket_id = ? AND complete = FALSE LIMIT 1`
	var exists int
	err := r.db.QueryRowContext(ctx, query, ticketID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check incomplete tasks: %w", err)
	}
	return true, nil
}

// GetByID retrieves a task by ID.
func (r *TasksRepo) GetByID(ctx context.Context, taskID int64) (*models.TicketTask, error) {
	query := `
		SELECT id, ticket_id, position, description, complete, created_at, updated_at
		FROM ticket_tasks
		WHERE id = ?
	`
	return r.scanOne(r.db.QueryRowContext(ctx, query, taskID))
}

// GetByPosition retrieves a task by ticket ID and position.
func (r *TasksRepo) GetByPosition(ctx context.Context, ticketID int64, position int) (*models.TicketTask, error) {
	query := `
		SELECT id, ticket_id, position, description, complete, created_at, updated_at
		FROM ticket_tasks
		WHERE ticket_id = ? AND position = ?
	`
	return r.scanOne(r.db.QueryRowContext(ctx, query, ticketID, position))
}

// CountIncomplete returns the count of incomplete tasks for a ticket.
func (r *TasksRepo) CountIncomplete(ctx context.Context, ticketID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ticket_tasks WHERE ticket_id = ? AND complete = FALSE", ticketID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count incomplete tasks: %w", err)
	}
	return count, nil
}

func (r *TasksRepo) scanOne(row *sql.Row) (*models.TicketTask, error) {
	var t models.TicketTask
	err := row.Scan(&t.ID, &t.TicketID, &t.Position, &t.Description, &t.Complete, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}
	return &t, nil
}

func (r *TasksRepo) scanMany(rows *sql.Rows) ([]*models.TicketTask, error) {
	var tasks []*models.TicketTask
	for rows.Next() {
		var t models.TicketTask
		err := rows.Scan(&t.ID, &t.TicketID, &t.Position, &t.Description, &t.Complete, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}
	return tasks, nil
}
