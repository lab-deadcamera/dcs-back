-- +goose Up
-- +goose StatementBegin

ALTER TABLE generation_logs
    ADD COLUMN shot_id VARCHAR(250) DEFAULT NULL;

CREATE INDEX idx_gen_logs_shot_id ON generation_logs(shot_id);

-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_gen_logs_shot_id;
ALTER TABLE generation_logs DROP COLUMN IF EXISTS shot_id;
