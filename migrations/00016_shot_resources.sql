-- +goose Up
-- +goose StatementBegin

-- ─── Shot resource assignments (characters) ─────────────────────────
CREATE TABLE shot_characters (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shot_id      UUID NOT NULL REFERENCES shots(id) ON DELETE CASCADE,
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shot_id, character_id)
);
CREATE INDEX idx_shot_characters_shot ON shot_characters(shot_id);

-- ─── Shot resource assignments (assets) ─────────────────────────────
CREATE TABLE shot_assets (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shot_id  UUID NOT NULL REFERENCES shots(id) ON DELETE CASCADE,
    file_id  UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    slot     VARCHAR(20) NOT NULL DEFAULT 'free',
      -- 'free' | 'first-frame' | 'last-frame'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shot_id, file_id)
);
CREATE INDEX idx_shot_assets_shot ON shot_assets(shot_id);

-- ─── Shot resource assignments (presets) ────────────────────────────
CREATE TABLE shot_presets (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shot_id  UUID NOT NULL REFERENCES shots(id) ON DELETE CASCADE,
    preset_id UUID NOT NULL REFERENCES presets(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shot_id, preset_id)
);
CREATE INDEX idx_shot_presets_shot ON shot_presets(shot_id);

-- ─── Shot model override ────────────────────────────────────────────
ALTER TABLE shots ADD COLUMN IF NOT EXISTS model_id VARCHAR(250) DEFAULT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS shot_presets CASCADE;
DROP TABLE IF EXISTS shot_assets CASCADE;
DROP TABLE IF EXISTS shot_characters CASCADE;
ALTER TABLE shots DROP COLUMN IF EXISTS model_id;

-- +goose StatementEnd
