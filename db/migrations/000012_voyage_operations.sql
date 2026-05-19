-- +goose Up

-- Allow charter_detail_id to be null (voyages can now come from deals too)
ALTER TABLE shipman.voyages ALTER COLUMN charter_detail_id DROP NOT NULL;

-- Link voyage to a deal
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS deal_id UUID REFERENCES shipman.deals(id) ON DELETE SET NULL;

-- Who created this voyage (for access control)
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES shipman.users(id) ON DELETE SET NULL;

-- AIS / tracking
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS imo_number TEXT;
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS vessel_type TEXT;
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS dwt NUMERIC(12,2);
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS flag_state TEXT;

-- Commercial terms
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS hire_rate NUMERIC(12,2);      -- USD/day (time charter)
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS freight_rate NUMERIC(12,2);   -- USD/MT (voyage charter)
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS cargo_quantity NUMERIC(12,2); -- MT
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS cargo_type TEXT;

-- Laytime / demurrage terms
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS laytime_allowed_hours NUMERIC(10,2);
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS demurrage_rate NUMERIC(12,2); -- USD/day
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS despatch_rate NUMERIC(12,2);  -- USD/day (usually 50% of demurrage)
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS demurrage_currency CHAR(3) NOT NULL DEFAULT 'USD';

CREATE INDEX IF NOT EXISTS idx_voyages_deal_id ON shipman.voyages(deal_id);
CREATE INDEX IF NOT EXISTS idx_voyages_owner_user_id ON shipman.voyages(owner_user_id);

-- +goose Down
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS deal_id;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS imo_number;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS vessel_type;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS dwt;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS flag_state;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS hire_rate;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS freight_rate;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS cargo_quantity;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS cargo_type;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS laytime_allowed_hours;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS demurrage_rate;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS despatch_rate;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS demurrage_currency;
