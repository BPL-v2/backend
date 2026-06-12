-- +goose Up

-- Rename old achievements table so we can migrate data
ALTER TABLE bpl2.achievements RENAME TO user_achievements_old;
ALTER INDEX bpl2.achievements_pkey RENAME TO user_achievements_old_pkey;

-- Create achievement definitions table
CREATE TABLE bpl2.achievements (
    id serial NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    is_custom bool NOT NULL DEFAULT false,
    icon bytea,
    icon_mime_type text,
    CONSTRAINT achievements_pkey PRIMARY KEY (id),
    CONSTRAINT achievements_name_key UNIQUE (name)
);

-- Seed system achievements
INSERT INTO bpl2.achievements (name) VALUES
    ('Participated in an event'),
    ('Won an event'),
    ('Teamlead'),
    ('MVP'),
    ('Played 5 leagues'),
    ('Played 10 leagues'),
    ('Reached level 90'),
    ('Reached level 95'),
    ('Reached level 100'),
    ('Submitted a bounty'),
    ('Submitted a point unique'),
    ('Played 5 different ascendancies'),
    ('Played 10 different ascendancies');

-- Create user_achievements table
CREATE TABLE bpl2.user_achievements (
    user_id int NOT NULL,
    achievement_id int NOT NULL,
    granted_at timestamptz NOT NULL DEFAULT now(),
    granted_by int,
    CONSTRAINT user_achievements_pkey PRIMARY KEY (user_id, achievement_id),
    CONSTRAINT user_achievements_user_id_fkey FOREIGN KEY (user_id) REFERENCES bpl2.users(id),
    CONSTRAINT user_achievements_achievement_id_fkey FOREIGN KEY (achievement_id) REFERENCES bpl2.achievements(id),
    CONSTRAINT user_achievements_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES bpl2.users(id)
);

-- Migrate existing data
INSERT INTO bpl2.user_achievements (user_id, achievement_id)
SELECT o.user_id, a.id
FROM bpl2.user_achievements_old o
JOIN bpl2.achievements a ON a.name = o.name
ON CONFLICT DO NOTHING;

-- Drop old table
DROP TABLE bpl2.user_achievements_old;

-- +goose Down

-- Recreate old achievements table
CREATE TABLE bpl2.user_achievements_old (
    user_id int NOT NULL,
    name text NOT NULL,
    CONSTRAINT achievements_pkey PRIMARY KEY (user_id, name)
);

-- Migrate data back
INSERT INTO bpl2.user_achievements_old (user_id, name)
SELECT ua.user_id, a.name
FROM bpl2.user_achievements ua
JOIN bpl2.achievements a ON a.id = ua.achievement_id;

DROP TABLE bpl2.user_achievements;
DROP TABLE bpl2.achievements;

ALTER TABLE bpl2.user_achievements_old RENAME TO achievements;
ALTER INDEX bpl2.user_achievements_old_pkey RENAME TO achievements_pkey;
