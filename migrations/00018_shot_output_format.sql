-- +goose Up
-- +goose StatementBegin

ALTER TABLE shots ADD COLUMN IF NOT EXISTS aspect_ratio VARCHAR(10) DEFAULT NULL;
ALTER TABLE shots ADD COLUMN IF NOT EXISTS duration_seconds INT DEFAULT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE shots DROP COLUMN IF EXISTS duration_seconds;
ALTER TABLE shots DROP COLUMN IF EXISTS aspect_ratio;

-- +goose StatementEnd
