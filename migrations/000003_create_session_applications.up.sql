CREATE TYPE application_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE session_applications (
    id            UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID               NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    applicant_id  UUID               NOT NULL,
    message       TEXT,
    status        application_status NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, applicant_id)
);

CREATE INDEX idx_session_applications_status ON session_applications (session_id, status);
