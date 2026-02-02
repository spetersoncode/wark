-- +goose Up
-- Projects indexes
CREATE UNIQUE INDEX idx_projects_key ON projects(key);

-- Tickets indexes
CREATE INDEX idx_tickets_project_id ON tickets(project_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_parent ON tickets(parent_ticket_id);
CREATE INDEX idx_tickets_priority_status ON tickets(priority, status);
CREATE UNIQUE INDEX idx_tickets_project_number ON tickets(project_id, number);

-- Dependencies indexes
CREATE INDEX idx_dependencies_depends_on ON ticket_dependencies(depends_on_id);

-- Claims indexes
CREATE INDEX idx_claims_ticket_active ON claims(ticket_id, status);
CREATE INDEX idx_claims_expires ON claims(expires_at);

-- Inbox messages indexes
CREATE INDEX idx_inbox_pending ON inbox_messages(responded_at);
CREATE INDEX idx_inbox_ticket ON inbox_messages(ticket_id);

-- Activity log indexes
CREATE INDEX idx_activity_ticket ON activity_log(ticket_id, created_at);
CREATE INDEX idx_activity_action ON activity_log(action);
CREATE INDEX idx_activity_actor ON activity_log(actor_type, actor_id);

-- +goose Down
-- Activity log indexes
DROP INDEX IF EXISTS idx_activity_actor;
DROP INDEX IF EXISTS idx_activity_action;
DROP INDEX IF EXISTS idx_activity_ticket;

-- Inbox messages indexes
DROP INDEX IF EXISTS idx_inbox_ticket;
DROP INDEX IF EXISTS idx_inbox_pending;

-- Claims indexes
DROP INDEX IF EXISTS idx_claims_expires;
DROP INDEX IF EXISTS idx_claims_ticket_active;

-- Dependencies indexes
DROP INDEX IF EXISTS idx_dependencies_depends_on;

-- Tickets indexes
DROP INDEX IF EXISTS idx_tickets_project_number;
DROP INDEX IF EXISTS idx_tickets_priority_status;
DROP INDEX IF EXISTS idx_tickets_parent;
DROP INDEX IF EXISTS idx_tickets_status;
DROP INDEX IF EXISTS idx_tickets_project_id;

-- Projects indexes
DROP INDEX IF EXISTS idx_projects_key;
