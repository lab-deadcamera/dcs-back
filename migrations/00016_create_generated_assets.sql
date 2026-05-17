-- +goose Up
CREATE TABLE generated_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id VARCHAR(250) NOT NULL,
    model_name VARCHAR(250) NOT NULL DEFAULT '',
    user_id INT,
    project_id VARCHAR(250),
    scene_id VARCHAR(250),
    scene_code VARCHAR(50),
    take_number INT DEFAULT 0,
    original_url TEXT NOT NULL,
    local_path TEXT,
    filename VARCHAR(500),
    mime_type VARCHAR(100),
    file_size BIGINT DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    confirmed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_gen_assets_task ON generated_assets(task_id);
CREATE INDEX idx_gen_assets_project ON generated_assets(project_id);
CREATE INDEX idx_gen_assets_scene ON generated_assets(scene_id);
CREATE INDEX idx_gen_assets_status ON generated_assets(status);

-- +goose Down
DROP TABLE IF EXISTS generated_assets;
