-- +goose Up
ALTER TABLE projects ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;
CREATE INDEX idx_projects_active ON projects(active);

ALTER TABLE scenes ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;
CREATE INDEX idx_scenes_active ON scenes(active);

ALTER TABLE takes ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;
CREATE INDEX idx_takes_active ON takes(active);

-- +goose Down
DROP INDEX IF EXISTS idx_projects_active;
ALTER TABLE projects DROP COLUMN IF EXISTS active;

DROP INDEX IF EXISTS idx_scenes_active;
ALTER TABLE scenes DROP COLUMN IF EXISTS active;

DROP INDEX IF EXISTS idx_takes_active;
ALTER TABLE takes DROP COLUMN IF EXISTS active;
