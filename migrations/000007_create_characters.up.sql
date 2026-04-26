CREATE TABLE characters (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID         NOT NULL,
    session_id  UUID         REFERENCES sessions (id) ON DELETE SET NULL,
    campaign_id UUID         REFERENCES campaigns (id) ON DELETE SET NULL,
    name        VARCHAR(100) NOT NULL,
    class       VARCHAR(100),
    level       SMALLINT     DEFAULT 1 CHECK (level >= 1),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_character_context CHECK (
        session_id IS NOT NULL OR campaign_id IS NOT NULL
    )
);

CREATE INDEX idx_characters_player_id   ON characters (player_id);
CREATE INDEX idx_characters_session_id  ON characters (session_id);
CREATE INDEX idx_characters_campaign_id ON characters (campaign_id);
