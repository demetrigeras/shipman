-- Shipman Starter Schema (PostgreSQL)
-- Purposefully small set of tables so the first iteration stays approachable.

CREATE SCHEMA IF NOT EXISTS shipman;
SET search_path TO shipman, public;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email CITEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user', -- user | admin (expand later)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS charter_details (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    charter_reference_code TEXT,
    vessel_name TEXT,
    counterparty_name TEXT,
    status TEXT NOT NULL DEFAULT 'draft', -- draft | active | completed | cancelled
    start_date DATE,
    end_date DATE,
    laytime_allowance_hours NUMERIC(10,2),
    demurrage_rate NUMERIC(12,2),
    demurrage_currency CHAR(3),
    fuel_clause TEXT,
    payment_terms TEXT,
    ai_status TEXT NOT NULL DEFAULT 'pending', -- pending | processing | completed | failed
    ai_document_path TEXT,
    ai_extracted_terms JSONB,
    last_reviewed_at TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS voyages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID REFERENCES charter_details(id) ON DELETE CASCADE,
    voyage_number TEXT,
    vessel_name TEXT,
    departure_port TEXT,
    arrival_port TEXT,
    planned_departure_at TIMESTAMPTZ,
    planned_arrival_at TIMESTAMPTZ,
    actual_departure_at TIMESTAMPTZ,
    actual_arrival_at TIMESTAMPTZ,
    distance_nm NUMERIC(12,2),
    time_at_sea_hours NUMERIC(12,2),
    fuel_consumed_mt NUMERIC(12,2),
    fuel_type TEXT,
    weather_summary TEXT,
    status TEXT NOT NULL DEFAULT 'planned', -- planned | sailing | completed | delayed
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS laytime_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID REFERENCES charter_details(id) ON DELETE CASCADE,
    voyage_id UUID REFERENCES voyages(id) ON DELETE SET NULL,
    port_name TEXT,
    activity TEXT, -- loading | discharging | waiting | weather_delay
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    hours_counted NUMERIC(10,2),
    remarks TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS voyage_ports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id UUID REFERENCES voyages(id) ON DELETE CASCADE,
    port_name TEXT NOT NULL,
    port_country TEXT,
    port_unlocode TEXT,
    latitude NUMERIC(9,6),
    longitude NUMERIC(9,6),
    arrived_at TIMESTAMPTZ,
    departed_at TIMESTAMPTZ,
    laytime_hours NUMERIC(10,2),
    cargo_operations TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ship_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id UUID REFERENCES voyages(id) ON DELETE CASCADE,
    recorded_at TIMESTAMPTZ NOT NULL,
    latitude NUMERIC(9,6) NOT NULL,
    longitude NUMERIC(9,6) NOT NULL,
    speed_knots NUMERIC(8,3),
    heading NUMERIC(8,3),
    distance_logged_nm NUMERIC(12,2),
    fuel_remaining_mt NUMERIC(12,2),
    source TEXT NOT NULL DEFAULT 'manual', -- manual | ais | api
    remarks TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID REFERENCES charter_details(id) ON DELETE CASCADE,
    voyage_id UUID REFERENCES voyages(id) ON DELETE SET NULL,
    category TEXT NOT NULL DEFAULT 'general', -- hire | bunker | demurrage | escrow | other
    due_date DATE,
    paid_at TIMESTAMPTZ,
    amount NUMERIC(12,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status TEXT NOT NULL DEFAULT 'pending', -- pending | paid | overdue | disputed
    payment_method TEXT,
    reference TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS disputes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID REFERENCES charter_details(id) ON DELETE CASCADE,
    voyage_id UUID REFERENCES voyages(id) ON DELETE SET NULL,
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
    laytime_entry_id UUID REFERENCES laytime_entries(id) ON DELETE SET NULL,
    raised_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    subject TEXT NOT NULL,
    description TEXT,
    claimed_amount NUMERIC(12,2),
    currency CHAR(3),
    status TEXT NOT NULL DEFAULT 'open', -- open | under_review | resolved | closed
    resolution_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bills_of_lading (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID REFERENCES charter_details(id) ON DELETE CASCADE,
    voyage_id UUID REFERENCES voyages(id) ON DELETE SET NULL,
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

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_charter_details_updated_at
    BEFORE UPDATE ON charter_details
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_voyages_updated_at
    BEFORE UPDATE ON voyages
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_voyage_ports_updated_at
    BEFORE UPDATE ON voyage_ports
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_ship_positions_updated_at
    BEFORE UPDATE ON ship_positions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_laytime_entries_updated_at
    BEFORE UPDATE ON laytime_entries
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_disputes_updated_at
    BEFORE UPDATE ON disputes
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS bills_of_lading (
CREATE TRIGGER trg_bills_of_lading_updated_at
    BEFORE UPDATE ON bills_of_lading
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS demurrage_records (
CREATE TABLE IF NOT EXISTS demurrage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID NOT NULL REFERENCES charter_details(id) ON DELETE CASCADE,
    voyage_id UUID REFERENCES voyages(id) ON DELETE SET NULL,
    laytime_entry_id UUID REFERENCES laytime_entries(id) ON DELETE SET NULL,
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

CREATE TRIGGER trg_demurrage_records_updated_at
    BEFORE UPDATE ON demurrage_records
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS vessels (
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

CREATE TRIGGER trg_vessels_updated_at
    BEFORE UPDATE ON vessels
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS cargo_loads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id UUID NOT NULL REFERENCES voyages(id) ON DELETE CASCADE,
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

CREATE TRIGGER trg_cargo_loads_updated_at
    BEFORE UPDATE ON cargo_loads
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- End of starter schema ------------------------------------------------------

