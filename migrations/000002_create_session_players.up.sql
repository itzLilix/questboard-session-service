CREATE TYPE player_status AS ENUM ('active', 'kicked', 'left');

CREATE TABLE session_players (
    session_id  UUID          NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    player_id   UUID          NOT NULL,
    joined_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    status      player_status NOT NULL DEFAULT 'active',
    PRIMARY KEY (session_id, player_id)
);

CREATE INDEX idx_session_players_player_id ON session_players (player_id);
