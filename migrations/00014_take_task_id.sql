-- +goose Up
-- +goose StatementBegin

-- Links a take to its generation log entry so we can retrieve the
-- request_payload (prompt + assets) without duplicating the JSON.
ALTER TABLE takes
    ADD COLUMN task_id VARCHAR(250) DEFAULT NULL;

CREATE INDEX idx_takes_task_id ON takes(task_id);

-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_takes_task_id;
ALTER TABLE DROP COLUMN IF EXISTS task_id;
