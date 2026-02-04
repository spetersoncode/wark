-- +goose Up
-- +goose StatementBegin
ALTER TABLE tickets ADD COLUMN milestone_id INTEGER REFERENCES milestones(id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tickets DROP COLUMN milestone_id;
-- +goose StatementEnd
