package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// TicketRepo provides database operations for tickets.
type TicketRepo struct {
	db *sql.DB
}

// NewTicketRepo creates a new TicketRepo.
func NewTicketRepo(db *sql.DB) *TicketRepo {
	return &TicketRepo{db: db}
}

// TicketFilter defines filters for listing tickets.
type TicketFilter struct {
	ProjectID  *int64
	ProjectKey string
	Status     *models.Status
	Priority   *models.Priority
	Complexity *models.Complexity
	ParentID   *int64
	Workable   bool
	Limit      int
	Offset     int
}

// Create creates a new ticket.
func (r *TicketRepo) Create(t *models.Ticket) error {
	// Set defaults - status must be set by caller based on dependency check
	if t.Status == "" {
		t.Status = models.StatusReady
	}
	if t.Priority == "" {
		t.Priority = models.PriorityMedium
	}
	if t.Complexity == "" {
		t.Complexity = models.ComplexityMedium
	}
	if t.MaxRetries == 0 {
		t.MaxRetries = 3
	}

	if err := t.Validate(); err != nil {
		return fmt.Errorf("invalid ticket: %w", err)
	}

	query := `
		INSERT INTO tickets (
			project_id, number, title, description, status, resolution, human_flag_reason,
			priority, complexity, branch_name, retry_count, max_retries,
			parent_ticket_id, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()

	// Number will be set by trigger if 0
	number := t.Number
	if number == 0 {
		// Get next number for project
		var maxNum sql.NullInt64
		err := r.db.QueryRow("SELECT MAX(number) FROM tickets WHERE project_id = ?", t.ProjectID).Scan(&maxNum)
		if err != nil {
			return fmt.Errorf("failed to get next ticket number: %w", err)
		}
		number = int(maxNum.Int64) + 1
	}

	result, err := r.db.Exec(query,
		t.ProjectID, number, t.Title, nullString(t.Description), t.Status, nullResolution(t.Resolution), nullString(t.HumanFlagReason),
		t.Priority, t.Complexity, nullString(t.BranchName), t.RetryCount, t.MaxRetries,
		nullInt64(t.ParentTicketID), now, now, nullTime(t.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to create ticket: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get ticket id: %w", err)
	}

	t.ID = id
	t.Number = number
	t.CreatedAt = now
	t.UpdatedAt = now
	return nil
}

// GetByID retrieves a ticket by ID.
func (r *TicketRepo) GetByID(id int64) (*models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.branch_name,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.id = ?
	`
	return r.scanOne(r.db.QueryRow(query, id))
}

// GetByKey retrieves a ticket by its key (e.g., "WEBAPP-42").
func (r *TicketRepo) GetByKey(projectKey string, number int) (*models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.branch_name,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE p.key = ? AND t.number = ?
	`
	return r.scanOne(r.db.QueryRow(query, projectKey, number))
}

// List retrieves tickets matching the given filter.
func (r *TicketRepo) List(filter TicketFilter) ([]*models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.branch_name,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.ProjectID != nil {
		query += " AND t.project_id = ?"
		args = append(args, *filter.ProjectID)
	}
	if filter.ProjectKey != "" {
		query += " AND p.key = ?"
		args = append(args, filter.ProjectKey)
	}
	if filter.Status != nil {
		query += " AND t.status = ?"
		args = append(args, *filter.Status)
	}
	if filter.Priority != nil {
		query += " AND t.priority = ?"
		args = append(args, *filter.Priority)
	}
	if filter.Complexity != nil {
		query += " AND t.complexity = ?"
		args = append(args, *filter.Complexity)
	}
	if filter.ParentID != nil {
		query += " AND t.parent_ticket_id = ?"
		args = append(args, *filter.ParentID)
	}

	query += ` ORDER BY
		CASE t.priority
			WHEN 'highest' THEN 1
			WHEN 'high' THEN 2
			WHEN 'medium' THEN 3
			WHEN 'low' THEN 4
			WHEN 'lowest' THEN 5
		END,
		t.created_at
	`

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// ListWorkable retrieves all workable tickets (ready status with no unresolved dependencies).
// A dependency is only resolved if its ticket is closed with 'completed' resolution.
func (r *TicketRepo) ListWorkable(filter TicketFilter) ([]*models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.branch_name,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status = 'ready'
		AND NOT EXISTS (
			SELECT 1 FROM ticket_dependencies td
			JOIN tickets dep ON td.depends_on_id = dep.id
			WHERE td.ticket_id = t.id
			AND NOT (dep.status = 'closed' AND dep.resolution = 'completed')
		)
	`
	args := []interface{}{}

	if filter.ProjectID != nil {
		query += " AND t.project_id = ?"
		args = append(args, *filter.ProjectID)
	}
	if filter.ProjectKey != "" {
		query += " AND p.key = ?"
		args = append(args, filter.ProjectKey)
	}
	if filter.Complexity != nil {
		query += " AND t.complexity = ?"
		args = append(args, *filter.Complexity)
	}

	query += ` ORDER BY
		CASE t.priority
			WHEN 'highest' THEN 1
			WHEN 'high' THEN 2
			WHEN 'medium' THEN 3
			WHEN 'low' THEN 4
			WHEN 'lowest' THEN 5
		END,
		t.created_at
	`

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workable tickets: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// Update updates a ticket.
func (r *TicketRepo) Update(t *models.Ticket) error {
	if t.ID <= 0 {
		return fmt.Errorf("ticket id is required")
	}

	query := `
		UPDATE tickets SET
			title = ?, description = ?, status = ?, resolution = ?, human_flag_reason = ?,
			priority = ?, complexity = ?, branch_name = ?,
			retry_count = ?, max_retries = ?, parent_ticket_id = ?, completed_at = ?
		WHERE id = ?
	`
	result, err := r.db.Exec(query,
		t.Title, nullString(t.Description), t.Status, nullResolution(t.Resolution), nullString(t.HumanFlagReason),
		t.Priority, t.Complexity, nullString(t.BranchName),
		t.RetryCount, t.MaxRetries, nullInt64(t.ParentTicketID), nullTime(t.CompletedAt),
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("ticket not found")
	}

	return nil
}

// UpdateStatus updates the status of a ticket.
func (r *TicketRepo) UpdateStatus(id int64, status models.Status) error {
	query := `UPDATE tickets SET status = ? WHERE id = ?`
	result, err := r.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update ticket status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("ticket not found")
	}

	return nil
}

// IncrementRetryCount increments the retry count for a ticket.
func (r *TicketRepo) IncrementRetryCount(id int64) error {
	query := `UPDATE tickets SET retry_count = retry_count + 1 WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// Delete deletes a ticket by ID.
func (r *TicketRepo) Delete(id int64) error {
	query := `DELETE FROM tickets WHERE id = ?`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete ticket: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("ticket not found")
	}

	return nil
}

// GetChildren retrieves all child tickets of a parent ticket.
func (r *TicketRepo) GetChildren(parentID int64) ([]*models.Ticket, error) {
	filter := TicketFilter{ParentID: &parentID}
	return r.List(filter)
}

// CountByStatus counts tickets by status for a project.
func (r *TicketRepo) CountByStatus(projectID int64) (map[models.Status]int, error) {
	query := `SELECT status, COUNT(*) FROM tickets WHERE project_id = ? GROUP BY status`
	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to count tickets: %w", err)
	}
	defer rows.Close()

	counts := make(map[models.Status]int)
	for rows.Next() {
		var status models.Status
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *TicketRepo) scanOne(row *sql.Row) (*models.Ticket, error) {
	var t models.Ticket
	var desc, resolution, humanFlag, branch sql.NullString
	var parentID sql.NullInt64
	var completedAt sql.NullTime

	err := row.Scan(
		&t.ID, &t.ProjectID, &t.Number, &t.Title, &desc, &t.Status,
		&resolution, &humanFlag, &t.Priority, &t.Complexity, &branch,
		&t.RetryCount, &t.MaxRetries, &parentID,
		&t.CreatedAt, &t.UpdatedAt, &completedAt,
		&t.ProjectKey,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan ticket: %w", err)
	}

	t.Description = desc.String
	t.HumanFlagReason = humanFlag.String
	t.BranchName = branch.String
	if resolution.Valid {
		res := models.Resolution(resolution.String)
		t.Resolution = &res
	}
	if parentID.Valid {
		t.ParentTicketID = &parentID.Int64
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	t.TicketKey = fmt.Sprintf("%s-%d", t.ProjectKey, t.Number)
	return &t, nil
}

func (r *TicketRepo) scanMany(rows *sql.Rows) ([]*models.Ticket, error) {
	var tickets []*models.Ticket
	for rows.Next() {
		var t models.Ticket
		var desc, resolution, humanFlag, branch sql.NullString
		var parentID sql.NullInt64
		var completedAt sql.NullTime

		err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Number, &t.Title, &desc, &t.Status,
			&resolution, &humanFlag, &t.Priority, &t.Complexity, &branch,
			&t.RetryCount, &t.MaxRetries, &parentID,
			&t.CreatedAt, &t.UpdatedAt, &completedAt,
			&t.ProjectKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}

		t.Description = desc.String
		t.HumanFlagReason = humanFlag.String
		t.BranchName = branch.String
		if resolution.Valid {
			res := models.Resolution(resolution.String)
			t.Resolution = &res
		}
		if parentID.Valid {
			t.ParentTicketID = &parentID.Int64
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		t.TicketKey = fmt.Sprintf("%s-%d", t.ProjectKey, t.Number)
		tickets = append(tickets, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tickets: %w", err)
	}
	return tickets, nil
}

// Helper functions for nullable types
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullResolution(r *models.Resolution) sql.NullString {
	if r == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(*r), Valid: true}
}
