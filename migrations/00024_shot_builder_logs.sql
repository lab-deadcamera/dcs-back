-- +goose Up
-- +goose StatementBegin

-- ─── Shot Builder Logs ─────────────────────────────────────────
-- One row per FAILED generate-shots call. Stores everything needed
-- to reconstruct the request: raw payload (incl. scene_context with
-- assigned resources), final composed prompts, user, model, skill,
-- tokens and duration.
CREATE TABLE shot_builder_logs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             INT,
    user_name           VARCHAR(250),
    user_email          VARCHAR(250),
    project_id          VARCHAR(250),
    scene_id            VARCHAR(250),
    key_model           VARCHAR(250),
    api_model           VARCHAR(250),
    skill_id            VARCHAR(250),
    skill_name          VARCHAR(250),
    request_payload     TEXT,
    system_prompt       TEXT,
    prompt              TEXT,
    status              VARCHAR(50) NOT NULL DEFAULT 'failed',
    error_message       TEXT,
    response            TEXT,
    attempts            INT DEFAULT 0,
    total_input_tokens  INT DEFAULT 0,
    total_output_tokens INT DEFAULT 0,
    duration_ms         BIGINT DEFAULT 0,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_shot_builder_logs_created_at ON shot_builder_logs(created_at DESC);
CREATE INDEX idx_shot_builder_logs_user_id ON shot_builder_logs(user_id);
CREATE INDEX idx_shot_builder_logs_project ON shot_builder_logs(project_id);
CREATE INDEX idx_shot_builder_logs_deleted_at ON shot_builder_logs(deleted_at);

-- ─── Shot Builder Attempts ──────────────────────────────────────
-- One row per Claude API call within a failed generate-shots call.
CREATE TABLE shot_builder_attempts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    log_id                UUID NOT NULL REFERENCES shot_builder_logs(id) ON DELETE CASCADE,
    attempt_number        INT NOT NULL,
    prompt                TEXT,
    response              TEXT,
    valid                 BOOLEAN,
    error_message         TEXT,
    input_tokens          INT DEFAULT 0,
    output_tokens         INT DEFAULT 0,
    cache_read_tokens     INT DEFAULT 0,
    cache_creation_tokens INT DEFAULT 0,
    duration_ms           BIGINT DEFAULT 0,
    created_at            TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_shot_builder_attempts_log_id ON shot_builder_attempts(log_id);

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS shot_builder_attempts;
DROP TABLE IF EXISTS shot_builder_logs;
