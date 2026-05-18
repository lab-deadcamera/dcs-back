-- +goose Up
CREATE TABLE server_communications (
    id              UUID PRIMARY KEY,
    task_id         TEXT NOT NULL,
    model_name      TEXT NOT NULL,
    endpoint        TEXT NOT NULL,
    method          TEXT NOT NULL,
    request_body    TEXT,
    response_body   TEXT,
    status_code     INT DEFAULT 0,
    duration_ms     BIGINT DEFAULT 0,
    error_message   TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS server_communications;
