-- +goose Up
CREATE TABLE recurring_expenses (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id       uuid NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
    vendor_id         uuid REFERENCES vendors (id) ON DELETE SET NULL,
    vendor            text NOT NULL DEFAULT '',
    category          text NOT NULL,
    amount_cents      bigint NOT NULL CHECK (amount_cents > 0),
    description       text NOT NULL DEFAULT '',
    day_of_month      integer NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
    every_months      integer NOT NULL DEFAULT 1 CHECK (every_months IN (1, 3, 6, 12)),
    start_date        date NOT NULL,
    end_date          date CHECK (end_date >= start_date),
    -- The last occurrence an expense was created for. Expenses deleted by
    -- hand are not created again.
    generated_through date,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX recurring_expenses_property_id_idx ON recurring_expenses (property_id);

ALTER TABLE expenses ADD COLUMN recurring_expense_id uuid REFERENCES recurring_expenses (id) ON DELETE SET NULL;
CREATE UNIQUE INDEX expenses_one_per_occurrence ON expenses (recurring_expense_id, spent_on)
    WHERE recurring_expense_id IS NOT NULL;

-- +goose Down
DROP INDEX expenses_one_per_occurrence;
ALTER TABLE expenses DROP COLUMN recurring_expense_id;
DROP TABLE recurring_expenses;
