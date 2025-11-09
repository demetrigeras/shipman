-- +goose Up
CREATE TABLE IF NOT EXISTS shipman.cargo_loads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id UUID NOT NULL REFERENCES shipman.voyages(id) ON DELETE CASCADE,
    load_port TEXT,
    discharge_port TEXT,
    commodity TEXT,
    quantity NUMERIC(12,2),
    unit TEXT,
    stowage_plan JSONB,
    hazardous BOOLEAN,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS shipman.cargo_loads;

