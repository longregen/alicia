-- Alicia Database Schema

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS http;

-- ============================================
-- CONVERSATIONS
-- ============================================
CREATE TABLE conversations (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL,
    title          TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'active',
    tip_message_id TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_conv_user ON conversations(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_conv_updated ON conversations(user_id, updated_at DESC) WHERE deleted_at IS NULL;

-- ============================================
-- MESSAGES
-- ============================================
CREATE TABLE messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    previous_id     TEXT REFERENCES messages(id),
    branch_index    SMALLINT NOT NULL DEFAULT 0,
    role            TEXT NOT NULL,
    content         TEXT NOT NULL,
    reasoning       TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending, streaming, completed, error
    trace_id        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_msg_conv_prev ON messages(conversation_id, previous_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_msg_conv ON messages(conversation_id, created_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_msg_trace ON messages(trace_id) WHERE trace_id IS NOT NULL;

ALTER TABLE conversations ADD CONSTRAINT fk_conv_tip
    FOREIGN KEY (tip_message_id) REFERENCES messages(id);

-- ============================================
-- MESSAGE FEEDBACK (thumbs up/down + note)
-- ============================================
CREATE TABLE message_feedback (
    id         TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id),
    rating     SMALLINT NOT NULL,  -- -1 = down, 0 = neutral, 1 = up
    note       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_msg_feedback_msg ON message_feedback(message_id);
CREATE INDEX idx_msg_feedback_rating ON message_feedback(rating);

-- ============================================
-- TOOL USES
-- ============================================
CREATE TABLE tool_uses (
    id         TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id),
    tool_name  TEXT NOT NULL,
    arguments  JSONB NOT NULL DEFAULT '{}',
    result     JSONB,
    status     TEXT NOT NULL DEFAULT 'pending',
    error      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tu_message ON tool_uses(message_id);

-- ============================================
-- TOOL USE FEEDBACK (was this call helpful?)
-- ============================================
CREATE TABLE tool_use_feedback (
    id          TEXT PRIMARY KEY,
    tool_use_id TEXT NOT NULL REFERENCES tool_uses(id),
    rating      SMALLINT NOT NULL,  -- -1 = harmful, 0 = neutral, 1 = helpful
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tu_feedback_tu ON tool_use_feedback(tool_use_id);

-- ============================================
-- MEMORIES
-- ============================================
CREATE TABLE memories (
    id             TEXT PRIMARY KEY,
    content        TEXT NOT NULL,
    embedding      vector(1024),
    importance     REAL NOT NULL DEFAULT 0.5,
    pinned         BOOLEAN NOT NULL DEFAULT FALSE,
    archived       BOOLEAN NOT NULL DEFAULT FALSE,
    source_msg_id  TEXT REFERENCES messages(id),
    tags           TEXT[] NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,
    deleted_reason TEXT  -- free-form feedback on why it was deleted
);

CREATE INDEX idx_mem_embed ON memories USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100) WHERE deleted_at IS NULL AND archived = FALSE;
CREATE INDEX idx_mem_importance ON memories(importance DESC)
    WHERE deleted_at IS NULL AND archived = FALSE;
CREATE INDEX idx_mem_tags ON memories USING GIN(tags)
    WHERE deleted_at IS NULL AND archived = FALSE;

-- ============================================
-- MEMORY USES (when a memory was retrieved for a message)
-- ============================================
CREATE TABLE memory_uses (
    id              TEXT PRIMARY KEY,
    memory_id       TEXT NOT NULL REFERENCES memories(id),
    message_id      TEXT NOT NULL REFERENCES messages(id),
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    similarity      REAL NOT NULL,  -- cosine similarity score
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mem_use_memory ON memory_uses(memory_id);
CREATE INDEX idx_mem_use_msg ON memory_uses(message_id);
CREATE INDEX idx_mem_use_conv ON memory_uses(conversation_id);

-- ============================================
-- MEMORY USE FEEDBACK (was this retrieval appropriate?)
-- ============================================
CREATE TABLE memory_use_feedback (
    id            TEXT PRIMARY KEY,
    memory_use_id TEXT NOT NULL REFERENCES memory_uses(id),
    rating        SMALLINT NOT NULL,  -- -1 = irrelevant, 0 = neutral, 1 = relevant
    note          TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mem_use_feedback_mu ON memory_use_feedback(memory_use_id);

-- ============================================
-- TOOLS (definitions)
-- ============================================
CREATE TABLE tools (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    schema      JSONB NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tools_enabled ON tools(enabled) WHERE enabled = TRUE;

-- ============================================
-- MCP SERVERS
-- ============================================
CREATE TABLE mcp_servers (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL UNIQUE,
    transport_type TEXT NOT NULL,
    command        TEXT NOT NULL DEFAULT '',
    args           TEXT[] NOT NULL DEFAULT '{}',
    url            TEXT NOT NULL DEFAULT '',
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_mcp_enabled ON mcp_servers(enabled) WHERE deleted_at IS NULL AND enabled = TRUE;

-- Seed default MCP servers
INSERT INTO mcp_servers (id, name, transport_type, command, args, enabled) VALUES
    ('mcp_garden', 'garden', 'stdio', 'mcp-garden', '{}', TRUE),
    ('mcp_web', 'web', 'stdio', 'mcp-web', '{}', TRUE),
    ('mcp_deno_calc', 'deno-calc', 'stdio', 'mcp-deno-calc', '{}', TRUE)
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- USER PREFERENCES
-- ============================================
CREATE TABLE user_preferences (
    user_id                 TEXT PRIMARY KEY,

    -- Appearance
    theme                   TEXT NOT NULL DEFAULT 'system',

    -- Voice
    audio_output_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    voice_speed             REAL NOT NULL DEFAULT 1.0,

    -- Memory thresholds (1-5 star rating, memory must pass ALL thresholds)
    memory_min_importance   INTEGER NOT NULL DEFAULT 3,
    memory_min_historical   INTEGER NOT NULL DEFAULT 2,
    memory_min_personal     INTEGER NOT NULL DEFAULT 2,
    memory_min_factual      INTEGER NOT NULL DEFAULT 3,

    -- Memory retrieval
    memory_retrieval_count  INTEGER NOT NULL DEFAULT 10,

    -- Agent
    max_tokens              INTEGER NOT NULL DEFAULT 4096,

    -- Pareto exploration
    pareto_target_score     REAL NOT NULL DEFAULT 3.0,
    pareto_max_generations  INTEGER NOT NULL DEFAULT 5,
    pareto_branches_per_gen INTEGER NOT NULL DEFAULT 3,
    pareto_archive_size     INTEGER NOT NULL DEFAULT 50,
    pareto_enable_crossover BOOLEAN NOT NULL DEFAULT TRUE,

    -- UI behavior
    confirm_delete_memory   BOOLEAN NOT NULL DEFAULT TRUE,
    show_relevance_scores   BOOLEAN NOT NULL DEFAULT FALSE,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_prefs_updated ON user_preferences(updated_at DESC);

-- ============================================
-- QUERY EMBEDDING FUNCTION
-- ============================================
-- query_embedding(text) - calls the embedding server and returns a vector
-- This function is overridden on startup by start-all with the correct
-- LLM_URL, EMBEDDING_MODEL, and EMBEDDING_DIMENSIONS from .env
--
-- Usage:
--   SELECT * FROM memories ORDER BY embedding <-> query_embedding('topic') LIMIT 10;
CREATE OR REPLACE FUNCTION query_embedding(
    input_text TEXT,
    url TEXT DEFAULT 'http://localhost:8000/v1/embeddings',
    model TEXT DEFAULT 'qwen3-embedding-0.6b',
    dims INT DEFAULT 1024
) RETURNS vector AS $$
DECLARE
    response http_response;
    embedding_json jsonb;
BEGIN
    SELECT * INTO response FROM http_post(
        url,
        json_build_object('input', input_text, 'model', model)::text,
        'application/json'
    );

    IF response.status != 200 THEN
        RAISE EXCEPTION 'Embedding API returned status %: %', response.status, response.content;
    END IF;

    embedding_json := response.content::jsonb->'data'->0->'embedding';
    RETURN embedding_json::text::vector(dims);
END;
$$ LANGUAGE plpgsql;
