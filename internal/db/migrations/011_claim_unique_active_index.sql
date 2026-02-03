-- +goose Up
-- +goose StatementBegin
-- Prevent race conditions where two agents can claim the same ticket simultaneously.
-- The pre-check SELECT is still useful for UX (shows who has the claim), but this
-- index serves as the actual mutex at the database level.
CREATE UNIQUE INDEX idx_claims_one_active ON claims(ticket_id) WHERE status = 'active';
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_claims_one_active;
