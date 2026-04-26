CREATE TABLE session_files (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID         NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    url         TEXT         NOT NULL,
    mime_type   VARCHAR(100),
    size_bytes  BIGINT,
    uploaded_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_files_session_id ON session_files (session_id);
