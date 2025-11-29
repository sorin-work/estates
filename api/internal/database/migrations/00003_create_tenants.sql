-- +goose Up
CREATE TABLE tenants (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    first_name text NOT NULL,
    last_name  text NOT NULL,
    email      text NOT NULL DEFAULT '',
    phone      text NOT NULL DEFAULT '',
    notes      text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tenants_owner_id_idx ON tenants (owner_id);

-- +goose Down
DROP TABLE tenants;
