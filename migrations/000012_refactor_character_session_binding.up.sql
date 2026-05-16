-- Move per-session character selection from characters.session_id to
-- session_players.character_id, so a player can have multiple characters in
-- one campaign and pick which one is in play for a given session.

ALTER TABLE session_players
    ADD COLUMN character_id UUID REFERENCES characters (id) ON DELETE SET NULL;

CREATE INDEX idx_session_players_character_id ON session_players (character_id);

-- Backfill: each existing session-scoped character becomes the selected
-- character on its matching session_players row.
UPDATE session_players sp
SET    character_id = c.id
FROM   characters c
WHERE  c.session_id = sp.session_id
  AND  c.player_id  = sp.player_id;

-- Retire all scope columns on characters. A character now belongs to a
-- player only; its appearance in a session or campaign is derived from
-- session_players.character_id and the campaign_sessions join.
ALTER TABLE characters DROP CONSTRAINT chk_character_context;
DROP INDEX IF EXISTS idx_characters_session_id;
DROP INDEX IF EXISTS idx_characters_campaign_id;
ALTER TABLE characters DROP COLUMN session_id;
ALTER TABLE characters DROP COLUMN campaign_id;
