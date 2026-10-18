-- +goose Up
-- Maintenance lists sum the expenses of every request, and vendors and
-- units are looked up from their jobs; none of these had an index.
CREATE INDEX expenses_maintenance_request_id_idx ON expenses (maintenance_request_id)
    WHERE maintenance_request_id IS NOT NULL;
CREATE INDEX maintenance_requests_unit_id_idx ON maintenance_requests (unit_id) WHERE unit_id IS NOT NULL;
CREATE INDEX deposit_transactions_kind_idx ON deposit_transactions (kind, occurred_on);

-- +goose Down
DROP INDEX deposit_transactions_kind_idx;
DROP INDEX maintenance_requests_unit_id_idx;
DROP INDEX expenses_maintenance_request_id_idx;
