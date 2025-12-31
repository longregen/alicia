-- Migration 004: Memory Voting Fix
-- Adds support for voting on memory_usage and memory_extraction

-- ============================================================================
-- Update alicia_votes target_type constraint
-- ============================================================================

-- Drop existing CHECK constraint
ALTER TABLE alicia_votes DROP CONSTRAINT IF EXISTS alicia_votes_target_type_check;

-- Add updated CHECK constraint with memory_usage and memory_extraction
ALTER TABLE alicia_votes ADD CONSTRAINT alicia_votes_target_type_check
    CHECK (target_type IN ('message', 'sentence', 'tool_use', 'memory', 'reasoning', 'memory_usage', 'memory_extraction'));

-- ============================================================================
-- Add indexes for memory voting
-- ============================================================================

-- Index for memory_usage votes (simple lookup by memory usage ID)
CREATE INDEX idx_votes_memory_usage ON alicia_votes(target_id)
    WHERE target_type = 'memory_usage' AND deleted_at IS NULL;

-- Index for memory_extraction votes (composite lookup by memory ID + message ID)
CREATE INDEX idx_votes_memory_extraction ON alicia_votes(target_id, message_id)
    WHERE target_type = 'memory_extraction' AND deleted_at IS NULL;

-- ============================================================================
-- Track source message for memory extraction
-- ============================================================================

-- Add column to track which message a memory was extracted from
ALTER TABLE alicia_memory ADD COLUMN IF NOT EXISTS source_message_id TEXT REFERENCES alicia_messages(id);

COMMENT ON COLUMN alicia_memory.source_message_id IS 'Message from which this memory was extracted (for memory_extraction voting)';
