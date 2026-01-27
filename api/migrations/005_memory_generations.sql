CREATE TABLE IF NOT EXISTS memory_generations (
    id                TEXT PRIMARY KEY,
    conversation_id   TEXT NOT NULL REFERENCES conversations(id),
    message_id        TEXT NOT NULL REFERENCES messages(id),
    memory_content    TEXT NOT NULL,

    -- Extraction prompt
    extract_prompt_name    TEXT,
    extract_prompt_version INT,

    -- Dimension results (1-5 rating + LLM thinking)
    importance_rating          INT,
    importance_thinking        TEXT,
    importance_prompt_name     TEXT,
    importance_prompt_version  INT,

    historical_rating          INT,
    historical_thinking        TEXT,
    historical_prompt_name     TEXT,
    historical_prompt_version  INT,

    personal_rating          INT,
    personal_thinking        TEXT,
    personal_prompt_name     TEXT,
    personal_prompt_version  INT,

    factual_rating          INT,
    factual_thinking        TEXT,
    factual_prompt_name     TEXT,
    factual_prompt_version  INT,

    -- Rerank result
    rerank_decision        TEXT,
    rerank_prompt_name     TEXT,
    rerank_prompt_version  INT,

    -- Outcome
    accepted    BOOLEAN NOT NULL DEFAULT FALSE,
    memory_id   TEXT REFERENCES memories(id),

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memgen_conv ON memory_generations(conversation_id);
CREATE INDEX IF NOT EXISTS idx_memgen_msg ON memory_generations(message_id);
CREATE INDEX IF NOT EXISTS idx_memgen_memory ON memory_generations(memory_id) WHERE memory_id IS NOT NULL;
