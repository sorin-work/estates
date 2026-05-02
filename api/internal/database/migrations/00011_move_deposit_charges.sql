-- +goose Up
-- Deposits used to be added as a charge and paid like rent, which counted
-- them as rent collected. Move every deposit charge that was paid into
-- deposit_transactions, and remove the payment that covered it: the
-- closest payment of the same amount within a month of the due date.
-- Deposit charges that were never paid are left alone.
CREATE TEMPORARY TABLE deposit_moves ON COMMIT DROP AS
SELECT DISTINCT ON (c.id)
       c.id AS charge_id, c.lease_id, c.amount_cents, pm.id AS payment_id, pm.paid_on,
       COALESCE(NULLIF(pm.notes, ''), NULLIF(pm.reference, ''), c.description) AS notes
  FROM charges c
  JOIN payments pm ON pm.lease_id = c.lease_id
                  AND pm.amount_cents = c.amount_cents
                  AND pm.paid_on BETWEEN c.due_date - 31 AND c.due_date + 31
 WHERE c.kind = 'deposit'
 ORDER BY c.id, abs(pm.paid_on - c.due_date), pm.created_at;

INSERT INTO deposit_transactions (lease_id, kind, amount_cents, occurred_on, notes)
SELECT lease_id, 'received', amount_cents, paid_on, notes FROM deposit_moves;

DELETE FROM payments WHERE id IN (SELECT payment_id FROM deposit_moves);
DELETE FROM charges WHERE id IN (SELECT charge_id FROM deposit_moves);

-- +goose Down
-- The moved deposits stay where they are.
SELECT 1;
