CREATE TABLE session_commentaries (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID        NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL,
    parent_id   UUID        REFERENCES session_commentaries (id) ON DELETE CASCADE,
    posted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ,
    is_deleted  BOOLEAN     NOT NULL DEFAULT FALSE,
    content     TEXT        NOT NULL
)

CREATE INDEX idx_session_commentaries_session_id ON session_commentaries (session_id)
CREATE INDEX idx_session_commentaries_parent_id ON session_commentaries (parent_id)
