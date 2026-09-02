-- +goose Up
ALTER TABLE team_sheet_entries ADD COLUMN specialization TEXT;

-- +goose Down
ALTER TABLE team_sheet_entries DROP COLUMN specialization;
