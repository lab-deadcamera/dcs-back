-- +goose Up
-- +goose StatementBegin

CREATE TABLE preset_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL
);

INSERT INTO preset_groups (name, slug, description) VALUES
    ('Lens', 'lens', 'Camera lens presets — focal length, depth of field, and optical character'),
    ('Camera Body', 'camera', 'Camera body presets — sensor, color science, and capture aesthetics'),
    ('Camera Motion', 'cameraMotion', 'Camera motion presets — movement, stabilization, and dynamic feel'),
    ('Color Grading', 'colorGrading', 'Color grading presets — look, palette, and mood'),
    ('Genre', 'genre', 'Genre presets — narrative tone, pacing, and visual conventions');

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS preset_groups CASCADE;
