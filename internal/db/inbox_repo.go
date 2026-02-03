package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// InboxRepo provides database operations for inbox messages.
type InboxRepo struct {
	db *sql.DB
}

// NewInboxRepo creates a new InboxRepo.
func NewInboxRepo(db *sql.DB) *InboxRepo {
	return &InboxRepo{db: db}
}

// InboxFilter defines filters for listing inbox messages.
type InboxFilter struct {
	TicketID    *int64
	ProjectID   *int64
	ProjectKey  string
	MessageType *models.MessageType
	Pending     bool
	Limit       int
	Offset      int
}

// Create creates a new inbox message.
func (r *InboxRepo) Create(m *models.InboxMessage) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid inbox message: %w", err)
	}

	query := `
		INSERT INTO inbox_messages (ticket_id, message_type, content, from_agent, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := r.db.Exec(query, m.TicketID, m.MessageType, m.Content, nullString(m.FromAgent), FormatTime(now))
	if err != nil {
		return fmt.Errorf("failed to create inbox message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get message id: %w", err)
	}

	m.ID = id
	m.CreatedAt = now
	return nil
}

// GetByID retrieves an inbox message by ID.
func (r *InboxRepo) GetByID(id int64) (*models.InboxMessage, error) {
	query := `
		SELECT m.id, m.ticket_id, m.message_type, m.content, m.from_agent,
			m.response, m.responded_at, m.created_at,
			t.title AS ticket_title, p.key || '-' || t.number AS ticket_key
		FROM inbox_messages m
		JOIN tickets t ON m.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE m.id = ?
	`
	return r.scanOne(r.db.QueryRow(query, id))
}

// List retrieves inbox messages matching the given filter.
func (r *InboxRepo) List(filter InboxFilter) ([]*models.InboxMessage, error) {
	query := `
		SELECT m.id, m.ticket_id, m.message_type, m.content, m.from_agent,
			m.response, m.responded_at, m.created_at,
			t.title AS ticket_title, p.key || '-' || t.number AS ticket_key
		FROM inbox_messages m
		JOIN tickets t ON m.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.TicketID != nil {
		query += " AND m.ticket_id = ?"
		args = append(args, *filter.TicketID)
	}
	if filter.ProjectID != nil {
		query += " AND t.project_id = ?"
		args = append(args, *filter.ProjectID)
	}
	if filter.ProjectKey != "" {
		query += " AND p.key = ?"
		args = append(args, filter.ProjectKey)
	}
	if filter.MessageType != nil {
		query += " AND m.message_type = ?"
		args = append(args, *filter.MessageType)
	}
	if filter.Pending {
		query += " AND m.responded_at IS NULL"
	}

	query += " ORDER BY m.created_at DESC"

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
		return nil, fmt.Errorf("failed to list inbox messages: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// ListPending retrieves all pending (unanswered) inbox messages.
func (r *InboxRepo) ListPending() ([]*models.InboxMessage, error) {
	return r.List(InboxFilter{Pending: true})
}

// Respond adds a response to an inbox message.
func (r *InboxRepo) Respond(id int64, response string) error {
	if response == "" {
		return fmt.Errorf("response cannot be empty")
	}

	now := NowRFC3339()
	query := `UPDATE inbox_messages SET response = ?, responded_at = ? WHERE id = ?`
	result, err := r.db.Exec(query, response, now, id)
	if err != nil {
		return fmt.Errorf("failed to respond to message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// Delete deletes an inbox message.
func (r *InboxRepo) Delete(id int64) error {
	query := `DELETE FROM inbox_messages WHERE id = ?`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// CountPending counts the number of pending messages.
func (r *InboxRepo) CountPending() (int, error) {
	query := `SELECT COUNT(*) FROM inbox_messages WHERE responded_at IS NULL`
	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending messages: %w", err)
	}
	return count, nil
}

// CountPendingByTicket counts the number of pending messages for a ticket.
func (r *InboxRepo) CountPendingByTicket(ticketID int64) (int, error) {
	query := `SELECT COUNT(*) FROM inbox_messages WHERE ticket_id = ? AND responded_at IS NULL`
	var count int
	err := r.db.QueryRow(query, ticketID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending messages: %w", err)
	}
	return count, nil
}

func (r *InboxRepo) scanOne(row *sql.Row) (*models.InboxMessage, error) {
	var m models.InboxMessage
	var fromAgent, response sql.NullString
	var respondedAt sql.NullTime
	var ticketTitle, ticketKey sql.NullString

	err := row.Scan(
		&m.ID, &m.TicketID, &m.MessageType, &m.Content, &fromAgent,
		&response, &respondedAt, &m.CreatedAt,
		&ticketTitle, &ticketKey,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan inbox message: %w", err)
	}

	m.FromAgent = fromAgent.String
	m.Response = response.String
	if respondedAt.Valid {
		m.RespondedAt = &respondedAt.Time
	}
	m.TicketTitle = ticketTitle.String
	m.TicketKey = ticketKey.String
	return &m, nil
}

func (r *InboxRepo) scanMany(rows *sql.Rows) ([]*models.InboxMessage, error) {
	var messages []*models.InboxMessage
	for rows.Next() {
		var m models.InboxMessage
		var fromAgent, response sql.NullString
		var respondedAt sql.NullTime
		var ticketTitle, ticketKey sql.NullString

		err := rows.Scan(
			&m.ID, &m.TicketID, &m.MessageType, &m.Content, &fromAgent,
			&response, &respondedAt, &m.CreatedAt,
			&ticketTitle, &ticketKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inbox message: %w", err)
		}

		m.FromAgent = fromAgent.String
		m.Response = response.String
		if respondedAt.Valid {
			m.RespondedAt = &respondedAt.Time
		}
		m.TicketTitle = ticketTitle.String
		m.TicketKey = ticketKey.String
		messages = append(messages, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inbox messages: %w", err)
	}
	return messages, nil
}
