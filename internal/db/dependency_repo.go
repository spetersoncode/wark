package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// DependencyRepo provides database operations for ticket dependencies.
type DependencyRepo struct {
	db *sql.DB
}

// NewDependencyRepo creates a new DependencyRepo.
func NewDependencyRepo(db *sql.DB) *DependencyRepo {
	return &DependencyRepo{db: db}
}

// Add adds a dependency between two tickets.
func (r *DependencyRepo) Add(ticketID, dependsOnID int64) error {
	if ticketID == dependsOnID {
		return fmt.Errorf("ticket cannot depend on itself")
	}

	// Check for circular dependency
	if circular, err := r.wouldCreateCycle(ticketID, dependsOnID); err != nil {
		return fmt.Errorf("failed to check for circular dependency: %w", err)
	} else if circular {
		return fmt.Errorf("adding this dependency would create a circular dependency")
	}

	query := `INSERT INTO ticket_dependencies (ticket_id, depends_on_id, created_at) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, ticketID, dependsOnID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add dependency: %w", err)
	}
	return nil
}

// Remove removes a dependency between two tickets.
func (r *DependencyRepo) Remove(ticketID, dependsOnID int64) error {
	query := `DELETE FROM ticket_dependencies WHERE ticket_id = ? AND depends_on_id = ?`
	result, err := r.db.Exec(query, ticketID, dependsOnID)
	if err != nil {
		return fmt.Errorf("failed to remove dependency: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("dependency not found")
	}
	return nil
}

// GetDependencies retrieves all tickets that the given ticket depends on.
func (r *DependencyRepo) GetDependencies(ticketID int64) ([]*models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.branch_name,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		JOIN ticket_dependencies td ON t.id = td.depends_on_id
		WHERE td.ticket_id = ?
		ORDER BY t.created_at
	`
	rows, err := r.db.Query(query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}
	defer rows.Close()

	return r.scanTickets(rows)
}

// GetDependents retrieves all tickets that depend on the given ticket.
func (r *DependencyRepo) GetDependents(ticketID int64) ([]*models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.branch_name,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		JOIN ticket_dependencies td ON t.id = td.ticket_id
		WHERE td.depends_on_id = ?
		ORDER BY t.created_at
	`
	rows, err := r.db.Query(query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependents: %w", err)
	}
	defer rows.Close()

	return r.scanTickets(rows)
}

// GetUnresolvedDependencies retrieves all unresolved dependencies for a ticket.
// A dependency is only resolved if its ticket is closed with 'completed' resolution.
func (r *DependencyRepo) GetUnresolvedDependencies(ticketID int64) ([]*models.Ticket, error) {
	query := `
		SELECT t.id, t.project_id, t.number, t.title, t.description, t.status,
			t.resolution, t.human_flag_reason, t.priority, t.complexity, t.branch_name,
			t.retry_count, t.max_retries, t.parent_ticket_id,
			t.created_at, t.updated_at, t.completed_at,
			p.key AS project_key
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		JOIN ticket_dependencies td ON t.id = td.depends_on_id
		WHERE td.ticket_id = ? AND NOT (t.status = 'closed' AND t.resolution = 'completed')
		ORDER BY t.created_at
	`
	rows, err := r.db.Query(query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unresolved dependencies: %w", err)
	}
	defer rows.Close()

	return r.scanTickets(rows)
}

// HasUnresolvedDependencies checks if a ticket has any unresolved dependencies.
// A dependency is only resolved if its ticket is closed with 'completed' resolution.
func (r *DependencyRepo) HasUnresolvedDependencies(ticketID int64) (bool, error) {
	query := `
		SELECT 1 FROM ticket_dependencies td
		JOIN tickets t ON td.depends_on_id = t.id
		WHERE td.ticket_id = ? AND NOT (t.status = 'closed' AND t.resolution = 'completed')
		LIMIT 1
	`
	var exists int
	err := r.db.QueryRow(query, ticketID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check dependencies: %w", err)
	}
	return true, nil
}

// IsBlocked checks if a ticket is blocked by unresolved dependencies.
func (r *DependencyRepo) IsBlocked(ticketID int64) (bool, error) {
	return r.HasUnresolvedDependencies(ticketID)
}

// Exists checks if a dependency exists between two tickets.
func (r *DependencyRepo) Exists(ticketID, dependsOnID int64) (bool, error) {
	query := `SELECT 1 FROM ticket_dependencies WHERE ticket_id = ? AND depends_on_id = ? LIMIT 1`
	var exists int
	err := r.db.QueryRow(query, ticketID, dependsOnID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check dependency: %w", err)
	}
	return true, nil
}

// wouldCreateCycle checks if adding a dependency from ticketID to dependsOnID would create a cycle.
// This uses a recursive CTE to traverse the dependency graph.
func (r *DependencyRepo) wouldCreateCycle(ticketID, dependsOnID int64) (bool, error) {
	// Check if dependsOnID already (transitively) depends on ticketID
	query := `
		WITH RECURSIVE dep_chain(id) AS (
			SELECT depends_on_id FROM ticket_dependencies WHERE ticket_id = ?
			UNION
			SELECT td.depends_on_id
			FROM ticket_dependencies td
			JOIN dep_chain dc ON td.ticket_id = dc.id
		)
		SELECT 1 FROM dep_chain WHERE id = ? LIMIT 1
	`
	var exists int
	err := r.db.QueryRow(query, dependsOnID, ticketID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CountDependencies counts the number of dependencies for a ticket.
func (r *DependencyRepo) CountDependencies(ticketID int64) (int, error) {
	query := `SELECT COUNT(*) FROM ticket_dependencies WHERE ticket_id = ?`
	var count int
	err := r.db.QueryRow(query, ticketID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count dependencies: %w", err)
	}
	return count, nil
}

// CountDependents counts the number of tickets that depend on the given ticket.
func (r *DependencyRepo) CountDependents(ticketID int64) (int, error) {
	query := `SELECT COUNT(*) FROM ticket_dependencies WHERE depends_on_id = ?`
	var count int
	err := r.db.QueryRow(query, ticketID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count dependents: %w", err)
	}
	return count, nil
}

func (r *DependencyRepo) scanTickets(rows *sql.Rows) ([]*models.Ticket, error) {
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
