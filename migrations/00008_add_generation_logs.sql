-- +goose Up
CREATE TABLE generation_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         VARCHAR(250) NOT NULL,
    model_name      VARCHAR(250) NOT NULL,
    request_payload TEXT,
    ai_response     TEXT,
    ai_call_payload TEXT,
    outputs         TEXT,
    status          VARCHAR(100) NOT NULL DEFAULT 'running',
    error_message   VARCHAR(800),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_generation_logs_task_id ON generation_logs(task_id);
CREATE INDEX idx_generation_logs_deleted_at ON generation_logs(deleted_at);
CREATE INDEX idx_generation_logs_created_at ON generation_logs(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS generation_logs;
