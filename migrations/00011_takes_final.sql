-- +goose Up
-- +goose StatementBegin

ALTER TABLE takes
    ADD COLUMN final BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN finalized_at TIMESTAMPTZ DEFAULT NULL;

CREATE INDEX idx_takes_final ON takes(scene_id) WHERE final = true;

-- +goose StatementEnd

-- +goose Down
ALTER TABLE takes
    DROP COLUMN IF EXISTS finalized_at,
    DROP COLUMN IF EXISTS final;
