-- +goose Up
-- +goose StatementBegin
CREATE TABLE milestones (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id      INTEGER NOT NULL REFERENCES projects(id),
    key             TEXT NOT NULL,
    name            TEXT NOT NULL,
    goal            TEXT,
    target_date     DATETIME,
    status          TEXT NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'achieved', 'abandoned')),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, key)
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS milestones;
