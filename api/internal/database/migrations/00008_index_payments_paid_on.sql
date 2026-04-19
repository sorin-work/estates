-- +goose Up
-- The dashboard and payments list filter by month across all leases.
CREATE INDEX payments_paid_on_idx ON payments (paid_on);
CREATE INDEX maintenance_requests_status_idx ON maintenance_requests (status);

-- +goose Down
DROP INDEX maintenance_requests_status_idx;
DROP INDEX payments_paid_on_idx;
