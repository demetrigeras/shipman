-- +goose Up

ALTER TABLE shipman.clause_negotiations ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;

ALTER TABLE shipman.clause_negotiations
    ADD CONSTRAINT uq_deal_clause UNIQUE (deal_id, clause_title);

-- +goose Down

ALTER TABLE shipman.clause_negotiations DROP CONSTRAINT IF EXISTS uq_deal_clause;

ALTER TABLE shipman.clause_negotiations DROP COLUMN IF EXISTS sort_order;
