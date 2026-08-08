CREATE TYPE chat_attach_source AS ENUM ('native', 'imported');

ALTER TABLE campaign_chat_attachments
    ADD COLUMN source chat_attach_source NOT NULL DEFAULT 'imported';