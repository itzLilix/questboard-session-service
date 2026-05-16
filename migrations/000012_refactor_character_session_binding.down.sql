-- Best-effort reversal: restore the scope columns on characters and rehydrate
-- session_id from session_players selections. campaign_id cannot be recovered
-- (the association was thrown away when we moved to usage-based listing) so
-- it comes back NULL.

ALTER TABLE characters
    ADD COLUMN session_id  UUID REFERENCES sessions  (id) ON DELETE SET NULL,
    ADD COLUMN campaign_id UUID REFERENCES campaigns (id) ON DELETE SET NULL;

CREATE INDEX idx_characters_session_id  ON characters (session_id);
CREATE INDEX idx_characters_campaign_id ON characters (campaign_id);

UPDATE characters c
SET    session_id = sp.session_id
FROM   session_players sp
WHERE  sp.character_id = c.id;

ALTER TABLE characters ADD CONSTRAINT chk_character_context CHECK (
    session_id IS NOT NULL OR campaign_id IS NOT NULL
);

DROP INDEX IF EXISTS idx_session_players_character_id;
ALTER TABLE session_players DROP COLUMN character_id;
