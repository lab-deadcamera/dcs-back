-- +goose Up
-- +goose StatementBegin

-- Add slot column to scene_characters (nullable, e.g. "@image1", "@image2")
ALTER TABLE scene_characters ADD COLUMN IF NOT EXISTS slot VARCHAR(20) DEFAULT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_scene_characters_slot
    ON scene_characters(scene_id, slot) WHERE slot IS NOT NULL;

-- Add slot column to shot_characters (nullable, e.g. "@image1", "@image2")
ALTER TABLE shot_characters ADD COLUMN IF NOT EXISTS slot VARCHAR(20) DEFAULT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_shot_characters_slot
    ON shot_characters(shot_id, slot) WHERE slot IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE scene_characters DROP COLUMN IF EXISTS slot;
ALTER TABLE shot_characters DROP COLUMN IF EXISTS slot;

-- +goose StatementEnd
