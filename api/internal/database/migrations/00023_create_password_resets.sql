-- +goose Up
-- One-use links for a forgotten password. Only a SHA-256 of the token is
-- kept, so a copy of the database cannot be used to reset anyone.
CREATE TABLE password_resets (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX password_resets_user_id_idx ON password_resets (user_id);

-- +goose Down
DROP TABLE password_resets;
