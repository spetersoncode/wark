-- +goose Up
-- +goose StatementBegin
CREATE TABLE ticket_tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id   INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    description TEXT NOT NULL,
    complete    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(ticket_id, position)
);

CREATE INDEX idx_ticket_tasks_ticket ON ticket_tasks(ticket_id, position);
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_ticket_tasks_ticket;
DROP TABLE IF EXISTS ticket_tasks;
