-- +goose Up
ALTER TABLE achievements ADD COLUMN auto_check_key TEXT;
ALTER TABLE achievements ADD COLUMN event_id INT REFERENCES events(id) ON DELETE SET NULL;

UPDATE achievements SET auto_check_key = CASE name
    WHEN 'Reached level 90' THEN 'level_90'
    WHEN 'Reached level 95' THEN 'level_95'
    WHEN 'Reached level 100' THEN 'level_100'
    WHEN 'Participated in an event' THEN 'participated_in_event'
    WHEN 'Played 5 leagues' THEN 'played_5_leagues'
    WHEN 'Played 10 leagues' THEN 'played_10_leagues'
    WHEN 'Played 5 different ascendancies' THEN 'played_5_ascendancies'
    WHEN 'Played 10 different ascendancies' THEN 'played_10_ascendancies'
    WHEN 'Teamlead' THEN 'teamlead'
    WHEN 'Submitted a bounty' THEN 'submitted_bounty'
    ELSE auto_check_key
END;

-- +goose Down
ALTER TABLE achievements DROP COLUMN event_id;
ALTER TABLE achievements DROP COLUMN auto_check_key;
