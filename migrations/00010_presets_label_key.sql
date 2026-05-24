-- +goose Up
-- +goose StatementBegin

ALTER TABLE presets
    ADD COLUMN label_key VARCHAR(200) NOT NULL DEFAULT '';

UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.LENSES.24MM_WIDE' WHERE code = 'wide_24mm';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.LENSES.35MM_CLASSIC' WHERE code = 'classic_35mm';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.LENSES.50MM_PORTRAIT' WHERE code = 'portrait_50mm';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.LENSES.85MM_TELE' WHERE code = 'tele_85mm';

UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.BODIES.ARRI_ALEXA_65' WHERE code = 'arri_alexa';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.BODIES.RED_KOMODO_6K' WHERE code = 'red_komodo';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.BODIES.SONY_VENICE' WHERE code = 'sony_venice';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.BODIES.FILM_16MM' WHERE code = 'film_16mm';

UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.MOTIONS.STATIC' WHERE code = 'static_lockoff';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.MOTIONS.DOLLY_IN' WHERE code = 'slow_dolly_in';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.MOTIONS.ORBIT' WHERE code = 'orbit';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.MOTIONS.HANDHELD' WHERE code = 'handheld';

UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.GRADES.TOKIO' WHERE code = 'tokio';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.GRADES.COLOMBIA' WHERE code = 'colombia';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.GRADES.OHIO' WHERE code = 'ohio';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.GRADES.BANK' WHERE code = 'bank';

UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.GENRES.DRAMA' WHERE code = 'drama';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.GENRES.ACTION' WHERE code = 'action';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.GENRES.NOIR' WHERE code = 'noir';
UPDATE presets SET label_key = 'STUDIO.CINEMATOGRAPHY.GENRES.HORROR' WHERE code = 'horror';

-- +goose StatementEnd

-- +goose Down
ALTER TABLE presets DROP COLUMN IF EXISTS label_key;
