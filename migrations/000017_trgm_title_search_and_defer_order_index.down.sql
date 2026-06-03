ALTER TABLE campaign_sessions
  DROP CONSTRAINT campaign_sessions_campaign_id_order_index_key,
  ADD  CONSTRAINT campaign_sessions_campaign_id_order_index_key
       UNIQUE (campaign_id, order_index);

DROP INDEX IF EXISTS idx_game_systems_canon_name_trgm;
DROP INDEX IF EXISTS idx_campaigns_title_trgm;
DROP INDEX IF EXISTS idx_sessions_title_trgm;

-- pg_trgm left installed: other objects may rely on it, and re-creating an
-- extension is cheap if a later migration needs it again.
