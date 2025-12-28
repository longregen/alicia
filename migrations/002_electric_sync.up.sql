-- Add version tracking for sync
ALTER TABLE alicia_messages ADD COLUMN IF NOT EXISTS electric_version BIGINT DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_messages_electric_version
    ON alicia_messages(conversation_id, electric_version)
    WHERE deleted_at IS NULL;

-- Auto-increment version on update
CREATE OR REPLACE FUNCTION increment_message_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.electric_version = COALESCE(OLD.electric_version, 0) + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER message_version_trigger
    BEFORE UPDATE ON alicia_messages
    FOR EACH ROW EXECUTE FUNCTION increment_message_version();
