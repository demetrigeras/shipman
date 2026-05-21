-- +goose Up
-- Voyages currently only know their `owner_user_id` (the creator). When
-- somebody accepted an invite, the backend stored their name/email as text
-- on the voyage but never linked them as a *user*, so the invited user got
-- 403 on /voyages/:id and an empty list on /voyages. This migration adds
-- proper user references for the counterparty and (optional) broker, mirroring
-- what we did for deals in 000019.
ALTER TABLE shipman.voyages
    ADD COLUMN IF NOT EXISTS counterparty_user_id UUID REFERENCES shipman.users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS broker_user_id       UUID REFERENCES shipman.users(id) ON DELETE SET NULL;

-- Backfill: where counterparty_email matches an existing user, link them.
UPDATE shipman.voyages v
SET counterparty_user_id = u.id
FROM shipman.users u
WHERE v.counterparty_user_id IS NULL
  AND v.counterparty_email IS NOT NULL
  AND lower(u.email) = lower(v.counterparty_email);

CREATE INDEX IF NOT EXISTS idx_voyages_counterparty_user_id ON shipman.voyages(counterparty_user_id);
CREATE INDEX IF NOT EXISTS idx_voyages_broker_user_id       ON shipman.voyages(broker_user_id);

-- +goose Down
DROP INDEX IF EXISTS shipman.idx_voyages_broker_user_id;
DROP INDEX IF EXISTS shipman.idx_voyages_counterparty_user_id;
ALTER TABLE shipman.voyages
    DROP COLUMN IF EXISTS broker_user_id,
    DROP COLUMN IF EXISTS counterparty_user_id;
