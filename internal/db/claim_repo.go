package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// ClaimRepo provides database operations for claims.
type ClaimRepo struct {
	db *sql.DB
}

// NewClaimRepo creates a new ClaimRepo.
func NewClaimRepo(db *sql.DB) *ClaimRepo {
	return &ClaimRepo{db: db}
}

// Create creates a new claim.
func (r *ClaimRepo) Create(c *models.Claim) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("invalid claim: %w", err)
	}

	query := `
		INSERT INTO claims (ticket_id, worker_id, claimed_at, expires_at, status)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, c.TicketID, c.WorkerID, c.ClaimedAt, c.ExpiresAt, c.Status)
	if err != nil {
		return fmt.Errorf("failed to create claim: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get claim id: %w", err)
	}

	c.ID = id
	return nil
}

// GetByID retrieves a claim by ID.
func (r *ClaimRepo) GetByID(id int64) (*models.Claim, error) {
	query := `
		SELECT c.id, c.ticket_id, c.worker_id, c.claimed_at, c.expires_at,
			c.released_at, c.status, t.title AS ticket_title,
			p.key || '-' || t.number AS ticket_key
		FROM claims c
		JOIN tickets t ON c.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE c.id = ?
	`
	return r.scanOne(r.db.QueryRow(query, id))
}

// GetActiveByTicketID retrieves the active claim for a ticket.
func (r *ClaimRepo) GetActiveByTicketID(ticketID int64) (*models.Claim, error) {
	query := `
		SELECT c.id, c.ticket_id, c.worker_id, c.claimed_at, c.expires_at,
			c.released_at, c.status, t.title AS ticket_title,
			p.key || '-' || t.number AS ticket_key
		FROM claims c
		JOIN tickets t ON c.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE c.ticket_id = ? AND c.status = 'active' AND c.expires_at > ?
	`
	return r.scanOne(r.db.QueryRow(query, ticketID, time.Now()))
}

// GetActiveByWorkerID retrieves all active claims for a worker.
func (r *ClaimRepo) GetActiveByWorkerID(workerID string) ([]*models.Claim, error) {
	query := `
		SELECT c.id, c.ticket_id, c.worker_id, c.claimed_at, c.expires_at,
			c.released_at, c.status, t.title AS ticket_title,
			p.key || '-' || t.number AS ticket_key
		FROM claims c
		JOIN tickets t ON c.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE c.worker_id = ? AND c.status = 'active' AND c.expires_at > ?
		ORDER BY c.claimed_at
	`
	rows, err := r.db.Query(query, workerID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get active claims: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// ListActive retrieves all active claims.
func (r *ClaimRepo) ListActive() ([]*models.Claim, error) {
	query := `
		SELECT c.id, c.ticket_id, c.worker_id, c.claimed_at, c.expires_at,
			c.released_at, c.status, t.title AS ticket_title,
			p.key || '-' || t.number AS ticket_key,
			CAST((julianday(c.expires_at) - julianday('now')) * 24 * 60 AS INTEGER) AS minutes_remaining
		FROM claims c
		JOIN tickets t ON c.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE c.status = 'active' AND c.expires_at > ?
		ORDER BY c.expires_at
	`
	rows, err := r.db.Query(query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to list active claims: %w", err)
	}
	defer rows.Close()

	return r.scanManyWithMinutes(rows)
}

// ListExpired retrieves all expired claims that are still marked as active.
func (r *ClaimRepo) ListExpired() ([]*models.Claim, error) {
	query := `
		SELECT c.id, c.ticket_id, c.worker_id, c.claimed_at, c.expires_at,
			c.released_at, c.status, t.title AS ticket_title,
			p.key || '-' || t.number AS ticket_key
		FROM claims c
		JOIN tickets t ON c.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE c.status = 'active' AND c.expires_at <= ?
		ORDER BY c.expires_at
	`
	rows, err := r.db.Query(query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to list expired claims: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// ListByTicketID retrieves all claims for a ticket.
func (r *ClaimRepo) ListByTicketID(ticketID int64) ([]*models.Claim, error) {
	query := `
		SELECT c.id, c.ticket_id, c.worker_id, c.claimed_at, c.expires_at,
			c.released_at, c.status, t.title AS ticket_title,
			p.key || '-' || t.number AS ticket_key
		FROM claims c
		JOIN tickets t ON c.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE c.ticket_id = ?
		ORDER BY c.claimed_at DESC
	`
	rows, err := r.db.Query(query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to list claims: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// Release releases a claim.
func (r *ClaimRepo) Release(id int64, status models.ClaimStatus) error {
	now := time.Now()
	query := `UPDATE claims SET status = ?, released_at = ? WHERE id = ?`
	result, err := r.db.Exec(query, status, now, id)
	if err != nil {
		return fmt.Errorf("failed to release claim: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("claim not found")
	}

	return nil
}

// ExpireAll marks all expired active claims as expired.
func (r *ClaimRepo) ExpireAll() (int64, error) {
	now := time.Now()
	query := `UPDATE claims SET status = 'expired', released_at = ? WHERE status = 'active' AND expires_at <= ?`
	result, err := r.db.Exec(query, now, now)
	if err != nil {
		return 0, fmt.Errorf("failed to expire claims: %w", err)
	}
	return result.RowsAffected()
}

// HasActiveClaim checks if a ticket has an active claim.
func (r *ClaimRepo) HasActiveClaim(ticketID int64) (bool, error) {
	query := `SELECT 1 FROM claims WHERE ticket_id = ? AND status = 'active' AND expires_at > ? LIMIT 1`
	var exists int
	err := r.db.QueryRow(query, ticketID, time.Now()).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check active claim: %w", err)
	}
	return true, nil
}

func (r *ClaimRepo) scanOne(row *sql.Row) (*models.Claim, error) {
	var c models.Claim
	var releasedAt sql.NullTime
	var ticketTitle, ticketKey sql.NullString

	err := row.Scan(
		&c.ID, &c.TicketID, &c.WorkerID, &c.ClaimedAt, &c.ExpiresAt,
		&releasedAt, &c.Status, &ticketTitle, &ticketKey,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan claim: %w", err)
	}

	if releasedAt.Valid {
		c.ReleasedAt = &releasedAt.Time
	}
	c.TicketTitle = ticketTitle.String
	c.TicketKey = ticketKey.String
	return &c, nil
}

func (r *ClaimRepo) scanMany(rows *sql.Rows) ([]*models.Claim, error) {
	var claims []*models.Claim
	for rows.Next() {
		var c models.Claim
		var releasedAt sql.NullTime
		var ticketTitle, ticketKey sql.NullString

		err := rows.Scan(
			&c.ID, &c.TicketID, &c.WorkerID, &c.ClaimedAt, &c.ExpiresAt,
			&releasedAt, &c.Status, &ticketTitle, &ticketKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan claim: %w", err)
		}

		if releasedAt.Valid {
			c.ReleasedAt = &releasedAt.Time
		}
		c.TicketTitle = ticketTitle.String
		c.TicketKey = ticketKey.String
		claims = append(claims, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating claims: %w", err)
	}
	return claims, nil
}

func (r *ClaimRepo) scanManyWithMinutes(rows *sql.Rows) ([]*models.Claim, error) {
	var claims []*models.Claim
	for rows.Next() {
		var c models.Claim
		var releasedAt sql.NullTime
		var ticketTitle, ticketKey sql.NullString
		var minutesRemaining sql.NullInt64

		err := rows.Scan(
			&c.ID, &c.TicketID, &c.WorkerID, &c.ClaimedAt, &c.ExpiresAt,
			&releasedAt, &c.Status, &ticketTitle, &ticketKey, &minutesRemaining,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan claim: %w", err)
		}

		if releasedAt.Valid {
			c.ReleasedAt = &releasedAt.Time
		}
		c.TicketTitle = ticketTitle.String
		c.TicketKey = ticketKey.String
		if minutesRemaining.Valid {
			c.MinutesRemaining = int(minutesRemaining.Int64)
		}
		claims = append(claims, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating claims: %w", err)
	}
	return claims, nil
}
