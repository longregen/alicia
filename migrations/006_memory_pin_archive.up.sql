-- Migration 006: Add pinned and archived fields to memory table
-- Allows users to pin important memories and archive old ones

ALTER TABLE alicia_memory
    ADD COLUMN pinned BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN archived BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_memory_pinned ON alicia_memory(pinned) WHERE deleted_at IS NULL AND pinned = TRUE;
CREATE INDEX idx_memory_archived ON alicia_memory(archived) WHERE deleted_at IS NULL;

COMMENT ON COLUMN alicia_memory.pinned IS 'Whether this memory is pinned for priority access';
COMMENT ON COLUMN alicia_memory.archived IS 'Whether this memory is archived and hidden from normal views';
