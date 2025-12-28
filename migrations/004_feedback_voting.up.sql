-- Alicia Database Schema
-- Migration 004: Feedback and voting system

-- ============================================================================
-- alicia_votes
-- ============================================================================
CREATE TABLE alicia_votes (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('av'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_id TEXT REFERENCES alicia_messages(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK (target_type IN ('message', 'tool_use', 'memory', 'reasoning')),
    target_id TEXT NOT NULL,
    vote TEXT NOT NULL CHECK (vote IN ('up', 'down', 'critical')),
    quick_feedback TEXT CHECK (quick_feedback IN ('wrong_tool', 'wrong_params', 'unnecessary', 'missing_context', 'outdated', 'wrong_context', 'too_generic', 'incorrect', 'incorrect_assumption', 'missed_consideration', 'overcomplicated', 'wrong_direction')),
    note TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_votes_conversation ON alicia_votes(conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_votes_message ON alicia_votes(message_id) WHERE message_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_votes_target ON alicia_votes(target_type, target_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_votes_type_vote ON alicia_votes(target_type, vote) WHERE deleted_at IS NULL;

COMMENT ON COLUMN alicia_votes.target_type IS 'Type of entity being voted on: message, tool_use, memory, or reasoning';
COMMENT ON COLUMN alicia_votes.target_id IS 'ID of the target entity being voted on';
COMMENT ON COLUMN alicia_votes.vote IS 'Vote type: up (positive), down (negative), critical (essential - for memories)';
COMMENT ON COLUMN alicia_votes.quick_feedback IS 'Optional predefined feedback category for quick structured feedback';

-- ============================================================================
-- alicia_notes
-- ============================================================================
CREATE TABLE alicia_notes (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('an'),
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('improvement', 'correction', 'context', 'general')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_notes_message ON alicia_notes(message_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_category ON alicia_notes(category) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_created_at ON alicia_notes(created_at DESC) WHERE deleted_at IS NULL;

COMMENT ON COLUMN alicia_notes.category IS 'Note category: improvement (suggestion), correction (factual error), context (clarification), general (freeform)';

-- ============================================================================
-- alicia_session_stats
-- ============================================================================
CREATE TABLE alicia_session_stats (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ass'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_count INTEGER NOT NULL DEFAULT 0,
    tool_call_count INTEGER NOT NULL DEFAULT 0,
    memories_used INTEGER NOT NULL DEFAULT 0,
    session_duration_seconds INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_stats_conversation ON alicia_session_stats(conversation_id);
CREATE INDEX idx_session_stats_created_at ON alicia_session_stats(created_at DESC);

COMMENT ON COLUMN alicia_session_stats.message_count IS 'Total number of messages in the session';
COMMENT ON COLUMN alicia_session_stats.tool_call_count IS 'Total number of tool calls made during the session';
COMMENT ON COLUMN alicia_session_stats.memories_used IS 'Total number of unique memories retrieved during the session';
COMMENT ON COLUMN alicia_session_stats.session_duration_seconds IS 'Total duration of the session in seconds';

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================
CREATE TRIGGER update_votes_updated_at
    BEFORE UPDATE ON alicia_votes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notes_updated_at
    BEFORE UPDATE ON alicia_notes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_session_stats_updated_at
    BEFORE UPDATE ON alicia_session_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
