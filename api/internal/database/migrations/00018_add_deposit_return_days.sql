-- +goose Up
-- How many days after a tenant moves out the landlord has to return the
-- deposit. It depends on the state, so each landlord sets their own.
ALTER TABLE users ADD COLUMN deposit_return_days integer NOT NULL DEFAULT 30
    CHECK (deposit_return_days BETWEEN 1 AND 90);

-- +goose Down
ALTER TABLE users DROP COLUMN deposit_return_days;
