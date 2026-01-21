-- Rollback: Remove user_edited tracking columns

DROP INDEX IF EXISTS idx_messages_user_edited;

ALTER TABLE alicia_messages
DROP COLUMN IF EXISTS user_edited,
DROP COLUMN IF EXISTS user_edited_at;
