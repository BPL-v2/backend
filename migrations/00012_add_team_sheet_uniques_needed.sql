-- +goose Up
ALTER TABLE team_sheet_entries ADD COLUMN uniques_needed TEXT;

-- +goose Down
ALTER TABLE team_sheet_entries DROP COLUMN uniques_needed;
