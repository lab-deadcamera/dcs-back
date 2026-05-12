-- +goose Up
CREATE TABLE files (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename    VARCHAR(255) NOT NULL,
    path        TEXT NOT NULL,
    size        BIGINT NOT NULL,
    mime_type   VARCHAR(127) NOT NULL,
    category    VARCHAR(31) NOT NULL,
    format      VARCHAR(15) NOT NULL,
    storage     VARCHAR(15) NOT NULL DEFAULT 'persistent',
    trashed     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_files_category ON files (category);
CREATE INDEX idx_files_storage ON files (storage);
CREATE INDEX idx_files_deleted_at ON files (deleted_at);
CREATE INDEX idx_files_trashed ON files (trashed);

-- +goose Down
DROP TABLE IF EXISTS files;
