-- +goose Up
-- +goose StatementBegin

-- Migration 014: Fix timestamp format for SQLite compatibility
--
-- Problem: Timestamps were stored in Go's verbose format:
--   "2026-02-02 10:03:03.54625057 -0800 PST m=+0.002057631"
-- SQLite's julianday() can't parse this, breaking cycle time calculations.
--
-- Solution: Convert to RFC 3339 format:
--   "2026-02-03T11:41:00Z"
--
-- The conversion extracts the first 19 characters (YYYY-MM-DD HH:MM:SS),
-- replaces the space with 'T', and appends 'Z' for UTC.
-- This is a simplification that loses timezone info but works for SQLite.

-- Fix tickets table
UPDATE tickets SET created_at = replace(substr(created_at, 1, 19), ' ', 'T') || 'Z'
WHERE created_at LIKE '____-__-__ __:__:__%' AND created_at NOT LIKE '____-__-__T__:__:__%';

UPDATE tickets SET updated_at = replace(substr(updated_at, 1, 19), ' ', 'T') || 'Z'
WHERE updated_at LIKE '____-__-__ __:__:__%' AND updated_at NOT LIKE '____-__-__T__:__:__%';

UPDATE tickets SET completed_at = replace(substr(completed_at, 1, 19), ' ', 'T') || 'Z'
WHERE completed_at IS NOT NULL 
AND completed_at LIKE '____-__-__ __:__:__%' AND completed_at NOT LIKE '____-__-__T__:__:__%';

-- Fix claims table
UPDATE claims SET claimed_at = replace(substr(claimed_at, 1, 19), ' ', 'T') || 'Z'
WHERE claimed_at LIKE '____-__-__ __:__:__%' AND claimed_at NOT LIKE '____-__-__T__:__:__%';

UPDATE claims SET expires_at = replace(substr(expires_at, 1, 19), ' ', 'T') || 'Z'
WHERE expires_at LIKE '____-__-__ __:__:__%' AND expires_at NOT LIKE '____-__-__T__:__:__%';

UPDATE claims SET released_at = replace(substr(released_at, 1, 19), ' ', 'T') || 'Z'
WHERE released_at IS NOT NULL 
AND released_at LIKE '____-__-__ __:__:__%' AND released_at NOT LIKE '____-__-__T__:__:__%';

-- Fix inbox_messages table
UPDATE inbox_messages SET created_at = replace(substr(created_at, 1, 19), ' ', 'T') || 'Z'
WHERE created_at LIKE '____-__-__ __:__:__%' AND created_at NOT LIKE '____-__-__T__:__:__%';

UPDATE inbox_messages SET responded_at = replace(substr(responded_at, 1, 19), ' ', 'T') || 'Z'
WHERE responded_at IS NOT NULL 
AND responded_at LIKE '____-__-__ __:__:__%' AND responded_at NOT LIKE '____-__-__T__:__:__%';

-- Fix activity_log table
UPDATE activity_log SET created_at = replace(substr(created_at, 1, 19), ' ', 'T') || 'Z'
WHERE created_at LIKE '____-__-__ __:__:__%' AND created_at NOT LIKE '____-__-__T__:__:__%';

-- Fix projects table
UPDATE projects SET created_at = replace(substr(created_at, 1, 19), ' ', 'T') || 'Z'
WHERE created_at LIKE '____-__-__ __:__:__%' AND created_at NOT LIKE '____-__-__T__:__:__%';

UPDATE projects SET updated_at = replace(substr(updated_at, 1, 19), ' ', 'T') || 'Z'
WHERE updated_at LIKE '____-__-__ __:__:__%' AND updated_at NOT LIKE '____-__-__T__:__:__%';

-- Fix ticket_tasks table
UPDATE ticket_tasks SET created_at = replace(substr(created_at, 1, 19), ' ', 'T') || 'Z'
WHERE created_at LIKE '____-__-__ __:__:__%' AND created_at NOT LIKE '____-__-__T__:__:__%';

UPDATE ticket_tasks SET updated_at = replace(substr(updated_at, 1, 19), ' ', 'T') || 'Z'
WHERE updated_at LIKE '____-__-__ __:__:__%' AND updated_at NOT LIKE '____-__-__T__:__:__%';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- No-op: We can't reliably restore the original Go format,
-- and RFC 3339 is the desired format going forward.
-- This migration is essentially one-way.

-- +goose StatementEnd
