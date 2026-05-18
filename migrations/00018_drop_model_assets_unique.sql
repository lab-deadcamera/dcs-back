-- +goose Up
DROP INDEX IF EXISTS idx_model_assets_model_file;
DROP INDEX IF EXISTS idx_model_assets_file;
DROP INDEX IF EXISTS idx_model_assets_model;
ALTER TABLE model_assets DROP CONSTRAINT IF EXISTS model_assets_model_id_file_id_key;
CREATE INDEX idx_model_assets_model_file ON model_assets (model_id, file_id);
CREATE INDEX idx_model_assets_model ON model_assets (model_id);

-- +goose Down
DROP INDEX IF EXISTS idx_model_assets_model_file;
DROP INDEX IF EXISTS idx_model_assets_model;
ALTER TABLE model_assets ADD UNIQUE(model_id, file_id);
CREATE INDEX idx_model_assets_file ON model_assets (file_id);
