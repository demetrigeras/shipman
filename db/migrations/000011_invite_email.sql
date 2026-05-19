-- +goose Up
ALTER TABLE shipman.deal_invites ADD COLUMN IF NOT EXISTS invited_email TEXT;

-- +goose Down
ALTER TABLE shipman.deal_invites DROP COLUMN IF EXISTS invited_email;
