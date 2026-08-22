-- +goose Up
ALTER TABLE team_sheet_entries ADD COLUMN realm TEXT;

-- +goose Down
ALTER TABLE team_sheet_entries DROP COLUMN realm;
