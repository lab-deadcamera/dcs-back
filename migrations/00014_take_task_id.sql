-- +goose Up
-- +goose StatementBegin

-- Links a take to its generation log entry so we can retrieve the
-- request_payload (prompt + assets) without duplicating the JSON.
ALTER TABLE takes
    ADD COLUMN task_id VARCHAR(250) DEFAULT NULL;

CREATE INDEX idx_takes_task_id ON takes(task_id);

-- Backfill: populate task_id for existing takes by matching
-- generation_logs on scene_id + take_number. This ensures retroactive
-- payload retrieval for takes created before the column existed.
UPDATE takes t
SET task_id = gl.task_id
FROM generation_logs gl
WHERE t.task_id IS NULL
  AND gl.deleted_at IS NULL
  AND gl.scene_id::text = t.scene_id::text
  AND gl.take_number = t.number
  AND gl.task_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_takes_task_id;
ALTER TABLE takes DROP COLUMN IF EXISTS task_id;
