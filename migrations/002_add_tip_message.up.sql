-- Add tip_message_id column to alicia_conversations for message branching
ALTER TABLE alicia_conversations ADD COLUMN tip_message_id TEXT REFERENCES alicia_messages(id);

-- Backfill: set tip to the latest message per conversation
UPDATE alicia_conversations c
SET tip_message_id = (
    SELECT id FROM alicia_messages m
    WHERE m.conversation_id = c.id
    AND m.deleted_at IS NULL
    ORDER BY m.created_at DESC
    LIMIT 1
);

-- Add index for tip_message_id lookups
CREATE INDEX idx_conversations_tip_message ON alicia_conversations(tip_message_id) WHERE deleted_at IS NULL;

COMMENT ON COLUMN alicia_conversations.tip_message_id IS 'Current head of the message chain for this conversation, enables message branching';
