-- +goose Up
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS document_id UUID REFERENCES shipman.documents(id) ON DELETE SET NULL;
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS charter_type TEXT; -- 'time_charter' | 'voyage_charter' | 'bareboat'

-- +goose Down
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS document_id;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS charter_type;
