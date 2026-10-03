-- +goose Up
ALTER TABLE events ADD COLUMN max_group_size int4 NOT NULL DEFAULT 1;
UPDATE events SET max_group_size = CASE WHEN duo_signups THEN 2 ELSE 1 END;
ALTER TABLE events DROP COLUMN duo_signups;

-- turn mutual partner wishes into groups (one random key per pair)
INSERT INTO signup_groups (event_id, user_id, group_key)
SELECT p.event_id, m.user_id, p.group_key
FROM (
	SELECT s1.event_id, s1.user_id AS user_a, s2.user_id AS user_b, gen_random_uuid()::text AS group_key
	FROM signups s1
	JOIN oauths o1 ON o1.user_id = s1.user_id AND o1.provider = 'poe'
	JOIN signups s2 ON s2.event_id = s1.event_id AND s2.user_id <> s1.user_id
	JOIN oauths o2 ON o2.user_id = s2.user_id AND o2.provider = 'poe'
	WHERE lower(split_part(s1.partner_wish, '#', 1)) = lower(split_part(o2.name, '#', 1))
		AND lower(split_part(s2.partner_wish, '#', 1)) = lower(split_part(o1.name, '#', 1))
		AND s1.user_id < s2.user_id
) p
CROSS JOIN LATERAL (VALUES (p.user_a), (p.user_b)) AS m(user_id)
ON CONFLICT (event_id, user_id) DO NOTHING;

-- groups whose members were already sorted are locked
UPDATE signup_groups SET locked = true
WHERE (event_id, group_key) IN (
	SELECT sg.event_id, sg.group_key
	FROM signup_groups sg
	JOIN team_users tu ON tu.user_id = sg.user_id
	JOIN teams t ON t.id = tu.team_id AND t.event_id = sg.event_id
);

ALTER TABLE signups DROP COLUMN partner_wish;

-- +goose Down
ALTER TABLE signups ADD COLUMN partner_wish text NULL;
ALTER TABLE events ADD COLUMN duo_signups bool NOT NULL DEFAULT false;
UPDATE events SET duo_signups = max_group_size > 1;
ALTER TABLE events DROP COLUMN max_group_size;
