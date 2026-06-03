-- Typo-tolerant search via trigrams.
-- gin_trgm_ops accelerates both ILIKE substring matches and the
-- similarity / word_similarity (<%) operators used by the repositories.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_sessions_title_trgm          ON sessions     USING gin (title gin_trgm_ops);
CREATE INDEX idx_campaigns_title_trgm         ON campaigns    USING gin (title gin_trgm_ops);
CREATE INDEX idx_game_systems_canon_name_trgm ON game_systems USING gin (canonical_name gin_trgm_ops);

-- Make (campaign_id, order_index) uniqueness deferrable so a reorder can
-- shuffle multiple rows within one transaction without tripping the
-- constraint on an intermediate state.
ALTER TABLE campaign_sessions
  DROP CONSTRAINT campaign_sessions_campaign_id_order_index_key,
  ADD  CONSTRAINT campaign_sessions_campaign_id_order_index_key
       UNIQUE (campaign_id, order_index) DEFERRABLE INITIALLY IMMEDIATE;
