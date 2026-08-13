-- +goose Up
-- +goose StatementBegin

-- ─── Chapter-level character assignments ─────────────────────────
CREATE TABLE chapter_characters (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id   UUID NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    slot         VARCHAR(20) DEFAULT NULL,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chapter_id, character_id)
);
CREATE INDEX idx_chapter_characters_chapter ON chapter_characters(chapter_id);
CREATE UNIQUE INDEX idx_chapter_characters_slot
    ON chapter_characters(chapter_id, slot) WHERE slot IS NOT NULL;

-- ─── Chapter-level asset assignments ─────────────────────────────
CREATE TABLE chapter_assets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id UUID NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    file_id    UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    slot       VARCHAR(20) DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chapter_id, file_id)
);
CREATE INDEX idx_chapter_assets_chapter ON chapter_assets(chapter_id);

-- ─── Chapter-level preset assignments ────────────────────────────
CREATE TABLE chapter_presets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id UUID NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    preset_id  UUID NOT NULL REFERENCES presets(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chapter_id, preset_id)
);
CREATE INDEX idx_chapter_presets_chapter ON chapter_presets(chapter_id);

-- ─── Add chapter_id to scenes number unique index ────────────────
-- Replace the existing unique index (project_id, number) with (chapter_id, number)
DROP INDEX IF EXISTS idx_scenes_chapter_number;
CREATE UNIQUE INDEX idx_scenes_chapter_number ON scenes(chapter_id, number)
    WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_scenes_chapter_number;
CREATE UNIQUE INDEX idx_scenes_chapter_number ON scenes(project_id, number)
    WHERE deleted_at IS NULL;

DROP TABLE IF EXISTS chapter_presets CASCADE;
DROP TABLE IF EXISTS chapter_assets CASCADE;
DROP TABLE IF EXISTS chapter_characters CASCADE;

-- +goose StatementEnd
