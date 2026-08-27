-- +goose Up
ALTER TABLE item_wishes ADD COLUMN quantity INTEGER NOT NULL DEFAULT 1;
-- +goose Down
ALTER TABLE item_wishes DROP COLUMN quantity;
