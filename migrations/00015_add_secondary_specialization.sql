-- +goose Up
ALTER TABLE team_sheet_entries ADD COLUMN secondary_specialization TEXT;

-- +goose Down
ALTER TABLE team_sheet_entries DROP COLUMN secondary_specialization;
