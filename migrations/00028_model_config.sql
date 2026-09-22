-- +goose Up
-- +goose StatementBegin

-- Per-model configuration (JSONB). Video models use it to declare their
-- quantity limits — minimum and maximum number of videos per generation —
-- e.g. {"min_videos": 1, "max_videos": 4}. The backend clamps the request's
-- quantity to this range in GenerateUnified.
ALTER TABLE models
    ADD COLUMN config JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose StatementEnd

-- +goose Down
ALTER TABLE models DROP COLUMN IF EXISTS config;
