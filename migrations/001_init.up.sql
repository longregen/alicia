-- Alicia Database Schema
-- Migration 001: Consolidated initial schema

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

-- Helper function to generate random IDs with prefixes
CREATE OR REPLACE FUNCTION generate_random_id(prefix TEXT) RETURNS TEXT AS $$
BEGIN
    RETURN prefix || '_' || encode(gen_random_bytes(12), 'base64');
END;
$$ LANGUAGE plpgsql;

-- Create enum types
CREATE TYPE message_role AS ENUM ('user', 'assistant', 'system');
CREATE TYPE tool_status AS ENUM ('pending', 'running', 'success', 'error', 'cancelled');
CREATE TYPE conversation_status AS ENUM ('active', 'archived', 'deleted');
CREATE TYPE audio_type AS ENUM ('input', 'output');
CREATE TYPE sync_status AS ENUM ('pending', 'synced', 'conflict');
CREATE TYPE completion_status AS ENUM ('pending', 'streaming', 'completed', 'failed');
CREATE TYPE optimization_status AS ENUM ('pending', 'running', 'completed', 'failed');

-- ============================================================================
-- system_prompt_versions (needed before alicia_conversations for FK)
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
-- alicia_conversations
-- ============================================================================
CREATE TABLE alicia_conversations (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ac'),
    user_id TEXT NOT NULL DEFAULT 'default_user',
    title TEXT NOT NULL DEFAULT '',
    status conversation_status NOT NULL DEFAULT 'active',
    livekit_room_name TEXT,
    preferences JSONB DEFAULT '{}',
    last_client_stanza_id INTEGER NOT NULL DEFAULT 0,
    last_server_stanza_id INTEGER NOT NULL DEFAULT -1,
    system_prompt_version_id TEXT REFERENCES system_prompt_versions(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_conversations_status ON alicia_conversations(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversations_livekit_room ON alicia_conversations(livekit_room_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversations_created_at ON alicia_conversations(created_at DESC);
CREATE INDEX idx_conversations_user_created ON alicia_conversations(user_id, created_at DESC) WHERE deleted_at IS NULL;

COMMENT ON COLUMN alicia_conversations.user_id IS 'User identifier for multi-user support and cross-device sync';
COMMENT ON COLUMN alicia_conversations.last_client_stanza_id IS 'Last stanza ID received from the client (for reconnection support)';
COMMENT ON COLUMN alicia_conversations.last_server_stanza_id IS 'Last stanza ID sent by the server - negative values (for reconnection support)';
COMMENT ON COLUMN alicia_conversations.system_prompt_version_id IS 'Version of the main system prompt used for this conversation';

-- ============================================================================
-- alicia_messages
-- ============================================================================
CREATE TABLE alicia_messages (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('am'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    sequence_number INTEGER NOT NULL,
    previous_id TEXT REFERENCES alicia_messages(id),
    message_role message_role NOT NULL,
    contents TEXT NOT NULL DEFAULT '',
    local_id TEXT,
    server_id TEXT,
    sync_status sync_status NOT NULL DEFAULT 'synced',
    synced_at TIMESTAMP,
    completion_status completion_status NOT NULL DEFAULT 'completed',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_messages_conversation ON alicia_messages(conversation_id, sequence_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_previous ON alicia_messages(previous_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_local_id ON alicia_messages(local_id) WHERE local_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_messages_server_id ON alicia_messages(server_id) WHERE server_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_messages_sync_status ON alicia_messages(conversation_id, sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_completion_status ON alicia_messages(completion_status) WHERE deleted_at IS NULL AND completion_status != 'completed';

COMMENT ON COLUMN alicia_messages.local_id IS 'Client-generated ID before server assignment (for offline support)';
COMMENT ON COLUMN alicia_messages.server_id IS 'Canonical server-assigned ID (for offline support)';
COMMENT ON COLUMN alicia_messages.sync_status IS 'Synchronization state: pending, synced, or conflict';
COMMENT ON COLUMN alicia_messages.synced_at IS 'Timestamp when the message was last synced with the server';
COMMENT ON COLUMN alicia_messages.completion_status IS 'Tracks message completion: pending (created), streaming (being generated), completed (fully generated), failed (error during generation)';

-- ============================================================================
-- Add tip_message_id to conversations (after messages table exists)
-- ============================================================================
ALTER TABLE alicia_conversations ADD COLUMN tip_message_id TEXT REFERENCES alicia_messages(id);

CREATE INDEX idx_conversations_tip_message ON alicia_conversations(tip_message_id) WHERE deleted_at IS NULL;

COMMENT ON COLUMN alicia_conversations.tip_message_id IS 'Current head of the message chain for this conversation, enables message branching';

-- ============================================================================
-- alicia_sentences
-- ============================================================================
CREATE TABLE alicia_sentences (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ams'),
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    sentence_sequence_number INTEGER NOT NULL,
    text TEXT NOT NULL DEFAULT '',
    audio_type audio_type,
    audio_format TEXT,
    duration_ms INTEGER,
    audio_bytesize INTEGER,
    audio_data BYTEA,
    meta JSONB DEFAULT '{}',
    completion_status completion_status NOT NULL DEFAULT 'completed',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_sentences_message ON alicia_sentences(message_id, sentence_sequence_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_sentences_completion_status ON alicia_sentences(completion_status) WHERE deleted_at IS NULL AND completion_status != 'completed';
CREATE INDEX idx_sentences_orphaned ON alicia_sentences(message_id, completion_status, created_at) WHERE deleted_at IS NULL AND completion_status IN ('pending', 'streaming', 'failed');

COMMENT ON COLUMN alicia_sentences.completion_status IS 'Tracks sentence completion: pending (created), streaming (being sent), completed (fully sent), failed (error during streaming)';

-- ============================================================================
-- alicia_audio
-- ============================================================================
CREATE TABLE alicia_audio (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('aa'),
    message_id TEXT REFERENCES alicia_messages(id) ON DELETE SET NULL,
    audio_type audio_type NOT NULL,
    audio_format TEXT NOT NULL,
    audio_data BYTEA,
    duration_ms INTEGER,
    transcription TEXT,
    livekit_track_sid TEXT,
    transcription_meta JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_audio_message ON alicia_audio(message_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_audio_livekit_track ON alicia_audio(livekit_track_sid) WHERE livekit_track_sid IS NOT NULL AND deleted_at IS NULL;

-- ============================================================================
-- alicia_memory
-- ============================================================================
CREATE TABLE alicia_memory (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('amem'),
    content TEXT NOT NULL,
    embeddings vector(1024),
    embeddings_info JSONB DEFAULT '{}',
    importance REAL DEFAULT 0.5,
    confidence REAL DEFAULT 1.0,
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    created_by TEXT,
    source_type TEXT,
    source_info JSONB DEFAULT '{}',
    source_message_id TEXT REFERENCES alicia_messages(id),
    tags TEXT[] DEFAULT '{}',
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_memory_embeddings ON alicia_memory USING ivfflat (embeddings vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_memory_importance ON alicia_memory(importance DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_tags ON alicia_memory USING gin(tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_pinned ON alicia_memory(pinned) WHERE deleted_at IS NULL AND pinned = TRUE;
CREATE INDEX idx_memory_archived ON alicia_memory(archived) WHERE deleted_at IS NULL;

COMMENT ON COLUMN alicia_memory.pinned IS 'Whether this memory is pinned for priority access';
COMMENT ON COLUMN alicia_memory.archived IS 'Whether this memory is archived and hidden from normal views';
COMMENT ON COLUMN alicia_memory.source_message_id IS 'Message from which this memory was extracted (for memory_extraction voting)';

-- ============================================================================
-- alicia_memory_used
-- ============================================================================
CREATE TABLE alicia_memory_used (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('amu'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    memory_id TEXT NOT NULL REFERENCES alicia_memory(id) ON DELETE CASCADE,
    query_prompt TEXT,
    query_prompt_meta JSONB DEFAULT '{}',
    similarity_score REAL,
    meta JSONB DEFAULT '{}',
    position_in_results INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_memory_used_conversation ON alicia_memory_used(conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_used_message ON alicia_memory_used(message_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_used_memory ON alicia_memory_used(memory_id) WHERE deleted_at IS NULL;

-- ============================================================================
-- alicia_tools
-- ============================================================================
CREATE TABLE alicia_tools (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('at'),
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    schema JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_tools_name ON alicia_tools(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_enabled ON alicia_tools(enabled) WHERE deleted_at IS NULL;

-- ============================================================================
-- alicia_tool_uses
-- ============================================================================
CREATE TABLE alicia_tool_uses (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('atu'),
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    tool_arguments JSONB DEFAULT '{}',
    tool_result JSONB,
    status tool_status NOT NULL DEFAULT 'pending',
    error_message TEXT,
    sequence_number INTEGER NOT NULL,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_tool_uses_message ON alicia_tool_uses(message_id, sequence_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_tool_uses_status ON alicia_tool_uses(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tool_uses_tool_name ON alicia_tool_uses(tool_name) WHERE deleted_at IS NULL;

-- ============================================================================
-- alicia_reasoning_steps
-- ============================================================================
CREATE TABLE alicia_reasoning_steps (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ar'),
    message_id TEXT NOT NULL REFERENCES alicia_messages(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    sequence_number INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_reasoning_steps_message ON alicia_reasoning_steps(message_id, sequence_number) WHERE deleted_at IS NULL;

-- ============================================================================
-- alicia_user_conversation_commentaries
-- ============================================================================
CREATE TABLE alicia_user_conversation_commentaries (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('aucc'),
    content TEXT NOT NULL,
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_id TEXT REFERENCES alicia_messages(id) ON DELETE SET NULL,
    created_by TEXT,
    meta JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_commentaries_conversation ON alicia_user_conversation_commentaries(conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commentaries_message ON alicia_user_conversation_commentaries(message_id) WHERE message_id IS NOT NULL AND deleted_at IS NULL;

-- ============================================================================
-- alicia_meta
-- ============================================================================
CREATE TABLE alicia_meta (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('amt'),
    ref TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_meta_ref ON alicia_meta(ref) WHERE deleted_at IS NULL;
CREATE INDEX idx_meta_ref_key ON alicia_meta(ref, key) WHERE deleted_at IS NULL;

-- ============================================================================
-- alicia_mcp_servers
-- ============================================================================
CREATE TABLE alicia_mcp_servers (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('amcp'),
    name TEXT NOT NULL UNIQUE,
    transport_type TEXT NOT NULL CHECK (transport_type IN ('stdio', 'sse', 'http')),
    command TEXT,
    args TEXT[],
    env JSONB,
    url TEXT,
    api_key TEXT,
    auto_reconnect BOOLEAN NOT NULL DEFAULT TRUE,
    reconnect_delay INTEGER NOT NULL DEFAULT 5,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mcp_servers_name ON alicia_mcp_servers(name);

-- ============================================================================
-- alicia_votes
-- ============================================================================
CREATE TABLE alicia_votes (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('av'),
    conversation_id TEXT NOT NULL REFERENCES alicia_conversations(id) ON DELETE CASCADE,
    message_id TEXT REFERENCES alicia_messages(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK (target_type IN ('message', 'sentence', 'tool_use', 'memory', 'reasoning', 'memory_usage', 'memory_extraction')),
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
CREATE INDEX idx_votes_memory_usage ON alicia_votes(target_id) WHERE target_type = 'memory_usage' AND deleted_at IS NULL;
CREATE INDEX idx_votes_memory_extraction ON alicia_votes(target_id, message_id) WHERE target_type = 'memory_extraction' AND deleted_at IS NULL;

COMMENT ON COLUMN alicia_votes.target_type IS 'Type of entity being voted on: message, sentence, tool_use, memory, reasoning, memory_usage, memory_extraction';
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
-- prompt_optimization_runs
-- ============================================================================
CREATE TABLE prompt_optimization_runs (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('aor'),
    signature_name VARCHAR(255) NOT NULL,
    status optimization_status NOT NULL DEFAULT 'pending',
    config JSONB DEFAULT '{}',
    best_score REAL,
    iterations INTEGER NOT NULL DEFAULT 0,
    dimension_weights JSONB DEFAULT '{"successRate": 0.25, "quality": 0.20, "efficiency": 0.15, "robustness": 0.15, "generalization": 0.10, "diversity": 0.10, "innovation": 0.05}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_optimization_runs_status ON prompt_optimization_runs(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_signature ON prompt_optimization_runs(signature_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_created ON prompt_optimization_runs(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_completed ON prompt_optimization_runs(completed_at DESC) WHERE deleted_at IS NULL AND completed_at IS NOT NULL;

COMMENT ON TABLE prompt_optimization_runs IS 'GEPA optimization run records tracking prompt optimization sessions';
COMMENT ON COLUMN prompt_optimization_runs.config IS 'Optimization configuration including hyperparameters and constraints';

-- ============================================================================
-- prompt_candidates
-- ============================================================================
CREATE TABLE prompt_candidates (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('apc'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    instructions TEXT NOT NULL,
    demos JSONB,
    coverage INTEGER NOT NULL DEFAULT 0,
    avg_score REAL,
    generation INTEGER NOT NULL DEFAULT 0,
    parent_id TEXT REFERENCES prompt_candidates(id) ON DELETE SET NULL,
    dimension_scores JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_candidates_run ON prompt_candidates(run_id, generation) WHERE deleted_at IS NULL;
CREATE INDEX idx_candidates_score ON prompt_candidates(run_id, avg_score DESC NULLS LAST) WHERE deleted_at IS NULL;
CREATE INDEX idx_candidates_parent ON prompt_candidates(parent_id) WHERE parent_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_candidates_coverage ON prompt_candidates(run_id, coverage DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE prompt_candidates IS 'Candidate prompts representing the Pareto frontier during optimization';
COMMENT ON COLUMN prompt_candidates.coverage IS 'Number of examples this candidate covers successfully';
COMMENT ON COLUMN prompt_candidates.generation IS 'Generation number in the evolutionary optimization process';

-- ============================================================================
-- prompt_evaluations
-- ============================================================================
CREATE TABLE prompt_evaluations (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ape'),
    candidate_id TEXT NOT NULL REFERENCES prompt_candidates(id) ON DELETE CASCADE,
    example_id VARCHAR(255) NOT NULL,
    score REAL NOT NULL,
    feedback TEXT,
    trace JSONB,
    dimension_scores JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_evaluations_candidate ON prompt_evaluations(candidate_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_evaluations_example ON prompt_evaluations(example_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_evaluations_score ON prompt_evaluations(candidate_id, score DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE prompt_evaluations IS 'Evaluation results for candidate prompts against test examples';
COMMENT ON COLUMN prompt_evaluations.trace IS 'Execution trace for debugging and analysis';

-- ============================================================================
-- optimized_programs
-- ============================================================================
CREATE TABLE optimized_programs (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('op'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    signature_name VARCHAR(255) NOT NULL,
    instructions TEXT NOT NULL,
    demos JSONB,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_programs_run ON optimized_programs(run_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_programs_signature ON optimized_programs(signature_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_programs_created ON optimized_programs(created_at DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE optimized_programs IS 'Final optimized prompt programs ready for deployment';

-- ============================================================================
-- optimized_tools
-- ============================================================================
CREATE TABLE optimized_tools (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ot'),
    tool_id TEXT NOT NULL REFERENCES alicia_tools(id) ON DELETE CASCADE,
    optimized_description TEXT NOT NULL,
    optimized_schema JSONB,
    result_template TEXT,
    examples JSONB,
    version INTEGER NOT NULL DEFAULT 1,
    score REAL,
    optimized_at TIMESTAMP NOT NULL DEFAULT NOW(),
    active BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_optimized_tools_tool ON optimized_tools(tool_id, version DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimized_tools_active ON optimized_tools(tool_id, active) WHERE deleted_at IS NULL AND active = true;
CREATE INDEX idx_optimized_tools_score ON optimized_tools(score DESC NULLS LAST) WHERE deleted_at IS NULL;

COMMENT ON TABLE optimized_tools IS 'Optimized tool configurations with improved descriptions and schemas';
COMMENT ON COLUMN optimized_tools.active IS 'Whether this optimized version is currently in use';

-- ============================================================================
-- tool_result_formatters
-- ============================================================================
CREATE TABLE tool_result_formatters (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('trf'),
    tool_name VARCHAR(255) NOT NULL UNIQUE,
    template TEXT NOT NULL,
    max_length INTEGER NOT NULL DEFAULT 2000,
    summarize_at INTEGER NOT NULL DEFAULT 1000,
    summary_prompt TEXT,
    key_fields JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_formatters_tool_name ON tool_result_formatters(tool_name) WHERE deleted_at IS NULL;

COMMENT ON TABLE tool_result_formatters IS 'Learned result formatting rules for optimizing tool output presentation';
COMMENT ON COLUMN tool_result_formatters.summarize_at IS 'Character threshold at which to apply summarization';

-- ============================================================================
-- tool_usage_patterns
-- ============================================================================
CREATE TABLE tool_usage_patterns (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('tup'),
    tool_name VARCHAR(255) NOT NULL,
    user_intent_pattern TEXT NOT NULL,
    success_rate REAL,
    avg_result_quality REAL,
    common_arguments JSONB,
    sample_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_usage_patterns_tool ON tool_usage_patterns(tool_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_usage_patterns_success ON tool_usage_patterns(tool_name, success_rate DESC NULLS LAST) WHERE deleted_at IS NULL;
CREATE INDEX idx_usage_patterns_updated ON tool_usage_patterns(updated_at DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE tool_usage_patterns IS 'Tool usage analytics for identifying optimization opportunities';
COMMENT ON COLUMN tool_usage_patterns.user_intent_pattern IS 'Pattern describing common user intent for this tool usage';

-- ============================================================================
-- pareto_archive
-- ============================================================================
CREATE TABLE pareto_archive (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('pa'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    instructions TEXT NOT NULL,
    demos JSONB DEFAULT '[]',
    dimension_scores JSONB NOT NULL DEFAULT '{}',
    generation INT NOT NULL DEFAULT 0,
    coverage INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pareto_archive_run ON pareto_archive(run_id);
CREATE INDEX idx_pareto_archive_scores ON pareto_archive USING GIN (dimension_scores);

COMMENT ON TABLE pareto_archive IS 'Stores elite solutions from GEPA Pareto optimization, representing non-dominated trade-offs across 7 dimensions';
COMMENT ON COLUMN pareto_archive.dimension_scores IS 'Per-dimension performance metrics: successRate, quality, efficiency, robustness, generalization, diversity, innovation';
COMMENT ON COLUMN pareto_archive.coverage IS 'Number of examples this solution solves best, used for coverage-based selection';

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_conversations_updated_at
    BEFORE UPDATE ON alicia_conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_messages_updated_at
    BEFORE UPDATE ON alicia_messages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sentences_updated_at
    BEFORE UPDATE ON alicia_sentences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_audio_updated_at
    BEFORE UPDATE ON alicia_audio
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_memory_updated_at
    BEFORE UPDATE ON alicia_memory
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_memory_used_updated_at
    BEFORE UPDATE ON alicia_memory_used
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tools_updated_at
    BEFORE UPDATE ON alicia_tools
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tool_uses_updated_at
    BEFORE UPDATE ON alicia_tool_uses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reasoning_steps_updated_at
    BEFORE UPDATE ON alicia_reasoning_steps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_commentaries_updated_at
    BEFORE UPDATE ON alicia_user_conversation_commentaries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_meta_updated_at
    BEFORE UPDATE ON alicia_meta
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_mcp_servers_updated_at
    BEFORE UPDATE ON alicia_mcp_servers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_votes_updated_at
    BEFORE UPDATE ON alicia_votes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notes_updated_at
    BEFORE UPDATE ON alicia_notes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_session_stats_updated_at
    BEFORE UPDATE ON alicia_session_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_usage_patterns_updated_at
    BEFORE UPDATE ON tool_usage_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
