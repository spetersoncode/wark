-- +goose Up
-- +goose StatementBegin
ALTER TABLE tickets DROP COLUMN brain;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tickets ADD COLUMN brain TEXT;
-- +goose StatementEnd
