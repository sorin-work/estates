-- +goose Up
CREATE TABLE properties (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name          text NOT NULL,
    kind          text NOT NULL DEFAULT 'single_family',
    address_line1 text NOT NULL,
    address_line2 text NOT NULL DEFAULT '',
    city          text NOT NULL,
    region        text NOT NULL DEFAULT '',
    postal_code   text NOT NULL DEFAULT '',
    notes         text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX properties_owner_id_idx ON properties (owner_id);

CREATE TABLE units (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id       uuid NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
    label             text NOT NULL,
    bedrooms          integer NOT NULL DEFAULT 0 CHECK (bedrooms >= 0),
    bathrooms         numeric(3, 1) NOT NULL DEFAULT 1 CHECK (bathrooms >= 0),
    square_feet       integer CHECK (square_feet > 0),
    market_rent_cents bigint NOT NULL DEFAULT 0 CHECK (market_rent_cents >= 0),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (property_id, label)
);

-- +goose Down
DROP TABLE units;
DROP TABLE properties;
