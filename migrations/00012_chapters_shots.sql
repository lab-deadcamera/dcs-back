-- +goose Up
-- +goose StatementBegin

-- ─── Chapters ──────────────────────────────────────────────────
CREATE TABLE chapters (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number      INT NOT NULL,
    name        VARCHAR(250) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    UNIQUE(project_id, number)
);

CREATE INDEX idx_chapters_project ON chapters(project_id);
CREATE INDEX idx_chapters_active ON chapters(active);
CREATE INDEX idx_chapters_deleted_at ON chapters(deleted_at);

-- ─── Shots ─────────────────────────────────────────────────────
CREATE TABLE shots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id    UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    number      INT NOT NULL,
    name        VARCHAR(250) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    UNIQUE(scene_id, number)
);

CREATE INDEX idx_shots_scene ON shots(scene_id);
CREATE INDEX idx_shots_active ON shots(active);
CREATE INDEX idx_shots_deleted_at ON shots(deleted_at);

-- ─── Scene: add chapter_id FK ──────────────────────────────────
ALTER TABLE scenes ADD COLUMN chapter_id UUID REFERENCES chapters(id) ON DELETE SET NULL;
CREATE INDEX idx_scenes_chapter ON scenes(chapter_id);

-- ─── Takes: add shot_id FK ─────────────────────────────────────
ALTER TABLE takes ADD COLUMN shot_id UUID REFERENCES shots(id) ON DELETE CASCADE;
CREATE INDEX idx_takes_shot ON takes(shot_id);

-- Replace the partial unique index to use (shot_id, number) instead of (scene_id, number)
DROP INDEX IF EXISTS idx_takes_active_unique;
CREATE UNIQUE INDEX idx_takes_active_unique ON takes(shot_id, number)
    WHERE deleted_at IS NULL AND active = true AND shot_id IS NOT NULL;

-- ─── Data migration ────────────────────────────────────────────

-- 1. Create a default chapter for every project that has scenes
INSERT INTO chapters (id, project_id, number, name, description, active)
SELECT
    gen_random_uuid(),
    p.id,
    1,
    'Chapter 1',
    'Default chapter (auto-migrated)',
    TRUE
FROM projects p
WHERE EXISTS (
    SELECT 1 FROM scenes s
    WHERE s.project_id = p.id AND s.deleted_at IS NULL
)
AND p.deleted_at IS NULL
AND NOT EXISTS (
    SELECT 1 FROM chapters c
    WHERE c.project_id = p.id AND c.deleted_at IS NULL
);

-- 2. Assign existing scenes to their default chapter
UPDATE scenes s
SET chapter_id = c.id
FROM chapters c
WHERE c.project_id = s.project_id
AND s.deleted_at IS NULL
AND s.chapter_id IS NULL;

-- 3. Create a default shot for every scene that has takes
INSERT INTO shots (id, scene_id, number, name, description, active)
SELECT
    gen_random_uuid(),
    sc.id,
    1,
    'Shot 1',
    'Default shot (auto-migrated)',
    TRUE
FROM scenes sc
WHERE EXISTS (
    SELECT 1 FROM takes t
    WHERE t.scene_id = sc.id AND t.deleted_at IS NULL
)
AND sc.deleted_at IS NULL
AND NOT EXISTS (
    SELECT 1 FROM shots sh
    WHERE sh.scene_id = sc.id AND sh.deleted_at IS NULL
);

-- 4. Assign existing takes to their default shot
UPDATE takes t
SET shot_id = sh.id
FROM shots sh
WHERE sh.scene_id = t.scene_id
AND t.deleted_at IS NULL
AND t.shot_id IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_takes_active_unique;
DROP INDEX IF EXISTS idx_takes_shot;
ALTER TABLE takes DROP COLUMN IF EXISTS shot_id;
DROP INDEX IF EXISTS idx_scenes_chapter;
ALTER TABLE scenes DROP COLUMN IF EXISTS chapter_id;
DROP INDEX IF EXISTS idx_shots_deleted_at;
DROP INDEX IF EXISTS idx_shots_active;
DROP INDEX IF EXISTS idx_shots_scene;
DROP TABLE IF EXISTS shots CASCADE;
DROP INDEX IF EXISTS idx_chapters_deleted_at;
DROP INDEX IF EXISTS idx_chapters_active;
DROP INDEX IF EXISTS idx_chapters_project;
DROP TABLE IF EXISTS chapters CASCADE;

-- Restore original partial unique index on takes(scene_id, number)
CREATE UNIQUE INDEX idx_takes_active_unique ON takes(scene_id, number)
    WHERE deleted_at IS NULL AND active = true;

-- Restore original scenes unique constraint on (project_id, number)
DROP INDEX IF EXISTS idx_scenes_chapter_number;
ALTER TABLE scenes ADD CONSTRAINT scenes_project_id_number_key UNIQUE (project_id, number);

-- +goose StatementEnd
