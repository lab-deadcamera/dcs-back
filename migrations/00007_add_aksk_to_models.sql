-- +goose Up
ALTER TABLE models ADD COLUMN access_key_id TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN secret_access_key TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN default_asset_group_id TEXT NOT NULL DEFAULT '';

CREATE TABLE model_assets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id        UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    file_id         UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    asset_id        TEXT NOT NULL,
    asset_group_id  TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active',
    error_message   TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(model_id, file_id)
);

CREATE INDEX idx_model_assets_model ON model_assets (model_id);
CREATE INDEX idx_model_assets_file ON model_assets (file_id);

-- +goose Down
DROP TABLE IF EXISTS model_assets;
ALTER TABLE models DROP COLUMN IF EXISTS default_asset_group_id;
ALTER TABLE models DROP COLUMN IF EXISTS secret_access_key;
ALTER TABLE models DROP COLUMN IF EXISTS access_key_id;
