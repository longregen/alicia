CREATE TABLE notes (
    id         TEXT PRIMARY KEY,               -- client-generated UUID
    user_id    TEXT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    content    TEXT NOT NULL DEFAULT '',
    embedding  vector(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_notes_user ON notes(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_embed ON notes USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100) WHERE deleted_at IS NULL;

-- Add notes preferences to user_preferences
ALTER TABLE user_preferences
    ADD COLUMN IF NOT EXISTS notes_similarity_threshold REAL NOT NULL DEFAULT 0.7,
    ADD COLUMN IF NOT EXISTS notes_max_count INTEGER NOT NULL DEFAULT 3;
