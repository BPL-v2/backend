-- +goose Up
-- Turn the build_enabling boolean flag into a 1-5 importance scale
-- (1 = nice to have, 5 = absolutely essential). Existing data maps
-- false -> 1 and true -> 5.
ALTER TABLE item_wishes ALTER COLUMN build_enabling DROP DEFAULT;
ALTER TABLE item_wishes
    ALTER COLUMN build_enabling TYPE integer
    USING (CASE WHEN build_enabling THEN 5 ELSE 1 END);
ALTER TABLE item_wishes ALTER COLUMN build_enabling SET DEFAULT 1;
ALTER TABLE item_wishes
    ADD CONSTRAINT item_wishes_build_enabling_check
    CHECK (build_enabling BETWEEN 1 AND 5);

-- +goose Down
ALTER TABLE item_wishes DROP CONSTRAINT item_wishes_build_enabling_check;
ALTER TABLE item_wishes ALTER COLUMN build_enabling DROP DEFAULT;
ALTER TABLE item_wishes
    ALTER COLUMN build_enabling TYPE boolean
    USING (build_enabling >= 4);
ALTER TABLE item_wishes ALTER COLUMN build_enabling SET DEFAULT false;
