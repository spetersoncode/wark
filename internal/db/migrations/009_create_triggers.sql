-- +goose Up
-- Auto-update timestamps for tickets
-- +goose StatementBegin
CREATE TRIGGER update_ticket_timestamp
AFTER UPDATE ON tickets
FOR EACH ROW
BEGIN
    UPDATE tickets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- Auto-update timestamps for projects
-- +goose StatementBegin
CREATE TRIGGER update_project_timestamp
AFTER UPDATE ON projects
FOR EACH ROW
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- Ticket number generation
-- +goose StatementBegin
CREATE TRIGGER generate_ticket_number
AFTER INSERT ON tickets
FOR EACH ROW
WHEN NEW.number IS NULL OR NEW.number = 0
BEGIN
    UPDATE tickets
    SET number = (
        SELECT COALESCE(MAX(number), 0) + 1
        FROM tickets
        WHERE project_id = NEW.project_id
    )
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- Record ticket creation in activity log
-- +goose StatementBegin
CREATE TRIGGER record_ticket_creation
AFTER INSERT ON tickets
FOR EACH ROW
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, summary)
    VALUES (NEW.id, 'created', 'system', 'Ticket created');
END;
-- +goose StatementEnd

-- Record status changes in activity log
-- +goose StatementBegin
CREATE TRIGGER record_status_change
AFTER UPDATE OF status ON tickets
FOR EACH ROW
WHEN OLD.status != NEW.status
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id,
        'field_changed',
        'system',
        json_object('field', 'status', 'old', OLD.status, 'new', NEW.status),
        'Status: ' || OLD.status || ' -> ' || NEW.status
    );
END;
-- +goose StatementEnd

-- Record priority changes
-- +goose StatementBegin
CREATE TRIGGER record_priority_change
AFTER UPDATE OF priority ON tickets
FOR EACH ROW
WHEN OLD.priority != NEW.priority
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id,
        'field_changed',
        'system',
        json_object('field', 'priority', 'old', OLD.priority, 'new', NEW.priority),
        'Priority: ' || OLD.priority || ' -> ' || NEW.priority
    );
END;
-- +goose StatementEnd

-- Record complexity changes
-- +goose StatementBegin
CREATE TRIGGER record_complexity_change
AFTER UPDATE OF complexity ON tickets
FOR EACH ROW
WHEN OLD.complexity != NEW.complexity
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id,
        'field_changed',
        'system',
        json_object('field', 'complexity', 'old', OLD.complexity, 'new', NEW.complexity),
        'Complexity: ' || OLD.complexity || ' -> ' || NEW.complexity
    );
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS record_complexity_change;
DROP TRIGGER IF EXISTS record_priority_change;
DROP TRIGGER IF EXISTS record_status_change;
DROP TRIGGER IF EXISTS record_ticket_creation;
DROP TRIGGER IF EXISTS generate_ticket_number;
DROP TRIGGER IF EXISTS update_project_timestamp;
DROP TRIGGER IF EXISTS update_ticket_timestamp;
