-- +goose Up
-- +goose StatementBegin

-- The migration 00012 added chapters, but the original scenes UNIQUE constraint
-- on (project_id, number) was NOT removed. Now that scenes belong to chapters,
-- uniqueness must be per chapter.
--
-- Steps:
--  1. Drop the old UNIQUE(project_id, number) constraint.
--  2. Drop the index that backed it.
--  3. Create a new partial unique index on (chapter_id, number).

-- Step 1: drop the table-level UNIQUE constraint
ALTER TABLE scenes DROP CONSTRAINT IF EXISTS scenes_project_id_number_key;

-- Step 2: the constraint may also have created an index; clean up any
--         auto-named index that might linger
DROP INDEX IF EXISTS scenes_project_id_number_key;

-- Step 3: create the new unique index per chapter (ignore rows with NULL chapter_id)
DROP INDEX IF EXISTS idx_scenes_chapter_number;
CREATE UNIQUE INDEX idx_scenes_chapter_number ON scenes(chapter_id, number)
    WHERE deleted_at IS NULL AND chapter_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_scenes_chapter_number;
ALTER TABLE scenes ADD CONSTRAINT scenes_project_id_number_key UNIQUE (project_id, number);

-- +goose StatementEnd
