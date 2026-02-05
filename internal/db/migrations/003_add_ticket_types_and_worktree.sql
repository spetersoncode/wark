-- +goose Up
-- +goose StatementBegin

-- =============================================================================
-- Add ticket types (epic/task) and rename branch_name to worktree
-- =============================================================================
-- Adds a ticket_type column to distinguish between epics and tasks.
-- Epics are parent tickets that own a worktree that child tasks inherit.
-- Also renames branch_name to worktree for clearer semantics.
-- =============================================================================

-- Add ticket_type column with default 'task'
ALTER TABLE tickets ADD COLUMN ticket_type TEXT NOT NULL DEFAULT 'task'
    CHECK (ticket_type IN ('task', 'epic'));

-- Rename branch_name to worktree
ALTER TABLE tickets RENAME COLUMN branch_name TO worktree;

-- Index for efficient queries
CREATE INDEX idx_tickets_type ON tickets(ticket_type);
CREATE INDEX idx_tickets_parent_type ON tickets(parent_ticket_id, ticket_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes
DROP INDEX IF EXISTS idx_tickets_parent_type;
DROP INDEX IF EXISTS idx_tickets_type;

-- Rename worktree back to branch_name
ALTER TABLE tickets RENAME COLUMN worktree TO branch_name;

-- Remove ticket_type column
ALTER TABLE tickets DROP COLUMN ticket_type;

-- +goose StatementEnd
