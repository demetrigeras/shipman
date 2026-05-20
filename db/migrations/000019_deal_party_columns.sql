-- +goose Up
-- Promote the three principal deal roles to first-class columns on the deals
-- table. We keep deal_participants as the source of truth for access control
-- (it can also model multiple brokers, observers, etc.) but these columns
-- give us a fast canonical pointer to the "official" party in each role —
-- useful for listings, queries, exports, and the FE deal header.
ALTER TABLE shipman.deals
    ADD COLUMN IF NOT EXISTS shipowner_user_id UUID REFERENCES shipman.users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS charterer_user_id UUID REFERENCES shipman.users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS broker_user_id    UUID REFERENCES shipman.users(id) ON DELETE SET NULL;

-- Backfill: for every existing deal, pick the most-recently-joined
-- participant per role (NULLS LAST so a real joined_at wins over a null one).
UPDATE shipman.deals d SET shipowner_user_id = sub.user_id
FROM (
    SELECT DISTINCT ON (deal_id) deal_id, user_id
    FROM shipman.deal_participants
    WHERE role = 'shipowner' AND user_id IS NOT NULL
    ORDER BY deal_id, joined_at DESC NULLS LAST, created_at DESC
) sub
WHERE d.id = sub.deal_id AND d.shipowner_user_id IS NULL;

UPDATE shipman.deals d SET charterer_user_id = sub.user_id
FROM (
    SELECT DISTINCT ON (deal_id) deal_id, user_id
    FROM shipman.deal_participants
    WHERE role = 'charterer' AND user_id IS NOT NULL
    ORDER BY deal_id, joined_at DESC NULLS LAST, created_at DESC
) sub
WHERE d.id = sub.deal_id AND d.charterer_user_id IS NULL;

UPDATE shipman.deals d SET broker_user_id = sub.user_id
FROM (
    SELECT DISTINCT ON (deal_id) deal_id, user_id
    FROM shipman.deal_participants
    WHERE role = 'broker' AND user_id IS NOT NULL
    ORDER BY deal_id, joined_at DESC NULLS LAST, created_at DESC
) sub
WHERE d.id = sub.deal_id AND d.broker_user_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_deals_shipowner_user_id ON shipman.deals(shipowner_user_id);
CREATE INDEX IF NOT EXISTS idx_deals_charterer_user_id ON shipman.deals(charterer_user_id);
CREATE INDEX IF NOT EXISTS idx_deals_broker_user_id    ON shipman.deals(broker_user_id);

-- +goose Down
DROP INDEX IF EXISTS shipman.idx_deals_broker_user_id;
DROP INDEX IF EXISTS shipman.idx_deals_charterer_user_id;
DROP INDEX IF EXISTS shipman.idx_deals_shipowner_user_id;
ALTER TABLE shipman.deals
    DROP COLUMN IF EXISTS broker_user_id,
    DROP COLUMN IF EXISTS charterer_user_id,
    DROP COLUMN IF EXISTS shipowner_user_id;
