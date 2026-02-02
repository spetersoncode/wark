-- +goose Up
-- +goose StatementBegin
CREATE TABLE claims (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id),
    worker_id       TEXT NOT NULL,
    claimed_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME NOT NULL,
    released_at     DATETIME,

    status          TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN (
                        'active',
                        'completed',
                        'expired',
                        'released'
                    ))
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS claims;
