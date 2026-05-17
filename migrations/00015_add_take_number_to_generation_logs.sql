-- +goose Up
ALTER TABLE generation_logs
  ADD COLUMN take_number INT;

-- +goose Down
ALTER TABLE generation_logs
  DROP COLUMN IF EXISTS take_number;
