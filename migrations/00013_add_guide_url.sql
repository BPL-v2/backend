-- +goose Up
ALTER TABLE team_sheet_entries ADD COLUMN guide_url TEXT;

-- +goose Down
ALTER TABLE team_sheet_entries DROP COLUMN guide_url;
