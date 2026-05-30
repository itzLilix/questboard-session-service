-- Generalize session_files into a unified files table that can live at either
-- session scope or campaign scope, mirroring the one-of pattern used by notes.

ALTER TABLE session_files RENAME TO files;
ALTER INDEX idx_session_files_session_id RENAME TO idx_files_session_id;

ALTER TABLE files ALTER COLUMN session_id DROP NOT NULL;

ALTER TABLE files
    ADD COLUMN campaign_id UUID REFERENCES campaigns (id) ON DELETE CASCADE;

ALTER TABLE files
    ADD COLUMN visibility content_visibility NOT NULL DEFAULT 'public';

ALTER TABLE files
    ADD CONSTRAINT chk_files_scope CHECK (
        (campaign_id IS NOT NULL)::INT + (session_id IS NOT NULL)::INT = 1
    );

DROP INDEX idx_files_session_id;
CREATE INDEX idx_files_session_id  ON files (session_id)  WHERE session_id  IS NOT NULL;
CREATE INDEX idx_files_campaign_id ON files (campaign_id) WHERE campaign_id IS NOT NULL;
