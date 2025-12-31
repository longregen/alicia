-- Migration 004 Rollback: Memory Voting Fix

-- ============================================================================
-- Remove source message tracking
-- ============================================================================

ALTER TABLE alicia_memory DROP COLUMN IF EXISTS source_message_id;

-- ============================================================================
-- Remove memory voting indexes
-- ============================================================================

DROP INDEX IF EXISTS idx_votes_memory_extraction;
DROP INDEX IF EXISTS idx_votes_memory_usage;

-- ============================================================================
-- Restore original alicia_votes target_type constraint
-- ============================================================================

-- Drop expanded CHECK constraint
ALTER TABLE alicia_votes DROP CONSTRAINT IF EXISTS alicia_votes_target_type_check;

-- Restore original CHECK constraint
ALTER TABLE alicia_votes ADD CONSTRAINT alicia_votes_target_type_check
    CHECK (target_type IN ('message', 'tool_use', 'memory', 'reasoning'));
