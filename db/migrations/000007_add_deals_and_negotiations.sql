-- +goose Up

-- Deals represent charter party negotiations between parties
CREATE TABLE IF NOT EXISTS shipman.deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    document_id UUID REFERENCES shipman.documents(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'active', -- active | completed | cancelled
    created_by UUID NOT NULL REFERENCES shipman.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Deal participants with their roles
CREATE TABLE IF NOT EXISTS shipman.deal_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id UUID NOT NULL REFERENCES shipman.deals(id) ON DELETE CASCADE,
    user_id UUID REFERENCES shipman.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('shipowner', 'charterer', 'broker')),
    invited_by UUID REFERENCES shipman.users(id),
    invite_email TEXT,
    joined_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(deal_id, user_id)
);

-- Invite tokens for joining deals
CREATE TABLE IF NOT EXISTS shipman.deal_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id UUID NOT NULL REFERENCES shipman.deals(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('shipowner', 'charterer', 'broker')),
    created_by UUID NOT NULL REFERENCES shipman.users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    used_by UUID REFERENCES shipman.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Clause negotiations track proposals on specific clauses
CREATE TABLE IF NOT EXISTS shipman.clause_negotiations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id UUID NOT NULL REFERENCES shipman.deals(id) ON DELETE CASCADE,
    clause_type TEXT NOT NULL,
    clause_title TEXT NOT NULL,
    original_content TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending | accepted | rejected | countered
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Clause proposals are individual changes proposed by parties
CREATE TABLE IF NOT EXISTS shipman.clause_proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    negotiation_id UUID NOT NULL REFERENCES shipman.clause_negotiations(id) ON DELETE CASCADE,
    proposed_by UUID NOT NULL REFERENCES shipman.users(id) ON DELETE CASCADE,
    proposed_content TEXT NOT NULL,
    comment TEXT,
    status TEXT NOT NULL DEFAULT 'pending', -- pending | accepted | rejected | superseded
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deals_created_by ON shipman.deals(created_by);
CREATE INDEX idx_deal_participants_deal_id ON shipman.deal_participants(deal_id);
CREATE INDEX idx_deal_participants_user_id ON shipman.deal_participants(user_id);
CREATE INDEX idx_deal_invites_token ON shipman.deal_invites(token);
CREATE INDEX idx_clause_negotiations_deal_id ON shipman.clause_negotiations(deal_id);
CREATE INDEX idx_clause_proposals_negotiation_id ON shipman.clause_proposals(negotiation_id);

CREATE TRIGGER trg_deals_updated_at
    BEFORE UPDATE ON shipman.deals
    FOR EACH ROW
    EXECUTE FUNCTION shipman.set_updated_at();

CREATE TRIGGER trg_clause_negotiations_updated_at
    BEFORE UPDATE ON shipman.clause_negotiations
    FOR EACH ROW
    EXECUTE FUNCTION shipman.set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trg_clause_negotiations_updated_at ON shipman.clause_negotiations;
DROP TRIGGER IF EXISTS trg_deals_updated_at ON shipman.deals;
DROP TABLE IF EXISTS shipman.clause_proposals;
DROP TABLE IF EXISTS shipman.clause_negotiations;
DROP TABLE IF EXISTS shipman.deal_invites;
DROP TABLE IF EXISTS shipman.deal_participants;
DROP TABLE IF EXISTS shipman.deals;
