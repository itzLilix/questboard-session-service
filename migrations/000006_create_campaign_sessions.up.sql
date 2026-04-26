CREATE TABLE campaign_sessions (
    campaign_id         UUID         NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    session_id          UUID         NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    order_index         SMALLINT     NOT NULL,
    cached_title        VARCHAR(255),
    cached_scheduled_at TIMESTAMPTZ,
    brief_description   TEXT,
    PRIMARY KEY (campaign_id, session_id),
    UNIQUE (campaign_id, order_index)
);

CREATE INDEX idx_campaign_sessions_order ON campaign_sessions (campaign_id, order_index);

CREATE OR REPLACE FUNCTION sync_session_type() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE sessions SET type = 'campaign' WHERE id = NEW.session_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE sessions SET type = 'oneshot' WHERE id = OLD.session_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_session_type
AFTER INSERT OR DELETE ON campaign_sessions
FOR EACH ROW EXECUTE FUNCTION sync_session_type();