-- +goose Up
-- +goose StatementBegin

-- =============================================================================
-- WARK Initial Schema
-- =============================================================================
-- A task management system for coordinating AI agents and humans.
-- All tables, indexes, views, and triggers in dependency order.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- PROJECTS
-- -----------------------------------------------------------------------------
-- Top-level organizational unit. Each project has a unique key used in ticket
-- identifiers (e.g., "WARK-42").

CREATE TABLE projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    key             TEXT NOT NULL UNIQUE,      -- Short identifier (e.g., "WARK")
    name            TEXT NOT NULL,             -- Human-readable name
    description     TEXT,                      -- Optional longer description
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- MILESTONES
-- -----------------------------------------------------------------------------
-- Time-boxed goals within a project. Tickets can be assigned to milestones
-- to track progress toward larger objectives.

CREATE TABLE milestones (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id      INTEGER NOT NULL REFERENCES projects(id),
    key             TEXT NOT NULL,             -- Short identifier (e.g., "v1.0")
    name            TEXT NOT NULL,             -- Human-readable name
    goal            TEXT,                      -- Description of what this milestone achieves
    target_date     DATETIME,                  -- Optional deadline
    status          TEXT NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'achieved', 'abandoned')),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, key)
);

-- -----------------------------------------------------------------------------
-- TICKETS
-- -----------------------------------------------------------------------------
-- The core work unit. Represents a task that can be worked on by agents or humans.
-- 
-- State machine:
--   blocked     → Waiting on dependencies or external factors
--   ready       → Available for an agent to claim
--   working → Currently being worked on
--   human       → Needs human input/decision
--   review      → Work complete, awaiting human review
--   closed      → Terminal state (with resolution)
--
-- Resolutions (only valid when status = 'closed'):
--   completed   → Work finished successfully
--   wont_do     → Decided not to do this
--   duplicate   → Same as another ticket
--   invalid     → Not a valid task
--   obsolete    → No longer relevant

CREATE TABLE tickets (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id          INTEGER NOT NULL REFERENCES projects(id),
    number              INTEGER NOT NULL,          -- Auto-generated per-project number
    title               TEXT NOT NULL,
    description         TEXT,

    -- Status (state machine)
    status              TEXT NOT NULL DEFAULT 'ready'
                        CHECK (status IN (
                            'blocked',
                            'ready',
                            'working',
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

    -- Human input flag (reason when status = 'human')
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

    -- Retry tracking (for failed agent attempts)
    retry_count         INTEGER NOT NULL DEFAULT 0,
    max_retries         INTEGER NOT NULL DEFAULT 3,

    -- Hierarchy (for decomposition into sub-tasks)
    parent_ticket_id    INTEGER REFERENCES tickets(id),

    -- Milestone assignment
    milestone_id        INTEGER REFERENCES milestones(id),

    -- Timestamps
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at        DATETIME,

    -- Composite unique constraint
    UNIQUE(project_id, number)
);

-- -----------------------------------------------------------------------------
-- TICKET DEPENDENCIES
-- -----------------------------------------------------------------------------
-- Defines blocking relationships between tickets. A ticket cannot transition
-- to 'ready' until all its dependencies are 'closed'.

CREATE TABLE ticket_dependencies (
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    depends_on_id   INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (ticket_id, depends_on_id),

    -- Prevent self-dependency
    CHECK (ticket_id != depends_on_id)
);

-- -----------------------------------------------------------------------------
-- CLAIMS
-- -----------------------------------------------------------------------------
-- Tracks which agent is currently working on a ticket. Claims have expiration
-- to handle abandoned work and prevent deadlocks.

CREATE TABLE claims (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id),
    worker_id       TEXT NOT NULL,             -- Agent identifier
    claimed_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME NOT NULL,         -- Claim auto-expires after this
    released_at     DATETIME,                  -- When claim was explicitly released

    status          TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN (
                        'active',              -- Currently held
                        'completed',           -- Work finished successfully
                        'expired',             -- Timed out
                        'released'             -- Explicitly released
                    ))
);

-- -----------------------------------------------------------------------------
-- INBOX MESSAGES
-- -----------------------------------------------------------------------------
-- Communication channel between agents and humans. When an agent needs input,
-- it creates an inbox message and waits for a human response.

CREATE TABLE inbox_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id),

    -- Message classification
    message_type    TEXT NOT NULL DEFAULT 'question'
                    CHECK (message_type IN (
                        'question',            -- Agent needs information
                        'decision',            -- Agent needs a decision made
                        'review',              -- Agent wants human to review work
                        'escalation',          -- Agent is escalating an issue
                        'info'                 -- Agent is providing information
                    )),

    -- Content
    content         TEXT NOT NULL,
    from_agent      TEXT,                      -- Which agent sent this

    -- Response
    response        TEXT,
    responded_at    DATETIME,

    -- Timestamps
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- ACTIVITY LOG
-- -----------------------------------------------------------------------------
-- Comprehensive audit trail of all ticket events. Used for debugging,
-- analytics, and reconstructing ticket history.

CREATE TABLE activity_log (
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

-- -----------------------------------------------------------------------------
-- TICKET TASKS
-- -----------------------------------------------------------------------------
-- Checklist items within a ticket. Allows breaking down work into smaller
-- trackable steps.

CREATE TABLE ticket_tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id   INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,              -- Order within ticket
    description TEXT NOT NULL,
    complete    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(ticket_id, position)
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Projects
CREATE UNIQUE INDEX idx_projects_key ON projects(key);

-- Tickets
CREATE INDEX idx_tickets_project_id ON tickets(project_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_parent ON tickets(parent_ticket_id);
CREATE INDEX idx_tickets_project_status ON tickets(project_id, status);

-- Dependencies
CREATE INDEX idx_dependencies_depends_on ON ticket_dependencies(depends_on_id);

-- Claims
CREATE INDEX idx_claims_ticket_active ON claims(ticket_id, status);
CREATE INDEX idx_claims_expires ON claims(expires_at);
-- Prevents race conditions: only one active claim per ticket at database level
CREATE UNIQUE INDEX idx_claims_one_active ON claims(ticket_id) WHERE status = 'active';

-- Inbox messages
CREATE INDEX idx_inbox_pending ON inbox_messages(responded_at);
CREATE INDEX idx_inbox_ticket ON inbox_messages(ticket_id);

-- Activity log
CREATE INDEX idx_activity_log_ticket_id ON activity_log(ticket_id);
CREATE INDEX idx_activity_log_action ON activity_log(action);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at);

-- Ticket tasks
CREATE INDEX idx_ticket_tasks_ticket ON ticket_tasks(ticket_id, position);

-- =============================================================================
-- VIEWS
-- =============================================================================

-- Workable tickets: ready tickets with all dependencies satisfied
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

-- Pending human input: inbox messages awaiting response
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

-- Active claims: currently held ticket claims with time remaining
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

-- Field change history: extracted from activity log for easy querying
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

-- =============================================================================
-- TRIGGERS
-- =============================================================================

-- Auto-update updated_at timestamp for tickets
CREATE TRIGGER update_ticket_timestamp
AFTER UPDATE ON tickets
FOR EACH ROW
BEGIN
    UPDATE tickets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Auto-update updated_at timestamp for projects
CREATE TRIGGER update_project_timestamp
AFTER UPDATE ON projects
FOR EACH ROW
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Auto-generate ticket number (if not provided)
CREATE TRIGGER generate_ticket_number
AFTER INSERT ON tickets
FOR EACH ROW
WHEN NEW.number IS NULL OR NEW.number = 0
BEGIN
    UPDATE tickets
    SET number = (
        SELECT COALESCE(MAX(number), 0) + 1
        FROM tickets
        WHERE project_id = NEW.project_id
    )
    WHERE id = NEW.id;
END;

-- Record ticket creation in activity log
CREATE TRIGGER record_ticket_creation
AFTER INSERT ON tickets
FOR EACH ROW
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, summary)
    VALUES (NEW.id, 'created', 'system', 'Ticket created');
END;

-- Record status changes in activity log
CREATE TRIGGER record_status_change
AFTER UPDATE OF status ON tickets
FOR EACH ROW
WHEN OLD.status != NEW.status
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id,
        'field_changed',
        'system',
        json_object('field', 'status', 'old', OLD.status, 'new', NEW.status),
        'Status: ' || OLD.status || ' -> ' || NEW.status
    );
END;

-- Record priority changes in activity log
CREATE TRIGGER record_priority_change
AFTER UPDATE OF priority ON tickets
FOR EACH ROW
WHEN OLD.priority != NEW.priority
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id,
        'field_changed',
        'system',
        json_object('field', 'priority', 'old', OLD.priority, 'new', NEW.priority),
        'Priority: ' || OLD.priority || ' -> ' || NEW.priority
    );
END;

-- Record complexity changes in activity log
CREATE TRIGGER record_complexity_change
AFTER UPDATE OF complexity ON tickets
FOR EACH ROW
WHEN OLD.complexity != NEW.complexity
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id,
        'field_changed',
        'system',
        json_object('field', 'complexity', 'old', OLD.complexity, 'new', NEW.complexity),
        'Complexity: ' || OLD.complexity || ' -> ' || NEW.complexity
    );
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers (reverse order)
DROP TRIGGER IF EXISTS record_complexity_change;
DROP TRIGGER IF EXISTS record_priority_change;
DROP TRIGGER IF EXISTS record_status_change;
DROP TRIGGER IF EXISTS record_ticket_creation;
DROP TRIGGER IF EXISTS generate_ticket_number;
DROP TRIGGER IF EXISTS update_project_timestamp;
DROP TRIGGER IF EXISTS update_ticket_timestamp;

-- Drop views
DROP VIEW IF EXISTS field_change_history;
DROP VIEW IF EXISTS active_claims;
DROP VIEW IF EXISTS pending_human_input;
DROP VIEW IF EXISTS workable_tickets;

-- Drop indexes (will be dropped with tables, but explicit for clarity)
DROP INDEX IF EXISTS idx_ticket_tasks_ticket;
DROP INDEX IF EXISTS idx_activity_log_created_at;
DROP INDEX IF EXISTS idx_activity_log_action;
DROP INDEX IF EXISTS idx_activity_log_ticket_id;
DROP INDEX IF EXISTS idx_inbox_ticket;
DROP INDEX IF EXISTS idx_inbox_pending;
DROP INDEX IF EXISTS idx_claims_one_active;
DROP INDEX IF EXISTS idx_claims_expires;
DROP INDEX IF EXISTS idx_claims_ticket_active;
DROP INDEX IF EXISTS idx_dependencies_depends_on;
DROP INDEX IF EXISTS idx_tickets_project_status;
DROP INDEX IF EXISTS idx_tickets_parent;
DROP INDEX IF EXISTS idx_tickets_priority;
DROP INDEX IF EXISTS idx_tickets_status;
DROP INDEX IF EXISTS idx_tickets_project_id;
DROP INDEX IF EXISTS idx_projects_key;

-- Drop tables (reverse dependency order)
DROP TABLE IF EXISTS ticket_tasks;
DROP TABLE IF EXISTS activity_log;
DROP TABLE IF EXISTS inbox_messages;
DROP TABLE IF EXISTS claims;
DROP TABLE IF EXISTS ticket_dependencies;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS milestones;
DROP TABLE IF EXISTS projects;

-- +goose StatementEnd
