-- +goose Up
-- +goose StatementBegin
CREATE TABLE projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    key             TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    description     TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS projects;
