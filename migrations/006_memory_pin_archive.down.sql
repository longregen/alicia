-- Migration 006: Rollback - Remove pinned and archived fields from memory table

DROP INDEX IF EXISTS idx_memory_archived;
DROP INDEX IF EXISTS idx_memory_pinned;

ALTER TABLE alicia_memory
    DROP COLUMN IF EXISTS archived,
    DROP COLUMN IF EXISTS pinned;
