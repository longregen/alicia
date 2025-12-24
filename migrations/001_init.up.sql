-- Alicia Database Schema
-- Migration 001: Initial schema (consolidated)

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
    embeddings vector(1536),
    embeddings_info JSONB DEFAULT '{}',
    importance REAL DEFAULT 0.5,
    confidence REAL DEFAULT 1.0,
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    created_by TEXT,
    source_type TEXT,
    source_info JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_memory_embeddings ON alicia_memory USING ivfflat (embeddings vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_memory_importance ON alicia_memory(importance DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_memory_tags ON alicia_memory USING gin(tags) WHERE deleted_at IS NULL;

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
