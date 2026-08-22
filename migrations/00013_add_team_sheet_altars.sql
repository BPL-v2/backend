-- +goose Up
ALTER TABLE team_sheet_entries ADD COLUMN altars TEXT;

-- +goose Down
ALTER TABLE team_sheet_entries DROP COLUMN altars;
