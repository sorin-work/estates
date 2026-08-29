-- +goose Up
-- A record of changes, for "who did what and when". entity_id and lease_id
-- are not foreign keys so the history outlives deleted records.
CREATE TABLE activity (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    entity_type text NOT NULL,
    entity_id   uuid NOT NULL,
    lease_id    uuid,
    action      text NOT NULL,
    summary     text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX activity_owner_id_id_idx ON activity (owner_id, id DESC);
CREATE INDEX activity_lease_id_idx ON activity (lease_id) WHERE lease_id IS NOT NULL;

-- +goose Down
DROP TABLE activity;
