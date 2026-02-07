-- +goose Up
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

-- Recreate tickets table with updated status CHECK constraint
CREATE TABLE tickets_new (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id          INTEGER NOT NULL REFERENCES projects(id),
    number              INTEGER NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT,
    status              TEXT NOT NULL DEFAULT 'backlog'
                        CHECK (status IN (
                            'backlog',
                            'blocked',
                            'ready',
                            'working',
                            'human',
                            'review',
                            'reviewing',
                            'closed'
                        )),
    resolution          TEXT
                        CHECK (resolution IS NULL OR resolution IN (
                            'completed',
                            'wont_do',
                            'duplicate',
                            'invalid',
                            'obsolete'
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
    type                TEXT NOT NULL DEFAULT 'task'
                        CHECK (type IN ('task', 'epic')),
    retry_count         INTEGER NOT NULL DEFAULT 0,
    max_retries         INTEGER NOT NULL DEFAULT 3,
    epic_id             INTEGER REFERENCES tickets(id),
    parent_ticket_id    INTEGER REFERENCES tickets(id),
    milestone_id        INTEGER REFERENCES milestones(id),
    role_id             INTEGER REFERENCES roles(id),
    worktree            TEXT,
    claim_id            INTEGER REFERENCES claims(id),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at        DATETIME,
    FOREIGN KEY (project_id, number) REFERENCES ticket_sequences(project_id, last_number)
);

-- Copy all data from old table
INSERT INTO tickets_new SELECT * FROM tickets;

-- Drop old table
DROP TABLE tickets;

-- Rename new table
ALTER TABLE tickets_new RENAME TO tickets;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

-- Revert to old status constraint (backlog and reviewing not allowed)
CREATE TABLE tickets_new (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id          INTEGER NOT NULL REFERENCES projects(id),
    number              INTEGER NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT,
    status              TEXT NOT NULL DEFAULT 'ready'
                        CHECK (status IN (
                            'blocked',
                            'ready',
                            'working',
                            'human',
                            'review',
                            'closed'
                        )),
    resolution          TEXT
                        CHECK (resolution IS NULL OR resolution IN (
                            'completed',
                            'wont_do',
                            'duplicate',
                            'invalid',
                            'obsolete'
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
    type                TEXT NOT NULL DEFAULT 'task'
                        CHECK (type IN ('task', 'epic')),
    retry_count         INTEGER NOT NULL DEFAULT 0,
    max_retries         INTEGER NOT NULL DEFAULT 3,
    epic_id             INTEGER REFERENCES tickets(id),
    parent_ticket_id    INTEGER REFERENCES tickets(id),
    milestone_id        INTEGER REFERENCES milestones(id),
    role_id             INTEGER REFERENCES roles(id),
    worktree            TEXT,
    claim_id            INTEGER REFERENCES claims(id),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at        DATETIME,
    FOREIGN KEY (project_id, number) REFERENCES ticket_sequences(project_id, last_number)
);

-- Copy data (will fail if any tickets have 'backlog' or 'reviewing' status)
INSERT INTO tickets_new SELECT * FROM tickets;

DROP TABLE tickets;
ALTER TABLE tickets_new RENAME TO tickets;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd
