-- +goose Up
-- Receipts are numbered per landlord in the order payments came in.
ALTER TABLE payments ADD COLUMN receipt_number integer;

UPDATE payments pm
   SET receipt_number = n.rn
  FROM (SELECT pm.id, row_number() OVER (PARTITION BY p.owner_id ORDER BY pm.paid_on, pm.created_at) AS rn
          FROM payments pm
          JOIN leases l ON l.id = pm.lease_id
          JOIN units u ON u.id = l.unit_id
          JOIN properties p ON p.id = u.property_id) n
 WHERE n.id = pm.id;

ALTER TABLE payments ALTER COLUMN receipt_number SET NOT NULL;

-- +goose Down
ALTER TABLE payments DROP COLUMN receipt_number;
