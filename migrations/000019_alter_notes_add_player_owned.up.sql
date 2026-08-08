-- 'gm_only' is misleading now that any player can write private notes
ALTER TYPE content_visibility RENAME VALUE 'gm_only' TO 'private';
-- (the column's DEFAULT 'gm_only' is stored by enum oid, not by label,
--  so it now reads as DEFAULT 'private' automatically — no separate
--  ALTER COLUMN needed)

-- Every note now has an owner, not just an implicit "the GM wrote this"
ALTER TABLE notes
    ADD COLUMN owner_id UUID;

-- Backfill: every note in the table right now came from the old
-- per-session master_notes migration, so its owner is that session's master
UPDATE notes n
SET    owner_id = s.master_id
FROM   sessions s
WHERE  n.session_id = s.id
  AND  s.master_id IS NOT NULL;

CREATE INDEX idx_notes_owner_id ON notes (owner_id) WHERE owner_id IS NOT NULL;