-- +goose Up
CREATE TABLE vendors (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         text NOT NULL,
    trade        text NOT NULL DEFAULT 'other',
    contact_name text NOT NULL DEFAULT '',
    phone        text NOT NULL DEFAULT '',
    email        text NOT NULL DEFAULT '',
    notes        text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX vendors_owner_id_name_key ON vendors (owner_id, lower(name));

ALTER TABLE maintenance_requests ADD COLUMN vendor_id uuid REFERENCES vendors (id) ON DELETE SET NULL;
ALTER TABLE expenses ADD COLUMN vendor_id uuid REFERENCES vendors (id) ON DELETE SET NULL;
CREATE INDEX maintenance_requests_vendor_id_idx ON maintenance_requests (vendor_id);
CREATE INDEX expenses_vendor_id_idx ON expenses (vendor_id);

-- Turn the vendor names typed in so far into vendors and link them up.
INSERT INTO vendors (owner_id, name)
SELECT DISTINCT ON (p.owner_id, lower(btrim(v.name))) p.owner_id, btrim(v.name)
  FROM (SELECT property_id, vendor AS name FROM maintenance_requests
        UNION ALL
        SELECT property_id, vendor FROM expenses) v
  JOIN properties p ON p.id = v.property_id
 WHERE btrim(v.name) <> ''
 ORDER BY p.owner_id, lower(btrim(v.name)), btrim(v.name);

UPDATE maintenance_requests m
   SET vendor_id = vd.id
  FROM properties p, vendors vd
 WHERE p.id = m.property_id AND vd.owner_id = p.owner_id AND lower(vd.name) = lower(btrim(m.vendor));

UPDATE expenses e
   SET vendor_id = vd.id
  FROM properties p, vendors vd
 WHERE p.id = e.property_id AND vd.owner_id = p.owner_id AND lower(vd.name) = lower(btrim(e.vendor));

-- +goose Down
DROP INDEX expenses_vendor_id_idx;
DROP INDEX maintenance_requests_vendor_id_idx;
ALTER TABLE expenses DROP COLUMN vendor_id;
ALTER TABLE maintenance_requests DROP COLUMN vendor_id;
DROP TABLE vendors;
