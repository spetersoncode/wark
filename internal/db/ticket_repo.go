package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/spetersoncode/wark/internal/models"
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
	ProjectID    *int64
	ProjectKey   string
	Status       *models.Status
	Priority     *models.Priority
	Complexity   *models.Complexity
	Type         *models.TicketType
	ParentID *int64
	Workable bool
	Limit        int
	Offset       int
}

// AutoReleaseExpiredClaims finds tickets with expired claims and releases them back to 'ready' status.
// This is called automatically on query operations to ensure tickets don't stay orphaned.
// Returns the number of tickets released.
func (r *TicketRepo) AutoReleaseExpiredClaims() (int64, error) {
	now := NowRFC3339()

	// Find tickets that are working with expired active claims
	// We need to:
	// 1. Update the claim status to 'expired'
	// 2. Update the ticket status to 'ready' (or 'human' if max retries exceeded)
	// 3. Increment the retry count

	// First, get all affected ticket/claim pairs
	query := `
		SELECT t.id, t.retry_count, t.max_retries, c.id AS claim_id
		FROM tickets t
		JOIN claims c ON c.ticket_id = t.id
		WHERE t.status = 'working'
		AND c.status = 'active'
		AND c.expires_at <= ?
	`
	rows, err := r.db.Query(query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to find expired claims: %w", err)
	}
	defer rows.Close()

	type expiredTicket struct {
		ticketID   int64
		claimID    int64
		retryCount int
		maxRetries int
	}
	var expired []expiredTicket

	for rows.Next() {
		var et expiredTicket
		if err := rows.Scan(&et.ticketID, &et.retryCount, &et.maxRetries, &et.claimID); err != nil {
			return 0, fmt.Errorf("failed to scan expired ticket: %w", err)
		}
		expired = append(expired, et)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating expired tickets: %w", err)
	}

	if len(expired) == 0 {
		return 0, nil
	}

	// Process each expired ticket
	for _, et := range expired {
		newRetryCount := et.retryCount + 1
		newStatus := models.StatusReady
		humanFlagReason := ""

		// Escalate to human if max retries exceeded
		if newRetryCount >= et.maxRetries {
			newStatus = models.StatusHuman
			humanFlagReason = "max_retries_exceeded"
		}

		// Update claim status
		_, err := r.db.Exec(`UPDATE claims SET status = 'expired', released_at = ? WHERE id = ?`, now, et.claimID)
		if err != nil {
			return 0, fmt.Errorf("failed to expire claim: %w", err)
		}

		// Update ticket status and retry count
		if humanFlagReason != "" {
			_, err = r.db.Exec(`UPDATE tickets SET status = ?, retry_count = ?, human_flag_reason = ? WHERE id = ?`,
				newStatus, newRetryCount, humanFlagReason, et.ticketID)
		} else {
			_, err = r.db.Exec(`UPDATE tickets SET status = ?, retry_count = ? WHERE id = ?`,
				newStatus, newRetryCount, et.ticketID)
		}
		if err != nil {
			return 0, fmt.Errorf("failed to update ticket: %w", err)
		}

		// Log activity (best effort - don't fail if this errors)
		summary := "Claim auto-expired"
		if newStatus == models.StatusHuman {
			summary = fmt.Sprintf("Claim auto-expired - escalated to human (retry %d/%d)", newRetryCount, et.maxRetries)
		}
		r.db.Exec(`
			INSERT INTO activity_log (ticket_id, action, actor_type, actor_id, summary, details, created_at)
			VALUES (?, 'expired', 'system', '', ?, '{}', ?)
		`, et.ticketID, summary, now)
	}

	return int64(len(expired)), nil
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
	if t.Type == "" {
		t.Type = models.TicketTypeTask
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
			priority, complexity, ticket_type, worktree, role_id, retry_count, max_retries,
			parent_ticket_id, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	nowStr := FormatTime(now)

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
		t.Priority, t.Complexity, t.Type, nullString(t.Worktree), nullInt64(t.RoleID), t.RetryCount, t.MaxRetries,
		nullInt64(t.ParentTicketID), nowStr, nowStr, FormatTimePtr(t.CompletedAt),
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
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.ticket_type, t.worktree, t.role_id,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key,
			r.name AS role_name
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		LEFT JOIN roles r ON t.role_id = r.id
		WHERE t.id = ?
	`
	return r.scanOne(r.db.QueryRow(query, id))
}

// GetByKey retrieves a ticket by its key (e.g., "WEBAPP-42").
func (r *TicketRepo) GetByKey(projectKey string, number int) (*models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.ticket_type, t.worktree, t.role_id,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key,
			r.name AS role_name
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		LEFT JOIN roles r ON t.role_id = r.id
		WHERE p.key = ? AND t.number = ?
	`
	return r.scanOne(r.db.QueryRow(query, projectKey, number))
}

// List retrieves tickets matching the given filter.
// It automatically releases any expired claims before querying.
func (r *TicketRepo) List(filter TicketFilter) ([]*models.Ticket, error) {
	// Auto-release expired claims to ensure accurate status
	if _, err := r.AutoReleaseExpiredClaims(); err != nil {
		// Log but don't fail - the query can still proceed
		// In production, you'd want proper logging here
		_ = err
	}

	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.ticket_type, t.worktree, t.role_id,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key,
			ro.name AS role_name
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		LEFT JOIN roles ro ON t.role_id = ro.id
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
	if filter.Type != nil {
		query += " AND t.ticket_type = ?"
		args = append(args, *filter.Type)
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
// It automatically releases any expired claims before querying, which may make
// previously claimed tickets workable again.
// Note: Epics are excluded from workable list - work through child tickets instead.
func (r *TicketRepo) ListWorkable(filter TicketFilter) ([]*models.Ticket, error) {
	// Auto-release expired claims - this is especially important for ListWorkable
	// since it makes orphaned tickets available for work again
	if _, err := r.AutoReleaseExpiredClaims(); err != nil {
		// Log but don't fail - the query can still proceed
		_ = err
	}

	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.ticket_type, t.worktree, t.role_id,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key,
			ro.name AS role_name
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		LEFT JOIN roles ro ON t.role_id = ro.id
		WHERE t.status = 'ready'
		AND t.ticket_type != 'epic'
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
			priority = ?, complexity = ?, ticket_type = ?, worktree = ?, role_id = ?,
			retry_count = ?, max_retries = ?, parent_ticket_id = ?, completed_at = ?
		WHERE id = ?
	`

	result, err := r.db.Exec(query,
		t.Title, nullString(t.Description), t.Status, nullResolution(t.Resolution), nullString(t.HumanFlagReason),
		t.Priority, t.Complexity, t.Type, nullString(t.Worktree), nullInt64(t.RoleID),
		t.RetryCount, t.MaxRetries, nullInt64(t.ParentTicketID), FormatTimePtr(t.CompletedAt),
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

// GetEpicChildren retrieves all child tickets of an epic.
// This is an alias for GetChildren but clarifies the semantic intent.
func (r *TicketRepo) GetEpicChildren(epicID int64) ([]*models.Ticket, error) {
	return r.GetChildren(epicID)
}

// Search searches tickets by key, title, or description.
func (r *TicketRepo) Search(query string, limit int) ([]*models.Ticket, error) {
	if query == "" {
		return []*models.Ticket{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	// Search pattern for LIKE matching
	pattern := "%" + query + "%"

	sqlQuery := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.ticket_type, t.worktree, t.role_id,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key,
			ro.name AS role_name
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		LEFT JOIN roles ro ON t.role_id = ro.id
		WHERE (p.key || '-' || t.number) LIKE ? COLLATE NOCASE
		   OR t.title LIKE ? COLLATE NOCASE
		   OR t.description LIKE ? COLLATE NOCASE
		ORDER BY
			CASE
				WHEN (p.key || '-' || t.number) LIKE ? COLLATE NOCASE THEN 1
				WHEN t.title LIKE ? COLLATE NOCASE THEN 2
				ELSE 3
			END,
			t.updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(sqlQuery, pattern, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search tickets: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
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
	var desc, resolution, humanFlag, ticketType, worktree, roleName sql.NullString
	var parentID, roleID sql.NullInt64
	var completedAt sql.NullTime

	err := row.Scan(
		&t.ID, &t.ProjectID, &t.Number, &t.Title, &desc, &t.Status,
		&resolution, &humanFlag, &t.Priority, &t.Complexity, &ticketType, &worktree, &roleID,
		&t.RetryCount, &t.MaxRetries, &parentID,
		&t.CreatedAt, &t.UpdatedAt, &completedAt,
		&t.ProjectKey, &roleName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan ticket: %w", err)
	}

	t.Description = desc.String
	t.HumanFlagReason = humanFlag.String
	t.Type = models.TicketType(ticketType.String)
	if t.Type == "" {
		t.Type = models.TicketTypeTask // Default to task
	}
	t.Worktree = worktree.String
	if roleID.Valid {
		t.RoleID = &roleID.Int64
	}
	if roleName.Valid {
		t.RoleName = roleName.String
	}
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
		var desc, resolution, humanFlag, ticketType, worktree, roleName sql.NullString
		var parentID, roleID sql.NullInt64
		var completedAt sql.NullTime

		err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Number, &t.Title, &desc, &t.Status,
			&resolution, &humanFlag, &t.Priority, &t.Complexity, &ticketType, &worktree, &roleID,
			&t.RetryCount, &t.MaxRetries, &parentID,
			&t.CreatedAt, &t.UpdatedAt, &completedAt,
			&t.ProjectKey, &roleName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}

		t.Description = desc.String
		t.HumanFlagReason = humanFlag.String
		t.Type = models.TicketType(ticketType.String)
		if t.Type == "" {
			t.Type = models.TicketTypeTask // Default to task
		}
		t.Worktree = worktree.String
		if roleID.Valid {
			t.RoleID = &roleID.Int64
		}
		if roleName.Valid {
			t.RoleName = roleName.String
		}
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
