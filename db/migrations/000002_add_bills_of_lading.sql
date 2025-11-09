-- +goose Up
CREATE TABLE IF NOT EXISTS shipman.bills_of_lading (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID REFERENCES shipman.charter_details(id) ON DELETE CASCADE,
    voyage_id UUID REFERENCES shipman.voyages(id) ON DELETE SET NULL,
    document_number TEXT NOT NULL,
    issue_date DATE,
    issuer TEXT,
    consignee TEXT,
    notify_party TEXT,
    cargo_description TEXT,
    quantity NUMERIC(12,2),
    quantity_unit TEXT,
    storage_uri TEXT,
    checksum TEXT,
    encrypted_key BYTEA,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS shipman.bills_of_lading;

