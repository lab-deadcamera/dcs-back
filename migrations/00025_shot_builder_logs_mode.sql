-- +goose Up
-- +goose StatementBegin

-- Distinguish generate-shots logs from refine-shots logs in the shared
-- shot_builder_logs table. Existing rows default to 'generate'.
ALTER TABLE shot_builder_logs ADD COLUMN IF NOT EXISTS mode VARCHAR(50) NOT NULL DEFAULT 'generate';
CREATE INDEX IF NOT EXISTS idx_shot_builder_logs_mode ON shot_builder_logs(mode);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_shot_builder_logs_mode;
ALTER TABLE shot_builder_logs DROP COLUMN IF EXISTS mode;

-- +goose StatementEnd
