-- +goose Up
CREATE TABLE character_files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    file_id      UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    role         VARCHAR(63) NOT NULL DEFAULT 'reference',
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(character_id, file_id, role)
);

CREATE INDEX idx_character_files_character ON character_files (character_id);
CREATE INDEX idx_character_files_file ON character_files (file_id);

-- +goose Down
DROP TABLE IF EXISTS character_files;
