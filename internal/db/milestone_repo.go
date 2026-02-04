package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/spetersoncode/wark/internal/models"
)

// MilestoneRepo provides database operations for milestones.
type MilestoneRepo struct {
	db *sql.DB
}

// NewMilestoneRepo creates a new MilestoneRepo.
func NewMilestoneRepo(db *sql.DB) *MilestoneRepo {
	return &MilestoneRepo{db: db}
}

// Create creates a new milestone.
func (r *MilestoneRepo) Create(projectID int64, key, name, goal string, targetDate *time.Time) (*models.Milestone, error) {
	m := &models.Milestone{
		ProjectID:  projectID,
		Key:        key,
		Name:       name,
		Goal:       goal,
		TargetDate: targetDate,
		Status:     models.MilestoneStatusOpen,
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid milestone: %w", err)
	}

	now := time.Now()
	nowStr := FormatTime(now)

	query := `
		INSERT INTO milestones (project_id, key, name, goal, target_date, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query, projectID, key, name, goal, FormatTimePtr(targetDate), m.Status, nowStr, nowStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create milestone: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get milestone id: %w", err)
	}

	m.ID = id
	m.CreatedAt = now
	m.UpdatedAt = now

	return m, nil
}

// Get retrieves a milestone by ID.
func (r *MilestoneRepo) Get(id int64) (*models.Milestone, error) {
	query := `
		SELECT m.id, m.project_id, m.key, m.name, m.goal, m.target_date, m.status, m.created_at, m.updated_at,
		       p.key AS project_key
		FROM milestones m
		JOIN projects p ON p.id = m.project_id
		WHERE m.id = ?
	`
	return r.scanOne(r.db.QueryRow(query, id))
}

// GetByKey retrieves a milestone by project key and milestone key.
func (r *MilestoneRepo) GetByKey(projectKey, milestoneKey string) (*models.Milestone, error) {
	query := `
		SELECT m.id, m.project_id, m.key, m.name, m.goal, m.target_date, m.status, m.created_at, m.updated_at,
		       p.key AS project_key
		FROM milestones m
		JOIN projects p ON p.id = m.project_id
		WHERE p.key = ? AND m.key = ?
	`
	return r.scanOne(r.db.QueryRow(query, projectKey, milestoneKey))
}

// List retrieves milestones with optional project filter and ticket stats.
func (r *MilestoneRepo) List(projectID *int64) ([]models.MilestoneWithStats, error) {
	query := `
		SELECT m.id, m.project_id, m.key, m.name, m.goal, m.target_date, m.status, m.created_at, m.updated_at,
		       p.key AS project_key,
		       COUNT(t.id) AS ticket_count,
		       SUM(CASE WHEN t.status = 'closed' AND t.resolution = 'completed' THEN 1 ELSE 0 END) AS completed_count
		FROM milestones m
		JOIN projects p ON p.id = m.project_id
		LEFT JOIN tickets t ON t.milestone_id = m.id
	`

	var args []interface{}
	if projectID != nil {
		query += " WHERE m.project_id = ?"
		args = append(args, *projectID)
	}

	query += `
		GROUP BY m.id
		ORDER BY m.status, m.target_date NULLS LAST, m.name
	`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list milestones: %w", err)
	}
	defer rows.Close()

	return r.scanManyWithStats(rows)
}

// Update updates a milestone with the given fields.
// Supported fields: name, goal, target_date, status
func (r *MilestoneRepo) Update(id int64, updates map[string]any) (*models.Milestone, error) {
	if len(updates) == 0 {
		return r.Get(id)
	}

	// Validate fields being updated
	if status, ok := updates["status"]; ok {
		if err := models.ValidateMilestoneStatus(status.(string)); err != nil {
			return nil, err
		}
	}

	// Build the update query dynamically
	setClauses := ""
	var args []interface{}

	allowedFields := map[string]bool{
		"name":        true,
		"goal":        true,
		"target_date": true,
		"status":      true,
	}

	for field, value := range updates {
		if !allowedFields[field] {
			return nil, fmt.Errorf("cannot update field: %s", field)
		}
		if setClauses != "" {
			setClauses += ", "
		}
		setClauses += field + " = ?"

		// Handle target_date specially for time formatting
		if field == "target_date" {
			if t, ok := value.(*time.Time); ok {
				args = append(args, FormatTimePtr(t))
			} else if t, ok := value.(time.Time); ok {
				args = append(args, FormatTime(t))
			} else {
				args = append(args, value)
			}
		} else {
			args = append(args, value)
		}
	}

	// Always update updated_at
	setClauses += ", updated_at = ?"
	args = append(args, NowRFC3339())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE milestones SET %s WHERE id = ?", setClauses)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update milestone: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("milestone not found")
	}

	return r.Get(id)
}

// Delete deletes a milestone by ID.
// This will set milestone_id to NULL on any linked tickets.
func (r *MilestoneRepo) Delete(id int64) error {
	// First, unlink any tickets from this milestone
	_, err := r.db.Exec("UPDATE tickets SET milestone_id = NULL WHERE milestone_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to unlink tickets: %w", err)
	}

	result, err := r.db.Exec("DELETE FROM milestones WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete milestone: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("milestone not found")
	}

	return nil
}

// GetLinkedTickets retrieves all tickets linked to a milestone.
func (r *MilestoneRepo) GetLinkedTickets(milestoneID int64) ([]models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status, t.resolution,
		       t.human_flag_reason, t.priority, t.complexity, t.branch_name, t.retry_count,
		       t.max_retries, t.parent_ticket_id, t.created_at, t.updated_at, t.completed_at,
		       p.key AS project_key
		FROM tickets t
		JOIN projects p ON p.id = t.project_id
		WHERE t.milestone_id = ?
		ORDER BY t.status, t.priority DESC, t.number
	`

	rows, err := r.db.Query(query, milestoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to get linked tickets: %w", err)
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		var desc, humanFlag, branchName sql.NullString
		var resolution sql.NullString
		var parentID sql.NullInt64
		var completedAt sql.NullTime

		err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Number, &t.Title, &desc, &t.Status, &resolution,
			&humanFlag, &t.Priority, &t.Complexity, &branchName, &t.RetryCount,
			&t.MaxRetries, &parentID, &t.CreatedAt, &t.UpdatedAt, &completedAt,
			&t.ProjectKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}

		t.Description = desc.String
		t.HumanFlagReason = humanFlag.String
		t.BranchName = branchName.String
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

		tickets = append(tickets, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tickets: %w", err)
	}

	return tickets, nil
}

// Exists checks if a milestone with the given project ID and key exists.
func (r *MilestoneRepo) Exists(projectID int64, key string) (bool, error) {
	query := `SELECT 1 FROM milestones WHERE project_id = ? AND key = ? LIMIT 1`
	var exists int
	err := r.db.QueryRow(query, projectID, key).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check milestone existence: %w", err)
	}
	return true, nil
}

func (r *MilestoneRepo) scanOne(row *sql.Row) (*models.Milestone, error) {
	var m models.Milestone
	var goal sql.NullString
	var targetDate sql.NullTime

	err := row.Scan(
		&m.ID, &m.ProjectID, &m.Key, &m.Name, &goal, &targetDate,
		&m.Status, &m.CreatedAt, &m.UpdatedAt, &m.ProjectKey,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan milestone: %w", err)
	}

	m.Goal = goal.String
	if targetDate.Valid {
		m.TargetDate = &targetDate.Time
	}

	return &m, nil
}

func (r *MilestoneRepo) scanManyWithStats(rows *sql.Rows) ([]models.MilestoneWithStats, error) {
	var milestones []models.MilestoneWithStats

	for rows.Next() {
		var ms models.MilestoneWithStats
		var goal sql.NullString
		var targetDate sql.NullTime
		var completedCount sql.NullInt64

		err := rows.Scan(
			&ms.ID, &ms.ProjectID, &ms.Key, &ms.Name, &goal, &targetDate,
			&ms.Status, &ms.CreatedAt, &ms.UpdatedAt, &ms.ProjectKey,
			&ms.TicketCount, &completedCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan milestone: %w", err)
		}

		ms.Goal = goal.String
		if targetDate.Valid {
			ms.TargetDate = &targetDate.Time
		}
		ms.CompletedCount = int(completedCount.Int64)

		// Calculate completion percentage
		if ms.TicketCount > 0 {
			ms.CompletionPct = float64(ms.CompletedCount) / float64(ms.TicketCount) * 100
		}

		milestones = append(milestones, ms)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating milestones: %w", err)
	}

	return milestones, nil
}
