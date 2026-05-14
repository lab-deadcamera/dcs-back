-- +goose Up
ALTER TABLE models ADD COLUMN favorite BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_models_favorite ON models (favorite) WHERE favorite = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_models_favorite;
ALTER TABLE models DROP COLUMN IF EXISTS favorite;
