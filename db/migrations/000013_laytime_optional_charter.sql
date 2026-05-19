-- +goose Up
-- Allow laytime_entries to exist without a charter_detail (when created from voyage/deal flow)
ALTER TABLE shipman.laytime_entries ALTER COLUMN charter_detail_id DROP NOT NULL;

-- +goose Down
-- Note: this may fail if NULLs exist
ALTER TABLE shipman.laytime_entries ALTER COLUMN charter_detail_id SET NOT NULL;
