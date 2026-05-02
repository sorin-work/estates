-- +goose Up
-- Security deposits are the tenant's money held by the landlord, not income,
-- so they are kept out of charges and payments.
CREATE TABLE deposit_transactions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    lease_id     uuid NOT NULL REFERENCES leases (id) ON DELETE CASCADE,
    kind         text NOT NULL CHECK (kind IN ('received', 'returned', 'withheld')),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    occurred_on  date NOT NULL,
    notes        text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX deposit_transactions_lease_id_idx ON deposit_transactions (lease_id);

-- +goose Down
DROP TABLE deposit_transactions;
