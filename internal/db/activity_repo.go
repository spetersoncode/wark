package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/spetersoncode/wark/internal/models"
)

// ActivityRepo provides database operations for activity log entries.
type ActivityRepo struct {
	db *sql.DB
}

// NewActivityRepo creates a new ActivityRepo.
func NewActivityRepo(db *sql.DB) *ActivityRepo {
	return &ActivityRepo{db: db}
}

// ActivityFilter defines filters for listing activity log entries.
type ActivityFilter struct {
	TicketID  *int64
	Action    *models.Action
	ActorType *models.ActorType
	ActorID   string
	Since     *time.Time
	Limit     int
	Offset    int
}

// Create creates a new activity log entry.
func (r *ActivityRepo) Create(a *models.ActivityLog) error {
	if err := a.Validate(); err != nil {
		return fmt.Errorf("invalid activity log: %w", err)
	}

	query := `
		INSERT INTO activity_log (ticket_id, action, actor_type, actor_id, details, summary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := r.db.Exec(query,
		a.TicketID, a.Action, a.ActorType, nullString(a.ActorID),
		nullString(a.Details), nullString(a.Summary), FormatTime(now),
	)
	if err != nil {
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get activity log id: %w", err)
	}

	a.ID = id
	a.CreatedAt = now
	return nil
}

// GetByID retrieves an activity log entry by ID.
func (r *ActivityRepo) GetByID(id int64) (*models.ActivityLog, error) {
	query := `
		SELECT a.id, a.ticket_id, a.action, a.actor_type, a.actor_id,
			a.details, a.summary, a.created_at,
			p.key || '-' || t.number AS ticket_key
		FROM activity_log a
		JOIN tickets t ON a.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE a.id = ?
	`
	return r.scanOne(r.db.QueryRow(query, id))
}

// List retrieves activity log entries matching the given filter.
func (r *ActivityRepo) List(filter ActivityFilter) ([]*models.ActivityLog, error) {
	query := `
		SELECT a.id, a.ticket_id, a.action, a.actor_type, a.actor_id,
			a.details, a.summary, a.created_at,
			p.key || '-' || t.number AS ticket_key
		FROM activity_log a
		JOIN tickets t ON a.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.TicketID != nil {
		query += " AND a.ticket_id = ?"
		args = append(args, *filter.TicketID)
	}
	if filter.Action != nil {
		query += " AND a.action = ?"
		args = append(args, *filter.Action)
	}
	if filter.ActorType != nil {
		query += " AND a.actor_type = ?"
		args = append(args, *filter.ActorType)
	}
	if filter.ActorID != "" {
		query += " AND a.actor_id = ?"
		args = append(args, filter.ActorID)
	}
	if filter.Since != nil {
		query += " AND a.created_at >= ?"
		args = append(args, FormatTime(*filter.Since))
	}

	query += " ORDER BY a.created_at DESC"

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
		return nil, fmt.Errorf("failed to list activity log: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// ListByTicket retrieves all activity log entries for a ticket.
func (r *ActivityRepo) ListByTicket(ticketID int64, limit int) ([]*models.ActivityLog, error) {
	filter := ActivityFilter{
		TicketID: &ticketID,
		Limit:    limit,
	}
	return r.List(filter)
}

// GetLatestByTicket retrieves the most recent activity log entry for a ticket.
func (r *ActivityRepo) GetLatestByTicket(ticketID int64) (*models.ActivityLog, error) {
	query := `
		SELECT a.id, a.ticket_id, a.action, a.actor_type, a.actor_id,
			a.details, a.summary, a.created_at,
			p.key || '-' || t.number AS ticket_key
		FROM activity_log a
		JOIN tickets t ON a.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE a.ticket_id = ?
		ORDER BY a.created_at DESC
		LIMIT 1
	`
	return r.scanOne(r.db.QueryRow(query, ticketID))
}

// CountByTicket counts activity log entries for a ticket.
func (r *ActivityRepo) CountByTicket(ticketID int64) (int, error) {
	query := `SELECT COUNT(*) FROM activity_log WHERE ticket_id = ?`
	var count int
	err := r.db.QueryRow(query, ticketID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count activity log: %w", err)
	}
	return count, nil
}

// LogAction is a convenience method to create an activity log entry.
func (r *ActivityRepo) LogAction(ticketID int64, action models.Action, actorType models.ActorType, actorID, summary string) error {
	log := models.NewActivityLog(ticketID, action, actorType, actorID, summary)
	return r.Create(log)
}

// LogActionWithDetails is a convenience method to create an activity log entry with details.
func (r *ActivityRepo) LogActionWithDetails(ticketID int64, action models.Action, actorType models.ActorType, actorID, summary string, details map[string]interface{}) error {
	log, err := models.NewActivityLogWithDetails(ticketID, action, actorType, actorID, summary, details)
	if err != nil {
		return err
	}
	return r.Create(log)
}

func (r *ActivityRepo) scanOne(row *sql.Row) (*models.ActivityLog, error) {
	var a models.ActivityLog
	var actorID, details, summary sql.NullString
	var ticketKey sql.NullString

	err := row.Scan(
		&a.ID, &a.TicketID, &a.Action, &a.ActorType, &actorID,
		&details, &summary, &a.CreatedAt, &ticketKey,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan activity log: %w", err)
	}

	a.ActorID = actorID.String
	a.Details = details.String
	a.Summary = summary.String
	a.TicketKey = ticketKey.String
	return &a, nil
}

func (r *ActivityRepo) scanMany(rows *sql.Rows) ([]*models.ActivityLog, error) {
	var logs []*models.ActivityLog
	for rows.Next() {
		var a models.ActivityLog
		var actorID, details, summary sql.NullString
		var ticketKey sql.NullString

		err := rows.Scan(
			&a.ID, &a.TicketID, &a.Action, &a.ActorType, &actorID,
			&details, &summary, &a.CreatedAt, &ticketKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity log: %w", err)
		}

		a.ActorID = actorID.String
		a.Details = details.String
		a.Summary = summary.String
		a.TicketKey = ticketKey.String
		logs = append(logs, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activity log: %w", err)
	}
	return logs, nil
}
