-- Migration: Add user_edited tracking for assistant messages
-- This tracks when users edit assistant messages, which is valuable training data

ALTER TABLE alicia_messages
ADD COLUMN user_edited BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN user_edited_at TIMESTAMP;

-- Index for finding user-edited messages (for training data export)
CREATE INDEX idx_messages_user_edited ON alicia_messages(user_edited, user_edited_at)
WHERE user_edited = TRUE AND deleted_at IS NULL;

COMMENT ON COLUMN alicia_messages.user_edited IS 'True if a user edited this assistant message (valuable for training)';
COMMENT ON COLUMN alicia_messages.user_edited_at IS 'Timestamp when the user edited the message';
