-- +goose Up
-- Signed leases, ID copies, inspection photos and the like. Files are small
-- and few, so they live in the database with everything else and are
-- backed up with it.
CREATE TABLE documents (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id    uuid REFERENCES tenants (id) ON DELETE CASCADE,
    lease_id     uuid REFERENCES leases (id) ON DELETE CASCADE,
    property_id  uuid REFERENCES properties (id) ON DELETE CASCADE,
    filename     text NOT NULL,
    content_type text NOT NULL,
    size_bytes   bigint NOT NULL CHECK (size_bytes > 0),
    content      bytea NOT NULL,
    notes        text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (num_nonnulls(tenant_id, lease_id, property_id) = 1)
);

CREATE INDEX documents_tenant_id_idx ON documents (tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX documents_lease_id_idx ON documents (lease_id) WHERE lease_id IS NOT NULL;
CREATE INDEX documents_property_id_idx ON documents (property_id) WHERE property_id IS NOT NULL;

-- +goose Down
DROP TABLE documents;
