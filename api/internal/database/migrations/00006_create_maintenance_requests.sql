-- +goose Up
CREATE TABLE maintenance_requests (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id  uuid NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
    unit_id      uuid REFERENCES units (id) ON DELETE SET NULL,
    title        text NOT NULL,
    description  text NOT NULL DEFAULT '',
    priority     text NOT NULL DEFAULT 'normal',
    status       text NOT NULL DEFAULT 'open',
    vendor       text NOT NULL DEFAULT '',
    reported_on  date NOT NULL DEFAULT CURRENT_DATE,
    completed_on date,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX maintenance_requests_property_id_idx ON maintenance_requests (property_id);

-- +goose Down
DROP TABLE maintenance_requests;
