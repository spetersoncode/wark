-- +goose Up
-- +goose StatementBegin

-- =============================================================================
-- Roles Table
-- =============================================================================
-- Stores roles for use in agent execution context.
-- Roles define different agent personas/capabilities that can be applied when
-- working on tickets (e.g., "senior-engineer", "code-reviewer", "architect").
-- =============================================================================

CREATE TABLE roles (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL UNIQUE,      -- Unique identifier (e.g., "senior-engineer")
    description     TEXT NOT NULL,             -- Human-readable description
    instructions    TEXT NOT NULL,             -- System prompt/contextual instructions
    is_builtin      BOOLEAN NOT NULL DEFAULT FALSE, -- Built-in vs user-defined role
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index on name for fast lookups (unique constraint already provides this)
CREATE UNIQUE INDEX idx_roles_name ON roles(name);

-- Index on is_builtin to filter built-in vs user-defined roles
CREATE INDEX idx_roles_is_builtin ON roles(is_builtin);

-- Trigger to auto-update updated_at timestamp
CREATE TRIGGER update_role_timestamp
AFTER UPDATE ON roles
FOR EACH ROW
BEGIN
    UPDATE roles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS update_role_timestamp;
DROP INDEX IF EXISTS idx_roles_is_builtin;
DROP INDEX IF EXISTS idx_roles_name;
DROP TABLE IF EXISTS roles;

-- +goose StatementEnd
