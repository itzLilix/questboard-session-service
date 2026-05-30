ALTER TABLE sessions ADD COLUMN master_notes TEXT;

-- Best-effort restore: pick the gm_only 'Master notes' entry per session, if any.
UPDATE sessions s
SET    master_notes = n.body
FROM   notes n
WHERE  n.session_id = s.id
  AND  n.visibility = 'public'
  AND  n.title      = 'Master notes';

DROP TABLE IF EXISTS notes;

DROP TYPE IF EXISTS content_visibility;
