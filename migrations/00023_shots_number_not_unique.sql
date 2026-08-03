-- +goose Up
-- +goose StatementBegin

-- Allow duplicate shot numbers within a scene so the shot builder can create
-- a new shot from an existing one ("clone a shot" keeps the same number).
-- Takes keep their own UNIQUE(shot_id, number) — untouched.
ALTER TABLE shots DROP CONSTRAINT IF EXISTS shots_scene_id_number_key;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin

-- Re-add the unique constraint. This fails if duplicates already exist —
-- deduplicate before reverting.
ALTER TABLE shots ADD CONSTRAINT shots_scene_id_number_key UNIQUE (scene_id, number);

-- +goose StatementEnd
