package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/models"
)

// ProjectRepo provides database operations for projects.
type ProjectRepo struct {
	db *sql.DB
}

// NewProjectRepo creates a new ProjectRepo.
func NewProjectRepo(db *sql.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

// Create creates a new project.
func (r *ProjectRepo) Create(p *models.Project) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	query := `
		INSERT INTO projects (key, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`
	now := time.Now()
	nowStr := FormatTime(now)
	result, err := r.db.Exec(query, p.Key, p.Name, p.Description, nowStr, nowStr)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get project id: %w", err)
	}

	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// GetByID retrieves a project by ID.
func (r *ProjectRepo) GetByID(id int64) (*models.Project, error) {
	query := `SELECT id, key, name, description, created_at, updated_at FROM projects WHERE id = ?`
	return r.scanOne(r.db.QueryRow(query, id))
}

// GetByKey retrieves a project by its key.
func (r *ProjectRepo) GetByKey(key string) (*models.Project, error) {
	query := `SELECT id, key, name, description, created_at, updated_at FROM projects WHERE key = ?`
	return r.scanOne(r.db.QueryRow(query, key))
}

// List retrieves all projects.
func (r *ProjectRepo) List() ([]*models.Project, error) {
	query := `SELECT id, key, name, description, created_at, updated_at FROM projects ORDER BY key`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// Update updates a project.
func (r *ProjectRepo) Update(p *models.Project) error {
	if p.ID <= 0 {
		return fmt.Errorf("project id is required")
	}
	if p.Name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	query := `UPDATE projects SET name = ?, description = ? WHERE id = ?`
	result, err := r.db.Exec(query, p.Name, p.Description, p.ID)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

// Delete deletes a project by ID.
func (r *ProjectRepo) Delete(id int64) error {
	query := `DELETE FROM projects WHERE id = ?`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

// GetStats retrieves statistics for a project.
func (r *ProjectRepo) GetStats(projectID int64) (*models.ProjectStats, error) {
	query := `
		SELECT
			COUNT(*) AS total,
			SUM(CASE WHEN status = 'blocked' THEN 1 ELSE 0 END) AS blocked,
			SUM(CASE WHEN status = 'ready' THEN 1 ELSE 0 END) AS ready,
			SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) AS in_progress,
			SUM(CASE WHEN status = 'human' THEN 1 ELSE 0 END) AS human,
			SUM(CASE WHEN status = 'review' THEN 1 ELSE 0 END) AS review,
			SUM(CASE WHEN status = 'closed' AND resolution = 'completed' THEN 1 ELSE 0 END) AS closed_completed,
			SUM(CASE WHEN status = 'closed' AND resolution != 'completed' THEN 1 ELSE 0 END) AS closed_other
		FROM tickets
		WHERE project_id = ?
	`
	var stats models.ProjectStats
	err := r.db.QueryRow(query, projectID).Scan(
		&stats.TotalTickets,
		&stats.BlockedCount,
		&stats.ReadyCount,
		&stats.InProgressCount,
		&stats.HumanCount,
		&stats.ReviewCount,
		&stats.ClosedCompletedCount,
		&stats.ClosedOtherCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}
	return &stats, nil
}

// Exists checks if a project with the given key exists.
func (r *ProjectRepo) Exists(key string) (bool, error) {
	query := `SELECT 1 FROM projects WHERE key = ? LIMIT 1`
	var exists int
	err := r.db.QueryRow(query, key).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check project existence: %w", err)
	}
	return true, nil
}

func (r *ProjectRepo) scanOne(row *sql.Row) (*models.Project, error) {
	var p models.Project
	var desc sql.NullString
	err := row.Scan(&p.ID, &p.Key, &p.Name, &desc, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}
	p.Description = desc.String
	return &p, nil
}

func (r *ProjectRepo) scanMany(rows *sql.Rows) ([]*models.Project, error) {
	var projects []*models.Project
	for rows.Next() {
		var p models.Project
		var desc sql.NullString
		err := rows.Scan(&p.ID, &p.Key, &p.Name, &desc, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		p.Description = desc.String
		projects = append(projects, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating projects: %w", err)
	}
	return projects, nil
}
