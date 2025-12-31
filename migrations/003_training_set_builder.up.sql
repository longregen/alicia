-- Migration 003: Training Set Builder for GEPA Optimization
-- Adds tables for tracking training examples, system prompt versions, and vote-based datasets

-- ============================================================================
-- system_prompt_versions
-- ============================================================================
CREATE TABLE system_prompt_versions (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('spv'),
    version_number SERIAL,
    prompt_hash TEXT NOT NULL,
    prompt_content TEXT NOT NULL,
    prompt_type TEXT NOT NULL CHECK (prompt_type IN ('main', 'tool_selection', 'memory_selection', 'memory_extraction')),
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMP,
    deactivated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    UNIQUE (prompt_type, version_number),
    UNIQUE (prompt_type, prompt_hash)
);

CREATE INDEX idx_prompt_versions_type_active ON system_prompt_versions(prompt_type, active) WHERE deleted_at IS NULL;

COMMENT ON TABLE system_prompt_versions IS 'Tracks versions of system prompts for GEPA optimization and A/B testing';
COMMENT ON COLUMN system_prompt_versions.prompt_type IS 'Type of prompt: main (base system), tool_selection, memory_selection, memory_extraction';
COMMENT ON COLUMN system_prompt_versions.prompt_hash IS 'SHA-256 hash of prompt_content for deduplication';
COMMENT ON COLUMN system_prompt_versions.active IS 'Whether this version is currently active for new conversations';

-- ============================================================================
-- gepa_training_examples
-- ============================================================================
CREATE TABLE gepa_training_examples (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('gte'),
    task_type TEXT NOT NULL CHECK (task_type IN ('tool_selection', 'memory_selection', 'memory_extraction')),
    vote_id TEXT REFERENCES alicia_votes(id) ON DELETE SET NULL,
    is_positive BOOLEAN NOT NULL,
    inputs JSONB NOT NULL,
    outputs JSONB NOT NULL,
    vote_metadata JSONB,
    source TEXT NOT NULL CHECK (source IN ('vote', 'synthetic')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_training_examples_task ON gepa_training_examples(task_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_training_examples_vote ON gepa_training_examples(vote_id) WHERE vote_id IS NOT NULL;

COMMENT ON TABLE gepa_training_examples IS 'GEPA training examples derived from user votes or synthetic generation';
COMMENT ON COLUMN gepa_training_examples.task_type IS 'GEPA task: tool_selection, memory_selection, or memory_extraction';
COMMENT ON COLUMN gepa_training_examples.is_positive IS 'Whether this is a positive (upvote) or negative (downvote) example';
COMMENT ON COLUMN gepa_training_examples.inputs IS 'GEPA input fields (user_message, context, available_tools, etc.)';
COMMENT ON COLUMN gepa_training_examples.outputs IS 'GEPA output fields (selected_tool, arguments, memories, etc.)';
COMMENT ON COLUMN gepa_training_examples.vote_metadata IS 'Vote metadata: quick_feedback, note, vote_value for diagnostic feedback';
COMMENT ON COLUMN gepa_training_examples.source IS 'Origin of example: vote (user feedback) or synthetic (generated)';

-- ============================================================================
-- Link conversations to prompt versions
-- ============================================================================
ALTER TABLE alicia_conversations
ADD COLUMN system_prompt_version_id TEXT REFERENCES system_prompt_versions(id);

COMMENT ON COLUMN alicia_conversations.system_prompt_version_id IS 'Version of the main system prompt used for this conversation';
