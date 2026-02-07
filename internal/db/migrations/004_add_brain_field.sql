-- +goose Up
-- +goose StatementBegin

-- =============================================================================
-- Add brain field to tickets
-- =============================================================================
-- Adds a brain field to specify what executes the work on a ticket.
-- A brain is a JSON structure containing either a model or a tool.
-- Format: {"type": "model"|"tool", "value": "sonnet"|"opus"|"qwen"|"claude-code"}
-- =============================================================================

-- Add brain column (nullable, as not all tickets need a brain specified)
ALTER TABLE tickets ADD COLUMN brain TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove brain column
ALTER TABLE tickets DROP COLUMN brain;

-- +goose StatementEnd
