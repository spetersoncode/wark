-- +goose Up
-- +goose StatementBegin

-- =============================================================================
-- Rename in_progress status to working
-- =============================================================================
-- Updates the status enum value from 'in_progress' to 'working' for simpler,
-- single-word consistency across all statuses.
--
-- NOTE: For fresh databases, the 001_initial_schema.sql already uses 'working'.
-- This migration is only needed to update existing databases that had 'in_progress'.
-- The CHECK constraint in 001_initial_schema.sql has been updated to use 'working',
-- so this migration only needs to update data for existing databases.
-- =============================================================================

-- Update existing tickets with in_progress status (safe even if none exist)
UPDATE tickets SET status = 'working' WHERE status = 'in_progress';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Revert tickets with working status back to in_progress (for rollback to old code)
UPDATE tickets SET status = 'in_progress' WHERE status = 'working';

-- +goose StatementEnd
