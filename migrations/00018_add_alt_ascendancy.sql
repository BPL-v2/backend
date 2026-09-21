-- +goose Up
ALTER TABLE team_sheet_entries ADD COLUMN alt_ascendancy TEXT;

-- +goose Down
ALTER TABLE team_sheet_entries DROP COLUMN alt_ascendancy;
