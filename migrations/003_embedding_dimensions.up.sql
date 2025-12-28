-- Migration: Change embedding dimensions from 1536 to 1024
-- This is required for e5-large model which outputs 1024 dimensions

-- Drop the existing ivfflat index (it's dimension-specific)
DROP INDEX IF EXISTS idx_memory_embeddings;

-- Clear existing embeddings (they are incompatible with new dimensions)
-- Memories will need to be re-embedded after this migration
UPDATE alicia_memory SET embeddings = NULL, embeddings_info = '{}';

-- Alter the column to use 1024 dimensions
ALTER TABLE alicia_memory ALTER COLUMN embeddings TYPE vector(1024);

-- Recreate the index with new dimensions
CREATE INDEX idx_memory_embeddings ON alicia_memory USING ivfflat (embeddings vector_cosine_ops) WITH (lists = 100);
