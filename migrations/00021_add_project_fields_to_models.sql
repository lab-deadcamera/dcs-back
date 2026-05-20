-- +goose Up
ALTER TABLE models ADD COLUMN project_name TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN project_number TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE models DROP COLUMN project_name;
ALTER TABLE models DROP COLUMN project_number;
