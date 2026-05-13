-- +goose Up
CREATE TABLE providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_providers_active ON providers (active);
CREATE INDEX idx_providers_deleted_at ON providers (deleted_at);

CREATE TABLE models (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    api_key     TEXT NOT NULL,
    url         TEXT NOT NULL,
    endpoint    VARCHAR(127) NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_models_provider ON models (provider_id);
CREATE INDEX idx_models_active ON models (active);
CREATE INDEX idx_models_deleted_at ON models (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS providers;
