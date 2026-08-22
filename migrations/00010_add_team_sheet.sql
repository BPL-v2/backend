-- +goose Up
CREATE TABLE team_sheet_entries (
    event_id                  INT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id                   INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    character_name            TEXT,
    role                      TEXT,
    secondary_role            TEXT,
    ascendancy                TEXT,
    main_skill                TEXT,
    build_notes               TEXT,
    pob_url                   TEXT,
    realm                     TEXT,
    uniques_needed            TEXT,
    altars                    TEXT,
    looking_for_group         BOOLEAN NOT NULL DEFAULT false,
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)
);

-- +goose Down
DROP TABLE team_sheet_entries;
