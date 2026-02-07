-- +goose Up
-- +goose StatementBegin

-- =============================================================================
-- Add brain field to tickets
-- =============================================================================
-- Adds a brain field to specify what executes the work on a ticket.
-- Brain is a freeform text field providing guidance for the execution harness.
-- Examples: "sonnet", "opus with extended thinking", "claude-code --skip-perms"
-- =============================================================================

-- Add brain column (nullable, as not all tickets need a brain specified)
ALTER TABLE tickets ADD COLUMN brain TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove brain column
ALTER TABLE tickets DROP COLUMN brain;

-- +goose StatementEnd
