-- +goose Up
-- +goose StatementBegin

-- Add claim_id column to claims table
ALTER TABLE claims ADD COLUMN claim_id TEXT;

-- Create index for claim_id lookups
CREATE UNIQUE INDEX idx_claims_claim_id ON claims(claim_id) WHERE claim_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_claims_claim_id;
ALTER TABLE claims DROP COLUMN claim_id;

-- +goose StatementEnd
