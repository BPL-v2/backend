-- +goose Up
DROP TABLE IF EXISTS unique_item_tracking;
DROP TYPE IF EXISTS unique_item_source;

-- +goose Down
CREATE TYPE unique_item_source AS ENUM ('public_stash', 'guild_stash', 'character');

CREATE TABLE unique_item_tracking (
    item_id text NOT NULL,
    item_ref_id int4 NOT NULL,
    team_id int4 NOT NULL,
    player_id int4 NULL,
    event_id int4 NOT NULL,
    source unique_item_source NOT NULL,
    "timestamp" timestamptz NOT NULL,
    CONSTRAINT unique_item_tracking_item_ref_fk FOREIGN KEY (item_ref_id) REFERENCES items(id),
    CONSTRAINT unique_item_tracking_team_fk FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT unique_item_tracking_player_fk FOREIGN KEY (player_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT unique_item_tracking_event_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);
CREATE INDEX unique_item_tracking_item_id_idx ON unique_item_tracking USING btree (item_id);
CREATE INDEX unique_item_tracking_team_id_idx ON unique_item_tracking USING btree (team_id);
CREATE INDEX unique_item_tracking_event_id_idx ON unique_item_tracking USING btree (event_id);
