-- +goose Up
-- +goose StatementBegin
CREATE TABLE ticket_dependencies (
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    depends_on_id   INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (ticket_id, depends_on_id),

    -- Prevent self-dependency
    CHECK (ticket_id != depends_on_id)
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS ticket_dependencies;
