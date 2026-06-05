-- Drop in reverse dependency order; tables take their own indexes/constraints with them.

DROP TABLE IF EXISTS campaign_chat_attachments;
DROP TABLE IF EXISTS chat_role_permissions;
DROP TABLE IF EXISTS pinned_messages;
DROP TABLE IF EXISTS message_mentions;
DROP TABLE IF EXISTS message_attachments;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chat_members;
DROP TABLE IF EXISTS chats;

DROP TYPE IF EXISTS chat_attach_source;
DROP TYPE IF EXISTS chat_member_role;
DROP TYPE IF EXISTS chat_kind;
