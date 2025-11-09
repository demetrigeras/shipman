-- +goose Up
CREATE TABLE IF NOT EXISTS shipman.demurrage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID NOT NULL REFERENCES shipman.charter_details(id) ON DELETE CASCADE,
    voyage_id UUID REFERENCES shipman.voyages(id) ON DELETE SET NULL,
    laytime_entry_id UUID REFERENCES shipman.laytime_entries(id) ON DELETE SET NULL,
    claimed_hours NUMERIC(10,2),
    claimed_amount NUMERIC(12,2),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status TEXT NOT NULL DEFAULT 'draft', -- draft | submitted | settled | disputed
    reference TEXT,
    supporting_doc_uri TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS shipman.demurrage_records;

