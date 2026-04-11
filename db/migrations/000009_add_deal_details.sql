-- +goose Up

-- Vessel details attached to a specific deal (filled in by shipowner/broker)
CREATE TABLE IF NOT EXISTS shipman.deal_vessel_details (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id UUID NOT NULL UNIQUE REFERENCES shipman.deals(id) ON DELETE CASCADE,
    filled_by UUID NOT NULL REFERENCES shipman.users(id),
    vessel_name TEXT,
    imo_number TEXT,
    vessel_type TEXT,
    flag_state TEXT,
    deadweight_tonnage NUMERIC(12,2),
    gross_tonnage NUMERIC(12,2),
    build_year SMALLINT,
    class_society TEXT,
    current_position TEXT,
    available_from DATE,
    asking_rate NUMERIC(12,2),
    asking_rate_currency CHAR(3) DEFAULT 'USD',
    asking_rate_type TEXT DEFAULT 'per_day',  -- per_day | lumpsum
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cargo details attached to a specific deal (filled in by charterer/broker)
CREATE TABLE IF NOT EXISTS shipman.deal_cargo_details (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id UUID NOT NULL UNIQUE REFERENCES shipman.deals(id) ON DELETE CASCADE,
    filled_by UUID NOT NULL REFERENCES shipman.users(id),
    commodity TEXT,
    quantity NUMERIC(12,2),
    quantity_unit TEXT DEFAULT 'MT',
    load_port TEXT,
    discharge_port TEXT,
    laycan_from DATE,
    laycan_to DATE,
    freight_idea NUMERIC(12,2),
    freight_currency CHAR(3) DEFAULT 'USD',
    freight_type TEXT DEFAULT 'per_mt',  -- per_mt | lumpsum | per_day
    special_requirements TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_deal_vessel_details_updated_at
    BEFORE UPDATE ON shipman.deal_vessel_details
    FOR EACH ROW EXECUTE FUNCTION shipman.set_updated_at();

CREATE TRIGGER trg_deal_cargo_details_updated_at
    BEFORE UPDATE ON shipman.deal_cargo_details
    FOR EACH ROW EXECUTE FUNCTION shipman.set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trg_deal_cargo_details_updated_at ON shipman.deal_cargo_details;
DROP TRIGGER IF EXISTS trg_deal_vessel_details_updated_at ON shipman.deal_vessel_details;
DROP TABLE IF EXISTS shipman.deal_cargo_details;
DROP TABLE IF EXISTS shipman.deal_vessel_details;
