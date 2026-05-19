-- +goose Up
CREATE TABLE IF NOT EXISTS shipman.voyage_invites (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id     UUID NOT NULL REFERENCES shipman.voyages(id) ON DELETE CASCADE,
    token         TEXT UNIQUE NOT NULL,
    role          TEXT NOT NULL,
    invited_email TEXT,
    created_by    UUID NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    used_at       TIMESTAMPTZ,
    used_by       UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS shipman.voyage_invites;
