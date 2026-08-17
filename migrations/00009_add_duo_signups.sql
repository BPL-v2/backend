-- +goose Up
ALTER TABLE events ADD COLUMN duo_signups bool NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE events DROP COLUMN duo_signups;
