CREATE TYPE campaign_status AS ENUM ('active', 'completed', 'cancelled', 'paused');

CREATE TABLE campaigns (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(255)    NOT NULL,
    description TEXT,
    master_id   UUID            NOT NULL,
    system_id   UUID            REFERENCES game_systems (id),
    status      campaign_status NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_campaigns_master_id ON campaigns (master_id);
CREATE INDEX idx_campaigns_status    ON campaigns (status);
