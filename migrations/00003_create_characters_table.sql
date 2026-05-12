-- +goose Up
CREATE TABLE characters (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_characters_name ON characters (name);
CREATE INDEX idx_characters_deleted_at ON characters (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS characters;
