-- +goose Up
-- The landlord's IANA time zone. It decides what "today" is: when rent falls
-- due, when it is overdue and which month is "this month". UTC keeps the
-- behaviour from before the setting existed.
ALTER TABLE users ADD COLUMN time_zone text NOT NULL DEFAULT 'UTC';

-- +goose Down
ALTER TABLE users DROP COLUMN time_zone;
