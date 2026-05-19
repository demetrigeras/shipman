-- +goose Up

ALTER TABLE shipman.users
    ADD COLUMN IF NOT EXISTS coinsub_merchant_id TEXT,
    ADD COLUMN IF NOT EXISTS wallet_address TEXT;

-- +goose Down

ALTER TABLE shipman.users
    DROP COLUMN IF EXISTS wallet_address,
    DROP COLUMN IF EXISTS coinsub_merchant_id;
