-- +goose Up
CREATE TABLE character_delve_history (
    id           SERIAL PRIMARY KEY,
    character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    delve        INT8 NOT NULL,
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX character_delve_history_character_id_idx ON character_delve_history (character_id, recorded_at);

-- +goose Down
DROP TABLE character_delve_history;
