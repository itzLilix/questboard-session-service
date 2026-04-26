CREATE TYPE session_format AS ENUM ('online', 'offline');
CREATE TYPE session_availability AS ENUM ('open', 'private', 'application');
CREATE TYPE session_status AS ENUM ('draft', 'published', 'ongoing', 'completed', 'cancelled');
CREATE TYPE session_type AS ENUM ('oneshot', 'campaign');

CREATE TABLE sessions (
    id            UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    title         VARCHAR(255)         NOT NULL,
    format        session_format       NOT NULL,
    scheduled_at  TIMESTAMPTZ          NOT NULL,
    location      TEXT,
    system_id     UUID                 NOT NULL REFERENCES game_systems (id),
    type          session_type         NOT NULL DEFAULT 'oneshot',
    availability  session_availability NOT NULL DEFAULT 'open',
    description   TEXT,
    preview_url   TEXT,
    max_seats     SMALLINT             NOT NULL DEFAULT 6 CHECK (max_seats > 0),
    master_id     UUID                 NOT NULL,
    price         NUMERIC(10, 2)       NOT NULL DEFAULT 0 CHECK (price >= 0),
    master_notes  TEXT,
    status        session_status       NOT NULL DEFAULT 'draft',
    free_seats    SMALLINT             NOT NULL DEFAULT 6,
    created_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_master_id    ON sessions (master_id);
CREATE INDEX idx_sessions_scheduled_at ON sessions (scheduled_at);
CREATE INDEX idx_sessions_status       ON sessions (status);
CREATE INDEX idx_sessions_system_id    ON sessions (system_id);
CREATE INDEX idx_sessions_free_seats   ON sessions (free_seats) WHERE free_seats > 0;
