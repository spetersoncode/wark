-- +goose Up
-- +goose StatementBegin
CREATE TABLE tickets (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id          INTEGER NOT NULL REFERENCES projects(id),
    number              INTEGER NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT,

    -- Status (state machine)
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

    -- Human input flag (reason when needs_human)
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
    parent_ticket_id    INTEGER REFERENCES tickets(id),

    -- Timestamps
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at        DATETIME,

    -- Composite unique constraint
    UNIQUE(project_id, number)
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS tickets;
