-- +goose Up
-- The currency amounts are shown in. Amounts are still stored in cents (or
-- the currency's minor unit); this only changes how they are displayed.
ALTER TABLE users ADD COLUMN currency text NOT NULL DEFAULT 'USD' CHECK (currency ~ '^[A-Z]{3}$');
ALTER TABLE users ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE users DROP COLUMN updated_at;
ALTER TABLE users DROP COLUMN currency;
