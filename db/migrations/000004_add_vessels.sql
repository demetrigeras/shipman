-- +goose Up
CREATE TABLE IF NOT EXISTS shipman.vessels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    imo_number TEXT UNIQUE,
    flag_state TEXT,
    vessel_type TEXT,
    call_sign TEXT,
    deadweight_tonnage NUMERIC(12,2),
    gross_tonnage NUMERIC(12,2),
    net_tonnage NUMERIC(12,2),
    capacity JSONB,
    build_year SMALLINT,
    class_society TEXT,
    owner TEXT,
    manager TEXT,
    documentation_uri TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS shipman.vessels;
