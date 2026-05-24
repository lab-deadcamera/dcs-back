-- +goose Up
-- +goose StatementBegin

ALTER TABLE generation_logs
    ADD COLUMN resource_type VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN content_types VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN estimated_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN cost_source VARCHAR(50) NOT NULL DEFAULT '';

CREATE INDEX idx_gen_logs_resource_type ON generation_logs(resource_type);

-- +goose StatementEnd

-- +goose Down
ALTER TABLE generation_logs
    DROP COLUMN IF EXISTS estimated_cost,
    DROP COLUMN IF EXISTS cost_source,
    DROP COLUMN IF EXISTS resource_type,
    DROP COLUMN IF EXISTS content_types;
