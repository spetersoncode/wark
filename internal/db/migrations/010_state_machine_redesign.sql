-- +goose Up
-- +goose StatementBegin

-- State Machine Redesign (WARK-12)
-- 
-- Changes:
-- 1. Remove 'created' state - tickets start in 'blocked' or 'ready'
-- 2. Replace 'done' and 'cancelled' with unified 'closed' state
-- 3. Add 'resolution' column for closed tickets
-- 4. Rename 'needs_human' to 'human' for consistency

-- First, drop all views that depend on the tickets table
DROP VIEW IF EXISTS field_change_history;
DROP VIEW IF EXISTS active_claims;
DROP VIEW IF EXISTS pending_human_input;
DROP VIEW IF EXISTS workable_tickets;

-- Create new tickets table with updated schema
CREATE TABLE tickets_new (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id          INTEGER NOT NULL REFERENCES projects(id),
    number              INTEGER NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT,

    -- Status (state machine)
    status              TEXT NOT NULL DEFAULT 'ready'
                        CHECK (status IN (
                            'blocked',
                            'ready',
                            'in_progress',
                            'human',
                            'review',
                            'closed'
                        )),

    -- Resolution (only set when status = 'closed')
    resolution          TEXT
                        CHECK (resolution IS NULL OR resolution IN (
                            'completed',
                            'wont_do',
                            'duplicate',
                            'invalid',
                            'obsolete'
                        )),

    -- Human input flag (reason when human)
    human_flag_reason   TEXT,

    -- Classification
    priority            TEXT NOT NULL DEFAULT 'medium'
                        CHECK (priority IN (
                            'highest', 'high', 'medium', 'low', 'lowest'
                        )),
    complexity          TEXT NOT NULL DEFAULT 'medium'
                        CHECK (complexity IN (
                            'trivial', 'small', 'medium', 'large', 'xlarge'
                        )),

    -- Git integration
    branch_name         TEXT,

    -- Retry tracking
    retry_count         INTEGER NOT NULL DEFAULT 0,
    max_retries         INTEGER NOT NULL DEFAULT 3,

    -- Hierarchy (for decomposition)
    parent_ticket_id    INTEGER REFERENCES tickets_new(id),

    -- Timestamps
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at        DATETIME,

    -- Composite unique constraint
    UNIQUE(project_id, number)
);

-- Migrate data with state mapping:
-- created → ready (no deps check in migration, will be corrected by app logic)
-- ready → ready
-- in_progress → in_progress
-- blocked → blocked
-- needs_human → human
-- review → review
-- done → closed (resolution = 'completed')
-- cancelled → closed (resolution = 'wont_do')
INSERT INTO tickets_new (
    id, project_id, number, title, description, status, resolution,
    human_flag_reason, priority, complexity, branch_name,
    retry_count, max_retries, parent_ticket_id,
    created_at, updated_at, completed_at
)
SELECT
    id, project_id, number, title, description,
    CASE status
        WHEN 'created' THEN 'ready'
        WHEN 'needs_human' THEN 'human'
        WHEN 'done' THEN 'closed'
        WHEN 'cancelled' THEN 'closed'
        ELSE status
    END AS status,
    CASE status
        WHEN 'done' THEN 'completed'
        WHEN 'cancelled' THEN 'wont_do'
        ELSE NULL
    END AS resolution,
    human_flag_reason, priority, complexity, branch_name,
    retry_count, max_retries, parent_ticket_id,
    created_at, updated_at, completed_at
FROM tickets;

-- Drop old table and rename new one
DROP TABLE tickets;
ALTER TABLE tickets_new RENAME TO tickets;

-- Recreate indexes (from 007_create_indexes.sql)
CREATE INDEX idx_tickets_project_id ON tickets(project_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_parent ON tickets(parent_ticket_id);
CREATE INDEX idx_tickets_project_status ON tickets(project_id, status);

-- Recreate all views with updated status references

-- Workable tickets view: only ready tickets with all dependencies closed
CREATE VIEW workable_tickets AS
SELECT t.*,
       p.key AS project_key,
       p.key || '-' || t.number AS ticket_key
FROM tickets t
JOIN projects p ON t.project_id = p.id
WHERE t.status = 'ready'
  AND NOT EXISTS (
      SELECT 1 FROM ticket_dependencies td
      JOIN tickets dep ON td.depends_on_id = dep.id
      WHERE td.ticket_id = t.id
        AND dep.status != 'closed'
  )
ORDER BY
    CASE t.priority
        WHEN 'highest' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        WHEN 'lowest' THEN 5
    END,
    t.created_at;

-- Pending human input view
CREATE VIEW pending_human_input AS
SELECT
    im.*,
    t.title AS ticket_title,
    p.key || '-' || t.number AS ticket_key
FROM inbox_messages im
JOIN tickets t ON im.ticket_id = t.id
JOIN projects p ON t.project_id = p.id
WHERE im.responded_at IS NULL
ORDER BY im.created_at;

-- Active claims view
CREATE VIEW active_claims AS
SELECT
    c.*,
    t.title AS ticket_title,
    p.key || '-' || t.number AS ticket_key,
    CAST((julianday(c.expires_at) - julianday('now')) * 24 * 60 AS INTEGER) AS minutes_remaining
FROM claims c
JOIN tickets t ON c.ticket_id = t.id
JOIN projects p ON t.project_id = p.id
WHERE c.status = 'active'
  AND c.expires_at > CURRENT_TIMESTAMP;

-- Field change history view
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

-- Drop all views that depend on the tickets table
DROP VIEW IF EXISTS field_change_history;
DROP VIEW IF EXISTS active_claims;
DROP VIEW IF EXISTS pending_human_input;
DROP VIEW IF EXISTS workable_tickets;

-- Recreate old tickets table
CREATE TABLE tickets_old (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id          INTEGER NOT NULL REFERENCES projects(id),
    number              INTEGER NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT,
    status              TEXT NOT NULL DEFAULT 'created'
                        CHECK (status IN (
                            'created',
                            'ready',
                            'in_progress',
                            'blocked',
                            'needs_human',
                            'review',
                            'done',
                            'cancelled'
                        )),
    human_flag_reason   TEXT,
    priority            TEXT NOT NULL DEFAULT 'medium'
                        CHECK (priority IN (
                            'highest', 'high', 'medium', 'low', 'lowest'
                        )),
    complexity          TEXT NOT NULL DEFAULT 'medium'
                        CHECK (complexity IN (
                            'trivial', 'small', 'medium', 'large', 'xlarge'
                        )),
    branch_name         TEXT,
    retry_count         INTEGER NOT NULL DEFAULT 0,
    max_retries         INTEGER NOT NULL DEFAULT 3,
    parent_ticket_id    INTEGER REFERENCES tickets_old(id),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at        DATETIME,
    UNIQUE(project_id, number)
);

-- Reverse migration
INSERT INTO tickets_old (
    id, project_id, number, title, description, status,
    human_flag_reason, priority, complexity, branch_name,
    retry_count, max_retries, parent_ticket_id,
    created_at, updated_at, completed_at
)
SELECT
    id, project_id, number, title, description,
    CASE 
        WHEN status = 'human' THEN 'needs_human'
        WHEN status = 'closed' AND resolution = 'completed' THEN 'done'
        WHEN status = 'closed' THEN 'cancelled'
        ELSE status
    END AS status,
    human_flag_reason, priority, complexity, branch_name,
    retry_count, max_retries, parent_ticket_id,
    created_at, updated_at, completed_at
FROM tickets;

DROP TABLE tickets;
ALTER TABLE tickets_old RENAME TO tickets;

-- Recreate indexes
CREATE INDEX idx_tickets_project_id ON tickets(project_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_parent ON tickets(parent_ticket_id);
CREATE INDEX idx_tickets_project_status ON tickets(project_id, status);

-- Recreate all views with old status references

-- Workable tickets view
CREATE VIEW workable_tickets AS
SELECT t.*,
       p.key AS project_key,
       p.key || '-' || t.number AS ticket_key
FROM tickets t
JOIN projects p ON t.project_id = p.id
WHERE t.status = 'ready'
  AND NOT EXISTS (
      SELECT 1 FROM ticket_dependencies td
      JOIN tickets dep ON td.depends_on_id = dep.id
      WHERE td.ticket_id = t.id
        AND dep.status NOT IN ('done', 'cancelled')
  )
ORDER BY
    CASE t.priority
        WHEN 'highest' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        WHEN 'lowest' THEN 5
    END,
    t.created_at;

-- Pending human input view
CREATE VIEW pending_human_input AS
SELECT
    im.*,
    t.title AS ticket_title,
    p.key || '-' || t.number AS ticket_key
FROM inbox_messages im
JOIN tickets t ON im.ticket_id = t.id
JOIN projects p ON t.project_id = p.id
WHERE im.responded_at IS NULL
ORDER BY im.created_at;

-- Active claims view
CREATE VIEW active_claims AS
SELECT
    c.*,
    t.title AS ticket_title,
    p.key || '-' || t.number AS ticket_key,
    CAST((julianday(c.expires_at) - julianday('now')) * 24 * 60 AS INTEGER) AS minutes_remaining
FROM claims c
JOIN tickets t ON c.ticket_id = t.id
JOIN projects p ON t.project_id = p.id
WHERE c.status = 'active'
  AND c.expires_at > CURRENT_TIMESTAMP;

-- Field change history view
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
