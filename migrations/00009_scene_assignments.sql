-- +goose Up
-- +goose StatementBegin

CREATE TABLE scene_presets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id UUID NOT NULL REFERENCES scenes(id),
    preset_id UUID NOT NULL REFERENCES presets(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(scene_id, preset_id)
);

CREATE INDEX idx_scene_presets_scene ON scene_presets(scene_id);

CREATE TABLE scene_characters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id UUID NOT NULL REFERENCES scenes(id),
    character_id UUID NOT NULL REFERENCES characters(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(scene_id, character_id)
);

CREATE INDEX idx_scene_characters_scene ON scene_characters(scene_id);

CREATE TABLE scene_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id UUID NOT NULL REFERENCES scenes(id),
    file_id UUID NOT NULL REFERENCES files(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(scene_id, file_id)
);

CREATE INDEX idx_scene_assets_scene ON scene_assets(scene_id);

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS scene_assets CASCADE;
DROP TABLE IF EXISTS scene_characters CASCADE;
DROP TABLE IF EXISTS scene_presets CASCADE;
