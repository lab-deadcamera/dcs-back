-- +goose Up
-- Drop the existing unique constraint so we can have multiple takes with
-- the same number (one active, previous versions discarded).
ALTER TABLE takes DROP CONSTRAINT IF EXISTS takes_scene_id_number_key;

-- Partial unique index: only one active (non-deleted, active=true) take per
-- scene per number. Discarded/inactive takes can share the same number.
CREATE UNIQUE INDEX idx_takes_active_unique ON takes(scene_id, number)
    WHERE deleted_at IS NULL AND active = true;

-- +goose Down
DROP INDEX IF EXISTS idx_takes_active_unique;
-- Recreate the original constraint (approximate, ignores stale inactive rows).
ALTER TABLE takes ADD CONSTRAINT takes_scene_id_number_key UNIQUE (scene_id, number);
