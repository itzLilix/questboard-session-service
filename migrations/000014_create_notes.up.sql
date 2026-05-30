CREATE TYPE content_visibility AS ENUM ('public', 'gm_only');

CREATE TABLE notes (
    id          UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID               REFERENCES campaigns (id) ON DELETE CASCADE,
    session_id  UUID               REFERENCES sessions  (id) ON DELETE CASCADE,
    visibility  content_visibility NOT NULL DEFAULT 'gm_only',
    title       VARCHAR(255)       NOT NULL,
    body        TEXT               NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_notes_scope CHECK (
        (campaign_id IS NOT NULL)::INT + (session_id IS NOT NULL)::INT = 1
    )
);

CREATE INDEX idx_notes_campaign_id ON notes (campaign_id) WHERE campaign_id IS NOT NULL;
CREATE INDEX idx_notes_session_id  ON notes (session_id)  WHERE session_id  IS NOT NULL;

-- Migrate existing per-session master notes into the new table as gm_only notes.
INSERT INTO notes (session_id, visibility, title, body, created_at, updated_at)
SELECT id, 'public', 'Master notes', master_notes, created_at, updated_at
FROM   sessions
WHERE  master_notes IS NOT NULL AND master_notes <> '';

ALTER TABLE sessions DROP COLUMN master_notes;
