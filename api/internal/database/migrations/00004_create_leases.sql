-- +goose Up
CREATE TABLE leases (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id        uuid NOT NULL REFERENCES units (id) ON DELETE RESTRICT,
    tenant_id      uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    start_date     date NOT NULL,
    end_date       date NOT NULL,
    rent_cents     bigint NOT NULL CHECK (rent_cents > 0),
    deposit_cents  bigint NOT NULL DEFAULT 0 CHECK (deposit_cents >= 0),
    due_day        integer NOT NULL DEFAULT 1 CHECK (due_day BETWEEN 1 AND 31),
    grace_days     integer NOT NULL DEFAULT 5 CHECK (grace_days BETWEEN 0 AND 31),
    late_fee_cents bigint NOT NULL DEFAULT 0 CHECK (late_fee_cents >= 0),
    terminated_on  date,
    notes          text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

CREATE INDEX leases_unit_id_idx ON leases (unit_id);
CREATE INDEX leases_tenant_id_idx ON leases (tenant_id);

-- +goose Down
DROP TABLE leases;
