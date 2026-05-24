-- +goose Up
-- +goose StatementBegin

CREATE TABLE presets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES preset_groups(id),
    code VARCHAR(100) NOT NULL,
    label VARCHAR(200) NOT NULL,
    prompt TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    UNIQUE(group_id, code)
);

CREATE INDEX idx_presets_group ON presets(group_id);

-- ─── Lens ────────────────────────────────────────────────────────
INSERT INTO presets (group_id, code, label, prompt) VALUES
    ((SELECT id FROM preset_groups WHERE slug = 'lens'), 'wide_24mm', '24mm Wide',
     'shot on a 24mm wide lens, expansive framing with subtle edge distortion, deep depth of field'),
    ((SELECT id FROM preset_groups WHERE slug = 'lens'), 'classic_35mm', '35mm Classic',
     'shot on a 35mm lens, natural human perspective, balanced framing, classic cinema feel'),
    ((SELECT id FROM preset_groups WHERE slug = 'lens'), 'portrait_50mm', '50mm Portrait',
     'shot on a 50mm lens, intimate perspective, clean subject isolation, natural compression'),
    ((SELECT id FROM preset_groups WHERE slug = 'lens'), 'tele_85mm', '85mm Tele',
     'shot on an 85mm lens, creamy shallow depth of field, compressed background, cinematic bokeh');

-- ─── Camera Body ─────────────────────────────────────────────────
INSERT INTO presets (group_id, code, label, prompt) VALUES
    ((SELECT id FROM preset_groups WHERE slug = 'camera'), 'arri_alexa', 'Arri Alexa 65',
     'captured on Arri Alexa 65, rich dynamic range, organic highlight rolloff, filmic skin tones'),
    ((SELECT id FROM preset_groups WHERE slug = 'camera'), 'red_komodo', 'Red Komodo 6K',
     'captured on Red Komodo 6K, crisp detail, deep contrast, modern digital clarity'),
    ((SELECT id FROM preset_groups WHERE slug = 'camera'), 'sony_venice', 'Sony Venice',
     'captured on Sony Venice, cinematic wide gamut, smooth highlight handling, premium color science'),
    ((SELECT id FROM preset_groups WHERE slug = 'camera'), 'film_16mm', '16mm Film',
     'shot on 16mm celluloid film, visible grain structure, slight halation, analog imperfection and warmth');

-- ─── Camera Motion ───────────────────────────────────────────────
INSERT INTO presets (group_id, code, label, prompt) VALUES
    ((SELECT id FROM preset_groups WHERE slug = 'cameraMotion'), 'static_lockoff', 'Static / Locked Off',
     'static locked-off camera, no camera movement, tripod-mounted, steady composition'),
    ((SELECT id FROM preset_groups WHERE slug = 'cameraMotion'), 'slow_dolly_in', 'Slow Dolly In',
     'slow dolly push into the subject, smooth and deliberate, building intimacy and focus'),
    ((SELECT id FROM preset_groups WHERE slug = 'cameraMotion'), 'orbit', 'Orbit',
     'orbiting camera movement around the subject, dynamic and immersive, encircling motion'),
    ((SELECT id FROM preset_groups WHERE slug = 'cameraMotion'), 'handheld', 'Handheld',
     'handheld camera, organic breathing motion, intimate and immediate, documentary feel');

-- ─── Color Grading ───────────────────────────────────────────────
INSERT INTO presets (group_id, code, label, prompt) VALUES
    ((SELECT id FROM preset_groups WHERE slug = 'colorGrading'), 'tokio', 'Tokio',
     'neon-drenched cyberpunk palette, magenta and cyan highlights, deep indigo shadows, cinematic urban night'),
    ((SELECT id FROM preset_groups WHERE slug = 'colorGrading'), 'colombia', 'Colombia',
     'warm vibrant tropical palette, amber and emerald tones, golden hour warmth, rich saturated colors'),
    ((SELECT id FROM preset_groups WHERE slug = 'colorGrading'), 'ohio', 'Ohio',
     'desaturated muted midwestern palette, faded teal and clay, overcast soft light, melancholic atmosphere'),
    ((SELECT id FROM preset_groups WHERE slug = 'colorGrading'), 'bank', 'Bank',
     'sterile corporate palette, cool whites and steel blues, clinical precision, minimalist institutional feel');

-- ─── Genre ───────────────────────────────────────────────────────
INSERT INTO presets (group_id, code, label, prompt) VALUES
    ((SELECT id FROM preset_groups WHERE slug = 'genre'), 'drama', 'Drama',
     'dramatic narrative pacing, emotional character focus, naturalistic performances, restrained camera'),
    ((SELECT id FROM preset_groups WHERE slug = 'genre'), 'action', 'Action',
     'high-energy action sequencing, rapid cuts, dynamic camera movement, intense stunts and choreography'),
    ((SELECT id FROM preset_groups WHERE slug = 'genre'), 'noir', 'Noir',
     'film noir aesthetic, high-contrast chiaroscuro lighting, shadowy compositions, moral ambiguity'),
    ((SELECT id FROM preset_groups WHERE slug = 'genre'), 'horror', 'Horror',
     'suspenseful horror atmosphere, creeping dread, unsettling sound design, tension-building pacing');

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS presets CASCADE;
