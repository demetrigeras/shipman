-- +goose Up

-- Payment schedule fields for tracking obligations from the charter party
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS payment_frequency TEXT;          -- 'monthly', 'semi_monthly', 'lump_sum', 'on_completion'
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS first_payment_date DATE;
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS total_contract_value NUMERIC(18,2);
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS commission_rate NUMERIC(5,2);    -- broker commission %
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS bunker_cost NUMERIC(18,2);
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS port_costs NUMERIC(18,2);
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS insurance_cost NUMERIC(18,2);
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS counterparty_name TEXT;
ALTER TABLE shipman.voyages ADD COLUMN IF NOT EXISTS counterparty_email TEXT;

-- +goose Down

ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS counterparty_email;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS counterparty_name;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS insurance_cost;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS port_costs;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS bunker_cost;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS commission_rate;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS total_contract_value;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS first_payment_date;
ALTER TABLE shipman.voyages DROP COLUMN IF EXISTS payment_frequency;
