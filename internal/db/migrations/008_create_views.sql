-- +goose Up
-- Workable tickets view: tickets ready to be picked up by an agent
-- +goose StatementBegin
CREATE VIEW workable_tickets AS
SELECT t.*,
       p.key AS project_key,
       p.key || '-' || t.number AS ticket_key
FROM tickets t
JOIN projects p ON t.project_id = p.id
WHERE t.status = 'ready'
  AND NOT EXISTS (
      SELECT 1 FROM ticket_dependencies td
      JOIN tickets dep ON td.depends_on_id = dep.id
      WHERE td.ticket_id = t.id
        AND dep.status NOT IN ('done', 'cancelled')
  )
ORDER BY
    CASE t.priority
        WHEN 'highest' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        WHEN 'lowest' THEN 5
    END,
    t.created_at;
-- +goose StatementEnd

-- Pending human input view
-- +goose StatementBegin
CREATE VIEW pending_human_input AS
SELECT
    im.*,
    t.title AS ticket_title,
    p.key || '-' || t.number AS ticket_key
FROM inbox_messages im
JOIN tickets t ON im.ticket_id = t.id
JOIN projects p ON t.project_id = p.id
WHERE im.responded_at IS NULL
ORDER BY im.created_at;
-- +goose StatementEnd

-- Active claims view
-- +goose StatementBegin
CREATE VIEW active_claims AS
SELECT
    c.*,
    t.title AS ticket_title,
    p.key || '-' || t.number AS ticket_key,
    CAST((julianday(c.expires_at) - julianday('now')) * 24 * 60 AS INTEGER) AS minutes_remaining
FROM claims c
JOIN tickets t ON c.ticket_id = t.id
JOIN projects p ON t.project_id = p.id
WHERE c.status = 'active'
  AND c.expires_at > CURRENT_TIMESTAMP;
-- +goose StatementEnd

-- Field change history view
-- +goose StatementBegin
CREATE VIEW field_change_history AS
SELECT
    id,
    ticket_id,
    json_extract(details, '$.field') AS field_name,
    json_extract(details, '$.old') AS old_value,
    json_extract(details, '$.new') AS new_value,
    actor_type || COALESCE(':' || actor_id, '') AS changed_by,
    created_at
FROM activity_log
WHERE action = 'field_changed';
-- +goose StatementEnd

-- +goose Down
DROP VIEW IF EXISTS field_change_history;
DROP VIEW IF EXISTS active_claims;
DROP VIEW IF EXISTS pending_human_input;
DROP VIEW IF EXISTS workable_tickets;
