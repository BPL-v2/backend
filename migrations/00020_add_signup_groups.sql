-- +goose Up
CREATE TABLE signup_groups (
	event_id int2 NOT NULL,
	user_id int2 NOT NULL,
	group_key text NOT NULL,
	locked bool DEFAULT false NOT NULL,
	joined_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT signup_groups_pkey PRIMARY KEY (event_id, user_id),
	CONSTRAINT signup_groups_signup_fkey FOREIGN KEY (event_id, user_id) REFERENCES signups(event_id, user_id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX signup_groups_event_id_group_key_idx ON signup_groups USING btree (event_id, group_key);

-- +goose Down
DROP TABLE IF EXISTS signup_groups;
