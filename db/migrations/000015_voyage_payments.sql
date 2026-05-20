-- +goose Up

CREATE TABLE IF NOT EXISTS shipman.voyage_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id UUID NOT NULL REFERENCES shipman.voyages(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES shipman.users(id) ON DELETE CASCADE,

    -- Payment definition
    payment_type TEXT NOT NULL CHECK (payment_type IN ('hire', 'freight', 'demurrage', 'despatch', 'bunker', 'port_charges', 'other')),
    description TEXT,
    amount NUMERIC(18,2) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USDC',

    -- Recipient
    recipient_email TEXT,
    recipient_wallet TEXT,

    -- Coinsub integration
    coinsub_session_id TEXT,
    coinsub_payment_id TEXT,
    coinsub_agreement_id TEXT,
    coinsub_checkout_url TEXT,
    coinsub_tx_hash TEXT,

    -- Status
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'completed', 'failed', 'cancelled')),

    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_voyage_payments_voyage_id ON shipman.voyage_payments(voyage_id);
CREATE INDEX IF NOT EXISTS idx_voyage_payments_coinsub_session ON shipman.voyage_payments(coinsub_session_id);

DROP TRIGGER IF EXISTS trg_voyage_payments_updated_at ON shipman.voyage_payments;
CREATE TRIGGER trg_voyage_payments_updated_at
    BEFORE UPDATE ON shipman.voyage_payments
    FOR EACH ROW
    EXECUTE FUNCTION shipman.set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trg_voyage_payments_updated_at ON shipman.voyage_payments;
DROP TABLE IF EXISTS shipman.voyage_payments;
