-- +goose Up
-- +goose StatementBegin

-- Add missing action types to activity_log CHECK constraint.
-- 
-- SQLite doesn't support ALTER TABLE to modify CHECK constraints,
-- so we need to recreate the table.
--
-- New actions being added:
-- - 'escalated': when an agent sends a message to the inbox
-- - 'promoted': when a ticket is promoted from draft to ready
-- - 'closed': when a ticket is closed
-- - 'task_completed': when a task is marked complete

-- Drop dependent views first
DROP VIEW IF EXISTS field_change_history;

-- Create new activity_log table with updated CHECK constraint
CREATE TABLE activity_log_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,

    -- What happened
    action          TEXT NOT NULL
                    CHECK (action IN (
                        -- Lifecycle actions
                        'created',
                        'vetted',
                        'claimed',
                        'released',
                        'expired',
                        'completed',
                        'accepted',
                        'rejected',
                        'cancelled',
                        'reopened',
                        'closed',
                        'promoted',

                        -- Dependency actions
                        'dependency_added',
                        'dependency_removed',
                        'blocked',
                        'unblocked',

                        -- Decomposition
                        'decomposed',
                        'child_created',

                        -- Task actions
                        'task_completed',

                        -- Human interaction
                        'escalated',
                        'flagged_human',
                        'human_responded',

                        -- Field changes
                        'field_changed',

                        -- Comments/notes
                        'comment'
                    )),

    -- Who did it
    actor_type      TEXT NOT NULL
                    CHECK (actor_type IN ('human', 'agent', 'system')),
    actor_id        TEXT,

    -- Details (JSON for flexibility)
    details         TEXT,

    -- Human-readable summary
    summary         TEXT,

    -- Timestamps
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Copy existing data
INSERT INTO activity_log_new (id, ticket_id, action, actor_type, actor_id, details, summary, created_at)
SELECT id, ticket_id, action, actor_type, actor_id, details, summary, created_at
FROM activity_log;

-- Drop old table and rename new one
DROP TABLE activity_log;
ALTER TABLE activity_log_new RENAME TO activity_log;

-- Recreate indexes from 007_create_indexes.sql
CREATE INDEX idx_activity_log_ticket_id ON activity_log(ticket_id);
CREATE INDEX idx_activity_log_action ON activity_log(action);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at);

-- Recreate the field_change_history view
CREATE VIEW field_change_history AS
SELECT
    id,
    ticket_id,
    json_extract(details, '$.field') AS field_name,
    json_extract(details, '$.old') AS old_value,
    json_extract(details, '$.new') AS new_value,
    actor_type || COALESCE(':' || actor_id, '') AS changed_by,
    created_at
FROM activity_log
WHERE action = 'field_changed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Revert to original CHECK constraint (without the new actions)
-- Note: This will fail if any of the new actions are present in the data

DROP VIEW IF EXISTS field_change_history;

CREATE TABLE activity_log_old (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    action          TEXT NOT NULL
                    CHECK (action IN (
                        'created', 'vetted', 'claimed', 'released', 'expired',
                        'completed', 'accepted', 'rejected', 'cancelled', 'reopened',
                        'dependency_added', 'dependency_removed', 'blocked', 'unblocked',
                        'decomposed', 'child_created',
                        'flagged_human', 'human_responded',
                        'field_changed', 'comment'
                    )),
    actor_type      TEXT NOT NULL CHECK (actor_type IN ('human', 'agent', 'system')),
    actor_id        TEXT,
    details         TEXT,
    summary         TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO activity_log_old (id, ticket_id, action, actor_type, actor_id, details, summary, created_at)
SELECT id, ticket_id, action, actor_type, actor_id, details, summary, created_at
FROM activity_log
WHERE action NOT IN ('escalated', 'promoted', 'closed', 'task_completed');

DROP TABLE activity_log;
ALTER TABLE activity_log_old RENAME TO activity_log;

CREATE INDEX idx_activity_log_ticket_id ON activity_log(ticket_id);
CREATE INDEX idx_activity_log_action ON activity_log(action);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at);

CREATE VIEW field_change_history AS
SELECT
    id,
    ticket_id,
    json_extract(details, '$.field') AS field_name,
    json_extract(details, '$.old') AS old_value,
    json_extract(details, '$.new') AS new_value,
    actor_type || COALESCE(':' || actor_id, '') AS changed_by,
    created_at
FROM activity_log
WHERE action = 'field_changed';

-- +goose StatementEnd
