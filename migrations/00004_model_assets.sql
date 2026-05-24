-- +goose Up
-- +goose StatementBegin

-- ─── Model Assets (sync entre archivos y galería BytePlus) ────
CREATE TABLE model_assets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id          UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    file_id           UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    asset_id          TEXT NOT NULL,
    asset_group_id    TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'active',
    error_message     TEXT NOT NULL DEFAULT '',
    asset_url         TEXT NOT NULL DEFAULT '',
    asset_type        TEXT NOT NULL DEFAULT '',
    reference_uri     TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_model_assets_model_file ON model_assets (model_id, file_id);
CREATE INDEX idx_model_assets_model ON model_assets (model_id);

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS model_assets CASCADE;
