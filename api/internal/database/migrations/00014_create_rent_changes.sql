-- +goose Up
CREATE TABLE rent_changes (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    lease_id       uuid NOT NULL REFERENCES leases (id) ON DELETE CASCADE,
    effective_date date NOT NULL,
    rent_cents     bigint NOT NULL CHECK (rent_cents > 0),
    notes          text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (lease_id, effective_date)
);

-- A renewal is a new lease that follows on from the old one.
ALTER TABLE leases ADD COLUMN renewed_from_id uuid REFERENCES leases (id) ON DELETE SET NULL;
CREATE UNIQUE INDEX leases_renewed_from_id_key ON leases (renewed_from_id) WHERE renewed_from_id IS NOT NULL;

-- +goose Down
DROP INDEX leases_renewed_from_id_key;
ALTER TABLE leases DROP COLUMN renewed_from_id;
DROP TABLE rent_changes;
