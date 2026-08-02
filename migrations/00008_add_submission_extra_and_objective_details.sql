-- +goose Up
ALTER TABLE submissions ADD COLUMN extra jsonb NOT NULL DEFAULT '{}';
ALTER TABLE objectives ADD COLUMN details jsonb NOT NULL DEFAULT '{}';
UPDATE objectives SET details = jsonb_build_object('tracked_value_explanation', tracked_value_explanation) WHERE tracked_value_explanation IS NOT NULL;
ALTER TABLE objectives DROP COLUMN tracked_value_explanation;

-- +goose Down
ALTER TABLE objectives ADD COLUMN tracked_value_explanation text NULL;
UPDATE objectives SET tracked_value_explanation = details->>'tracked_value_explanation';
ALTER TABLE objectives DROP COLUMN details;
ALTER TABLE submissions DROP COLUMN extra;
