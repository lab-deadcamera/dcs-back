-- +goose Up
ALTER TABLE model_assets ADD COLUMN asset_url TEXT NOT NULL DEFAULT '';
ALTER TABLE model_assets ADD COLUMN asset_type TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE model_assets DROP COLUMN IF EXISTS asset_type;
ALTER TABLE model_assets DROP COLUMN IF EXISTS asset_url;
