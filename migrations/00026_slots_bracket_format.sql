-- +goose Up
-- +goose StatementBegin

-- Convert legacy "@imageN" slots (lowercase, @-prefixed) to the canonical
-- bracketed "[ImageN]" format across every assignment table. Tokens embedded
-- in saved pre-prompts and request payloads are converted with regexp_replace.
--
-- shot_assets.slot is intentionally NOT touched: it uses the 'free' /
-- 'first-frame' / 'last-frame' vocabulary, not @imageN slots.

UPDATE chapter_characters
   SET slot = '[Image' || (regexp_match(slot, '(\d+)'))[1] || ']'
 WHERE slot LIKE '@image%';

UPDATE chapter_assets
   SET slot = '[Image' || (regexp_match(slot, '(\d+)'))[1] || ']'
 WHERE slot LIKE '@image%';

UPDATE scene_characters
   SET slot = '[Image' || (regexp_match(slot, '(\d+)'))[1] || ']'
 WHERE slot LIKE '@image%';

UPDATE shot_characters
   SET slot = '[Image' || (regexp_match(slot, '(\d+)'))[1] || ']'
 WHERE slot LIKE '@image%';

-- Tokens "@imageN" inside saved pre-prompts and generation request payloads.
UPDATE shots
   SET description = regexp_replace(description, '@[Ii]mage(\d+)', '[Image\1]', 'g')
 WHERE description LIKE '%@image%';

UPDATE generation_logs
   SET request_payload = regexp_replace(request_payload, '@[Ii]mage(\d+)', '[Image\1]', 'g')
 WHERE request_payload LIKE '%@image%';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE chapter_characters
   SET slot = '@image' || (regexp_match(slot, '(\d+)'))[1]
 WHERE slot LIKE '[Image%';

UPDATE chapter_assets
   SET slot = '@image' || (regexp_match(slot, '(\d+)'))[1]
 WHERE slot LIKE '[Image%';

UPDATE scene_characters
   SET slot = '@image' || (regexp_match(slot, '(\d+)'))[1]
 WHERE slot LIKE '[Image%';

UPDATE shot_characters
   SET slot = '@image' || (regexp_match(slot, '(\d+)'))[1]
 WHERE slot LIKE '[Image%';

UPDATE shots
   SET description = regexp_replace(description, '\[Image(\d+)\]', '@image\1', 'g')
 WHERE description LIKE '%[Image%';

UPDATE generation_logs
   SET request_payload = regexp_replace(request_payload, '\[Image(\d+)\]', '@image\1', 'g')
 WHERE request_payload LIKE '%[Image%';

-- +goose StatementEnd
