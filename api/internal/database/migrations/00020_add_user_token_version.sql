-- +goose Up
-- Tokens carry the version they were issued with; changing the password
-- bumps it, which signs out every other session.
ALTER TABLE users ADD COLUMN token_version integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN token_version;
