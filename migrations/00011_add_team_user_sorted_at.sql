-- +goose Up
ALTER TABLE team_users ADD COLUMN sorted_at TIMESTAMPTZ;

UPDATE team_users tu
SET sorted_at = s.timestamp
FROM signups s, teams t
WHERE t.id = tu.team_id AND s.event_id = t.event_id AND s.user_id = tu.user_id;

UPDATE team_users SET sorted_at = now() WHERE sorted_at IS NULL;

ALTER TABLE team_users ALTER COLUMN sorted_at SET DEFAULT now();
ALTER TABLE team_users ALTER COLUMN sorted_at SET NOT NULL;

-- +goose Down
ALTER TABLE team_users DROP COLUMN sorted_at;
