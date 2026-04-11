-- +goose Up

-- Add marketplace listing fields to vessels
ALTER TABLE shipman.vessels
ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES shipman.users(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS listing_status TEXT NOT NULL DEFAULT 'unlisted', -- unlisted | for_charter | for_sale
ADD COLUMN IF NOT EXISTS asking_price NUMERIC(14,2),
ADD COLUMN IF NOT EXISTS price_currency CHAR(3) DEFAULT 'USD',
ADD COLUMN IF NOT EXISTS charter_rate_daily NUMERIC(12,2),
ADD COLUMN IF NOT EXISTS charter_rate_currency CHAR(3) DEFAULT 'USD',
ADD COLUMN IF NOT EXISTS listing_description TEXT,
ADD COLUMN IF NOT EXISTS contact_email TEXT;

CREATE INDEX idx_vessels_owner_user_id ON shipman.vessels(owner_user_id);
CREATE INDEX idx_vessels_listing_status ON shipman.vessels(listing_status);

-- Subscriptions for Coinsub integration
CREATE TABLE IF NOT EXISTS shipman.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES shipman.users(id) ON DELETE CASCADE,
    coinsub_subscription_id TEXT UNIQUE,
    plan TEXT NOT NULL DEFAULT 'basic', -- basic | premium | enterprise
    status TEXT NOT NULL DEFAULT 'active', -- active | cancelled | past_due | trialing
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user_id ON shipman.subscriptions(user_id);
CREATE INDEX idx_subscriptions_coinsub_id ON shipman.subscriptions(coinsub_subscription_id);

CREATE TRIGGER trg_subscriptions_updated_at
    BEFORE UPDATE ON shipman.subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION shipman.set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trg_subscriptions_updated_at ON shipman.subscriptions;
DROP TABLE IF EXISTS shipman.subscriptions;
DROP INDEX IF EXISTS shipman.idx_vessels_listing_status;
DROP INDEX IF EXISTS shipman.idx_vessels_owner_user_id;
ALTER TABLE shipman.vessels
DROP COLUMN IF EXISTS contact_email,
DROP COLUMN IF EXISTS listing_description,
DROP COLUMN IF EXISTS charter_rate_currency,
DROP COLUMN IF EXISTS charter_rate_daily,
DROP COLUMN IF EXISTS price_currency,
DROP COLUMN IF EXISTS asking_price,
DROP COLUMN IF EXISTS listing_status,
DROP COLUMN IF EXISTS owner_user_id;
