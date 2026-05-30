-- Campaigns get their own visibility, independent from sessions.
-- Reuses session_availability ENUM ('open' | 'application' | 'private').
ALTER TABLE campaigns
    ADD COLUMN availability session_availability NOT NULL DEFAULT 'open';
