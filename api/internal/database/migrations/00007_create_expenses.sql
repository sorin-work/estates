-- +goose Up
CREATE TABLE expenses (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id            uuid NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
    maintenance_request_id uuid REFERENCES maintenance_requests (id) ON DELETE SET NULL,
    category               text NOT NULL,
    amount_cents           bigint NOT NULL CHECK (amount_cents > 0),
    spent_on               date NOT NULL,
    vendor                 text NOT NULL DEFAULT '',
    description            text NOT NULL DEFAULT '',
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX expenses_property_id_spent_on_idx ON expenses (property_id, spent_on);

-- +goose Down
DROP TABLE expenses;
