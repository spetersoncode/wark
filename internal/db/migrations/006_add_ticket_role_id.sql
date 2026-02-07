-- +goose Up
-- +goose StatementBegin

-- =============================================================================
-- Add role_id to tickets
-- =============================================================================
-- Allows tickets to specify a role for execution instead of raw brain config.
-- When a ticket has a role_id, the role's instructions are used as the
-- execution context. The brain field is kept for backward compatibility.
-- =============================================================================

-- Add role_id column (nullable foreign key to roles table)
ALTER TABLE tickets ADD COLUMN role_id INTEGER REFERENCES roles(id);

-- Index for fast role lookups
CREATE INDEX idx_tickets_role_id ON tickets(role_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove index and column
DROP INDEX IF EXISTS idx_tickets_role_id;
ALTER TABLE tickets DROP COLUMN role_id;

-- +goose StatementEnd
