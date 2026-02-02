-- +goose Up
-- +goose StatementBegin
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

                        -- Dependency actions
                        'dependency_added',
                        'dependency_removed',
                        'blocked',
                        'unblocked',

                        -- Decomposition
                        'decomposed',
                        'child_created',

                        -- Human interaction
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
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS activity_log;
