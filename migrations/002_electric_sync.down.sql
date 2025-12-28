-- Rollback version tracking for sync

DROP TRIGGER IF EXISTS message_version_trigger ON alicia_messages;
DROP FUNCTION IF EXISTS increment_message_version();
DROP INDEX IF EXISTS idx_messages_electric_version;
ALTER TABLE alicia_messages DROP COLUMN IF EXISTS electric_version;
