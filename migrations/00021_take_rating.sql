-- +goose Up
-- +goose StatementBegin

ALTER TABLE takes
    ADD COLUMN rating INT NOT NULL DEFAULT 0 CHECK (rating >= 0 AND rating <= 5);

-- +goose StatementEnd

-- +goose Down
ALTER TABLE takes DROP COLUMN IF EXISTS rating;
