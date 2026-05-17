-- +goose Up
ALTER TABLE generation_logs
  ADD COLUMN user_id INT,
  ADD COLUMN project_id VARCHAR(250),
  ADD COLUMN scene_id VARCHAR(250),
  ADD COLUMN scene_code VARCHAR(50);

CREATE INDEX idx_gen_logs_user_id ON generation_logs(user_id);
CREATE INDEX idx_gen_logs_project ON generation_logs(project_id);
CREATE INDEX idx_gen_logs_scene ON generation_logs(scene_id);

-- +goose Down
ALTER TABLE generation_logs
  DROP COLUMN IF EXISTS user_id,
  DROP COLUMN IF EXISTS project_id,
  DROP COLUMN IF EXISTS scene_id,
  DROP COLUMN IF EXISTS scene_code;
