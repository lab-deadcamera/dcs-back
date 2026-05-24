-- +goose Up
-- +goose StatementBegin

-- ─── Generation Logs ──────────────────────────────────────────
CREATE TABLE generation_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         VARCHAR(250) NOT NULL,
    model_name      VARCHAR(250) NOT NULL,
    request_payload TEXT,
    outputs         TEXT,
    status          VARCHAR(100) NOT NULL DEFAULT 'running',
    error_message   VARCHAR(800),
    user_id         INT,
    project_id      VARCHAR(250),
    scene_id        VARCHAR(250),
    scene_code      VARCHAR(50),
    take_number     INT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_generation_logs_task_id ON generation_logs(task_id);
CREATE INDEX idx_gen_logs_user_id ON generation_logs(user_id);
CREATE INDEX idx_gen_logs_project ON generation_logs(project_id);
CREATE INDEX idx_gen_logs_scene ON generation_logs(scene_id);
CREATE INDEX idx_generation_logs_created_at ON generation_logs(created_at DESC);
CREATE INDEX idx_generation_logs_deleted_at ON generation_logs(deleted_at);

-- ─── Generated Assets ─────────────────────────────────────────
CREATE TABLE generated_assets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       VARCHAR(250) NOT NULL,
    model_name    VARCHAR(250) NOT NULL DEFAULT '',
    user_id       INT,
    project_id    VARCHAR(250),
    scene_id      VARCHAR(250),
    scene_code    VARCHAR(50),
    take_number   INT DEFAULT 0,
    original_url  TEXT NOT NULL,
    local_path    TEXT,
    filename      VARCHAR(500),
    mime_type     VARCHAR(100),
    file_size     BIGINT DEFAULT 0,
    status        VARCHAR(50) NOT NULL DEFAULT 'pending',
    confirmed_at  TIMESTAMP WITH TIME ZONE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_gen_assets_task ON generated_assets(task_id);
CREATE INDEX idx_gen_assets_project ON generated_assets(project_id);
CREATE INDEX idx_gen_assets_scene ON generated_assets(scene_id);
CREATE INDEX idx_gen_assets_status ON generated_assets(status);

-- ─── Server Communications ────────────────────────────────────
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

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS server_communications;
DROP TABLE IF EXISTS generated_assets;
DROP TABLE IF EXISTS generation_logs;
