-- +goose Up
-- Every account needs a real zone to work out its "today"; an empty one is
-- refused outright.
UPDATE users SET time_zone = 'UTC' WHERE time_zone = '';
ALTER TABLE users ADD CONSTRAINT users_time_zone_not_empty CHECK (time_zone <> '');

-- +goose Down
ALTER TABLE users DROP CONSTRAINT users_time_zone_not_empty;
