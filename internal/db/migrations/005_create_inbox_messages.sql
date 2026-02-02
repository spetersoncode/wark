-- +goose Up
-- +goose StatementBegin
CREATE TABLE inbox_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id),

    -- Message classification
    message_type    TEXT NOT NULL DEFAULT 'question'
                    CHECK (message_type IN (
                        'question',
                        'decision',
                        'review',
                        'escalation',
                        'info'
                    )),

    -- Content
    content         TEXT NOT NULL,
    from_agent      TEXT,

    -- Response
    response        TEXT,
    responded_at    DATETIME,

    -- Timestamps
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS inbox_messages;
