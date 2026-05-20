-- +goose Up
ALTER TABLE models ADD COLUMN model_type TEXT NOT NULL DEFAULT 'video';

-- +goose Down
ALTER TABLE models DROP COLUMN IF EXISTS model_type;
