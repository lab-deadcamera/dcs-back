-- +goose Up
-- +goose StatementBegin

ALTER TABLE files
    ADD COLUMN duration DOUBLE PRECISION NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
ALTER TABLE files
    DROP COLUMN IF EXISTS duration;
