-- Remove tip_message_id column from alicia_conversations
DROP INDEX IF EXISTS idx_conversations_tip_message;
ALTER TABLE alicia_conversations DROP COLUMN IF EXISTS tip_message_id;
