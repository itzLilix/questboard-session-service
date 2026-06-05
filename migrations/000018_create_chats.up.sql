-- Chat system: room-scoped chats (one-of session | campaign, like notes/files),
-- members, messages with attachments/mentions, pinned messages, per-chat role
-- permissions, plus the campaign display-attachment and tie-provenance layers.

CREATE TYPE chat_kind          AS ENUM ('general', 'group', 'direct');
CREATE TYPE chat_member_role   AS ENUM ('member', 'admin', 'owner');  -- declaration order = rank: owner > admin > member
CREATE TYPE chat_attach_source AS ENUM ('native', 'imported');

-- ── Chats: the tenant boundary (a standalone session OR a campaign) ──────────
CREATE TABLE chats (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID        REFERENCES sessions  (id) ON DELETE CASCADE,
    campaign_id  UUID        REFERENCES campaigns (id) ON DELETE CASCADE,
    kind         chat_kind   NOT NULL,
    title        VARCHAR(255),                          -- groups only; NULL for general/direct
    picture_url  TEXT,                                   -- groups only, paired with title
    created_by   UUID,                                   -- NULL for system 'general'
    dm_user_low  UUID,                                   -- direct only: LEAST(a, b)
    dm_user_high UUID,                                   -- direct only: GREATEST(a, b)
    archived_at  TIMESTAMPTZ,                            -- soft-archive (dormant on retie / closed); messages stay
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ,                         -- NULL = never used (uninitialized); doubles as import filter + chat-list sort key

    CONSTRAINT chk_chats_scope CHECK (
        (campaign_id IS NOT NULL)::INT + (session_id IS NOT NULL)::INT = 1
    ),
    CONSTRAINT chk_chats_dm_pair CHECK (
        ((kind = 'direct') = (dm_user_low  IS NOT NULL))
        AND ((kind = 'direct') = (dm_user_high IS NOT NULL))
        AND (dm_user_low IS NULL OR dm_user_low < dm_user_high)
    )
);

-- exactly one general chat per room
CREATE UNIQUE INDEX uq_chats_general_session  ON chats (session_id)  WHERE kind = 'general' AND session_id  IS NOT NULL;
CREATE UNIQUE INDEX uq_chats_general_campaign ON chats (campaign_id) WHERE kind = 'general' AND campaign_id IS NOT NULL;

-- one direct chat per (room, participant-pair); groups intentionally allow duplicates
CREATE UNIQUE INDEX uq_chats_dm_session  ON chats (session_id,  dm_user_low, dm_user_high) WHERE kind = 'direct' AND session_id  IS NOT NULL;
CREATE UNIQUE INDEX uq_chats_dm_campaign ON chats (campaign_id, dm_user_low, dm_user_high) WHERE kind = 'direct' AND campaign_id IS NOT NULL;

CREATE INDEX idx_chats_session  ON chats (session_id)  WHERE session_id  IS NOT NULL;
CREATE INDEX idx_chats_campaign ON chats (campaign_id) WHERE campaign_id IS NOT NULL;

-- ── Membership ──────────────────────────────────────────────────────────────
CREATE TABLE chat_members (
    chat_id      UUID             NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    user_id      UUID             NOT NULL,             -- bare UUID, no FK (profile-service owns users)
    role         chat_member_role NOT NULL DEFAULT 'member',
    joined_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    last_read_id UUID,                                   -- UUIDv7 of last read message; unread = ids > this
    PRIMARY KEY (chat_id, user_id)
);
CREATE INDEX idx_chat_members_user ON chat_members (user_id);

-- at most one owner per chat (general/group have exactly one; DMs have none)
CREATE UNIQUE INDEX uq_chat_members_one_owner ON chat_members (chat_id) WHERE role = 'owner';

-- ── Messages ────────────────────────────────────────────────────────────────
CREATE TABLE messages (
    id          UUID        PRIMARY KEY,                 -- UUIDv7 generated in Go (or uuidv7() on PG18); NOT gen_random_uuid()
    chat_id     UUID        NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    sender_id   UUID        NOT NULL,                    -- bare UUID, no FK
    body        TEXT        NOT NULL DEFAULT '',
    reply_to_id UUID        REFERENCES messages (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at   TIMESTAMPTZ,
    deleted_at  TIMESTAMPTZ,                             -- soft delete; keeps replies/mentions valid

    -- required so pinned_messages can use a composite FK on (id, chat_id)
    CONSTRAINT uq_messages_id_chat UNIQUE (id, chat_id)
);
CREATE INDEX idx_messages_chat ON messages (chat_id, id DESC);   -- keyset pagination on UUIDv7

CREATE TABLE message_attachments (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID         NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    url        TEXT         NOT NULL,
    mime_type  VARCHAR(100),
    size_bytes BIGINT
);
CREATE INDEX idx_message_attachments_message ON message_attachments (message_id);

CREATE TABLE message_mentions (
    message_id        UUID NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    mentioned_user_id UUID NOT NULL,                     -- resolved @username -> id at send time
    PRIMARY KEY (message_id, mentioned_user_id)
);
CREATE INDEX idx_message_mentions_user ON message_mentions (mentioned_user_id);

-- ── Pinned messages ─────────────────────────────────────────────────────────
CREATE TABLE pinned_messages (
    chat_id     UUID        NOT NULL,
    message_id  UUID        NOT NULL,
    pinned_by   UUID        NOT NULL,                    -- bare UUID, no FK
    pinned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    order_index SMALLINT,                                -- optional manual ordering; else sort by pinned_at
    PRIMARY KEY (message_id),                            -- a message is pinned at most once
    FOREIGN KEY (message_id, chat_id) REFERENCES messages (id, chat_id) ON DELETE CASCADE
);
CREATE INDEX idx_pinned_messages_chat ON pinned_messages (chat_id, pinned_at DESC);

-- ── Per-chat role permissions (app-enforced config; seed admin+member per chat) ─
CREATE TABLE chat_role_permissions (
    chat_id                UUID             NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    role                   chat_member_role NOT NULL,
    can_send_messages      BOOLEAN          NOT NULL DEFAULT TRUE,
    can_send_files         BOOLEAN          NOT NULL DEFAULT TRUE,
    can_pin_messages       BOOLEAN          NOT NULL DEFAULT TRUE,
    can_change_info        BOOLEAN          NOT NULL DEFAULT FALSE,   -- title / picture
    can_add_members        BOOLEAN          NOT NULL DEFAULT FALSE,
    can_remove_members     BOOLEAN          NOT NULL DEFAULT FALSE,
    can_delete_messages    BOOLEAN          NOT NULL DEFAULT FALSE,   -- others' messages
    can_manage_roles       BOOLEAN          NOT NULL DEFAULT FALSE,   -- promote/demote members (rank-gated)
    can_manage_permissions BOOLEAN          NOT NULL DEFAULT FALSE,   -- edit this table
    PRIMARY KEY (chat_id, role)
);

-- ── Display layer: what a campaign's Chats tab shows (the "migrate" checkbox) ─
CREATE TABLE campaign_chat_attachments (
    campaign_id UUID               NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    chat_id     UUID               NOT NULL REFERENCES chats (id)     ON DELETE CASCADE,
    source      chat_attach_source NOT NULL DEFAULT 'imported',       -- 'native' = born here, 'imported' = migrated in
    attached_by UUID,
    attached_at TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    PRIMARY KEY (campaign_id, chat_id)
);
CREATE INDEX idx_cca_chat ON campaign_chat_attachments (chat_id);
