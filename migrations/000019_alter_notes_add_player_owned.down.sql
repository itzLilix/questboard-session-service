-- migration.down.sql

DROP INDEX IF EXISTS idx_notes_owner_id;

ALTER TABLE notes
    DROP COLUMN owner_id;

ALTER TYPE content_visibility RENAME VALUE 'private' TO 'gm_only';