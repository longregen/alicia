-- Rollback: Revert embedding dimensions from 1024 to 1536

-- Drop the ivfflat index
DROP INDEX IF EXISTS idx_memory_embeddings;

-- Clear existing embeddings (they are incompatible with old dimensions)
UPDATE alicia_memory SET embeddings = NULL, embeddings_info = '{}';

-- Alter the column back to 1536 dimensions
ALTER TABLE alicia_memory ALTER COLUMN embeddings TYPE vector(1536);

-- Recreate the index
CREATE INDEX idx_memory_embeddings ON alicia_memory USING ivfflat (embeddings vector_cosine_ops) WITH (lists = 100);
