-- +goose Up
-- +goose StatementBegin

-- ─── Projects ─────────────────────────────────────────────────
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(250) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metadata    TEXT,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_projects_active ON projects(active);
CREATE INDEX idx_projects_deleted_at ON projects(deleted_at);

-- ─── Scenes ───────────────────────────────────────────────────
CREATE TABLE scenes (
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

CREATE INDEX idx_scenes_project ON scenes(project_id);
CREATE INDEX idx_scenes_active ON scenes(active);
CREATE INDEX idx_scenes_deleted_at ON scenes(deleted_at);

-- ─── Takes ────────────────────────────────────────────────────
CREATE TABLE takes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id        UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    number          INT NOT NULL CHECK (number >= 1 AND number <= 100),
    video_url       TEXT NOT NULL DEFAULT '',
    video_local_url TEXT NOT NULL DEFAULT '',
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_takes_scene ON takes(scene_id);
CREATE INDEX idx_takes_active ON takes(active);
CREATE INDEX idx_takes_deleted_at ON takes(deleted_at);

-- Partial unique index: only one active (non-deleted, active=true)
-- take per scene per number. Discarded takes can share the same number.
CREATE UNIQUE INDEX idx_takes_active_unique ON takes(scene_id, number)
    WHERE deleted_at IS NULL AND active = true;

-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_takes_active_unique;
DROP TABLE IF EXISTS takes CASCADE;
DROP TABLE IF EXISTS scenes CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
