package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/spetersoncode/wark/internal/models"
)

// RoleRepo provides database operations for roles.
type RoleRepo struct {
	db *sql.DB
}

// NewRoleRepo creates a new RoleRepo.
func NewRoleRepo(db *sql.DB) *RoleRepo {
	return &RoleRepo{db: db}
}

// Create creates a new role.
func (r *RoleRepo) Create(role *models.Role) error {
	if err := role.Validate(); err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	query := `
		INSERT INTO roles (name, description, instructions, is_builtin, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	nowStr := FormatTime(now)
	result, err := r.db.Exec(query, role.Name, role.Description, role.Instructions, role.IsBuiltin, nowStr, nowStr)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get role id: %w", err)
	}

	role.ID = id
	role.CreatedAt = now
	role.UpdatedAt = now
	return nil
}

// GetByID retrieves a role by ID.
func (r *RoleRepo) GetByID(id int64) (*models.Role, error) {
	query := `SELECT id, name, description, instructions, is_builtin, created_at, updated_at FROM roles WHERE id = ?`
	return r.scanOne(r.db.QueryRow(query, id))
}

// GetByName retrieves a role by its name.
func (r *RoleRepo) GetByName(name string) (*models.Role, error) {
	query := `SELECT id, name, description, instructions, is_builtin, created_at, updated_at FROM roles WHERE name = ?`
	return r.scanOne(r.db.QueryRow(query, name))
}

// List retrieves all roles, optionally filtered by builtin status.
// If builtinFilter is nil, returns all roles.
// If builtinFilter is true, returns only built-in roles.
// If builtinFilter is false, returns only user-defined roles.
func (r *RoleRepo) List(builtinFilter *bool) ([]*models.Role, error) {
	var query string
	var rows *sql.Rows
	var err error

	if builtinFilter == nil {
		query = `SELECT id, name, description, instructions, is_builtin, created_at, updated_at FROM roles ORDER BY name`
		rows, err = r.db.Query(query)
	} else {
		query = `SELECT id, name, description, instructions, is_builtin, created_at, updated_at FROM roles WHERE is_builtin = ? ORDER BY name`
		rows, err = r.db.Query(query, *builtinFilter)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// Update updates a role.
// Built-in roles cannot be updated (name, is_builtin cannot change).
// Only description and instructions can be updated.
func (r *RoleRepo) Update(role *models.Role) error {
	if role.ID <= 0 {
		return fmt.Errorf("role id is required")
	}
	if err := role.Validate(); err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	// Check if role exists and if it's built-in
	existing, err := r.GetByID(role.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("role not found")
	}
	if existing.IsBuiltin {
		return fmt.Errorf("cannot update built-in role")
	}

	query := `UPDATE roles SET description = ?, instructions = ? WHERE id = ?`
	result, err := r.db.Exec(query, role.Description, role.Instructions, role.ID)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not found")
	}

	return nil
}

// Delete deletes a role by ID.
// Built-in roles cannot be deleted.
func (r *RoleRepo) Delete(id int64) error {
	// Check if role exists and if it's built-in
	existing, err := r.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("role not found")
	}
	if existing.IsBuiltin {
		return fmt.Errorf("cannot delete built-in role")
	}

	query := `DELETE FROM roles WHERE id = ?`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not found")
	}

	return nil
}

// Exists checks if a role with the given name exists.
func (r *RoleRepo) Exists(name string) (bool, error) {
	query := `SELECT 1 FROM roles WHERE name = ? LIMIT 1`
	var exists int
	err := r.db.QueryRow(query, name).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check role existence: %w", err)
	}
	return true, nil
}

// Count returns the total number of roles, optionally filtered by builtin status.
func (r *RoleRepo) Count(builtinFilter *bool) (int, error) {
	var query string
	var count int
	var err error

	if builtinFilter == nil {
		query = `SELECT COUNT(*) FROM roles`
		err = r.db.QueryRow(query).Scan(&count)
	} else {
		query = `SELECT COUNT(*) FROM roles WHERE is_builtin = ?`
		err = r.db.QueryRow(query, *builtinFilter).Scan(&count)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to count roles: %w", err)
	}
	return count, nil
}

func (r *RoleRepo) scanOne(row *sql.Row) (*models.Role, error) {
	var role models.Role
	err := row.Scan(&role.ID, &role.Name, &role.Description, &role.Instructions, &role.IsBuiltin, &role.CreatedAt, &role.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan role: %w", err)
	}
	return &role, nil
}

func (r *RoleRepo) scanMany(rows *sql.Rows) ([]*models.Role, error) {
	var roles []*models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.Instructions, &role.IsBuiltin, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, &role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}
	return roles, nil
}
