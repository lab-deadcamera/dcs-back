-- +goose Up
-- +goose StatementBegin

-- ─── Roles ────────────────────────────────────────────────────
CREATE TABLE roles (
    id    SERIAL PRIMARY KEY,
    name  VARCHAR(50) UNIQUE NOT NULL,
    level INT UNIQUE NOT NULL
);

INSERT INTO roles (name, level) VALUES
    ('SUPER_ADMIN', 0),
    ('ADMIN', 1),
    ('DIRECTOR', 2),
    ('USER', 3);

-- ─── Users ────────────────────────────────────────────────────
CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(255) NOT NULL,
    surname       VARCHAR(255) NOT NULL,
    active        BOOLEAN DEFAULT TRUE,
    role_id       INT NOT NULL DEFAULT 4 REFERENCES roles(id),
    user_name     VARCHAR(255) NOT NULL DEFAULT '',
    email         VARCHAR(255) NOT NULL DEFAULT '',
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_users_role_id ON users(role_id);

-- ─── Providers ────────────────────────────────────────────────
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

-- ─── Models ───────────────────────────────────────────────────
CREATE TABLE models (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id           UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name                  VARCHAR(255) NOT NULL,
    api_key               TEXT NOT NULL,
    url                   TEXT NOT NULL,
    endpoint              VARCHAR(127) NOT NULL,
    active                BOOLEAN NOT NULL DEFAULT TRUE,
    favorite              BOOLEAN NOT NULL DEFAULT FALSE,
    access_key_id         TEXT NOT NULL DEFAULT '',
    secret_access_key     TEXT NOT NULL DEFAULT '',
    default_asset_group_id TEXT NOT NULL DEFAULT '',
    model_type            TEXT NOT NULL DEFAULT 'video',
    project_name          TEXT NOT NULL DEFAULT '',
    project_number        TEXT NOT NULL DEFAULT '',
    created_at            TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at            TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_models_provider ON models (provider_id);
CREATE INDEX idx_models_active ON models (active);
CREATE INDEX idx_models_deleted_at ON models (deleted_at);
CREATE INDEX idx_models_favorite ON models (favorite) WHERE favorite = TRUE;

-- ─── Files ────────────────────────────────────────────────────
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

-- ─── Characters ───────────────────────────────────────────────
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

-- ─── Character-Files (M:N) ────────────────────────────────────
CREATE TABLE character_files (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id  UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    file_id       UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    role          VARCHAR(63) NOT NULL DEFAULT 'reference',
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(character_id, file_id, role)
);

CREATE INDEX idx_character_files_character ON character_files (character_id);
CREATE INDEX idx_character_files_file ON character_files (file_id);

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS character_files CASCADE;
DROP TABLE IF EXISTS characters CASCADE;
DROP TABLE IF EXISTS files CASCADE;
DROP TABLE IF EXISTS models CASCADE;
DROP TABLE IF EXISTS providers CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
