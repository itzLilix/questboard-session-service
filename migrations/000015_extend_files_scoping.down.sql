-- Campaign-scoped files have no place in the old schema; drop them on rollback.
DELETE FROM files WHERE campaign_id IS NOT NULL;

DROP INDEX IF EXISTS idx_files_campaign_id;
DROP INDEX IF EXISTS idx_files_session_id;

ALTER TABLE files DROP CONSTRAINT IF EXISTS chk_files_scope;
ALTER TABLE files DROP COLUMN IF EXISTS visibility;
ALTER TABLE files DROP COLUMN IF EXISTS campaign_id;
ALTER TABLE files ALTER COLUMN session_id SET NOT NULL;

ALTER TABLE files RENAME TO session_files;
CREATE INDEX idx_session_files_session_id ON session_files (session_id);
