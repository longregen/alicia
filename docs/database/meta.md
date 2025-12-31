# alicia_meta

Flexible key-value metadata store for any entity.

## Schema

```sql
CREATE TABLE alicia_meta (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('amt'),
    ref TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `amt_` prefix |
| `ref` | TEXT | Reference to parent entity (e.g., `ac_xxx`, `am_xxx`) |
| `key` | TEXT | Metadata key |
| `value` | TEXT | Metadata value |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_meta_ref ON alicia_meta(ref) WHERE deleted_at IS NULL;
CREATE INDEX idx_meta_ref_key ON alicia_meta(ref, key) WHERE deleted_at IS NULL;
```

## Purpose

Provides extensibility without schema changes:

- Store arbitrary metadata on any entity
- Track processing metrics
- Store user preferences
- System configuration

## Common Patterns

```sql
-- Set metadata (no unique constraint, use DELETE+INSERT)
DELETE FROM alicia_meta
WHERE ref = 'ac_xxx' AND key = 'theme' AND deleted_at IS NULL;

INSERT INTO alicia_meta (ref, key, value)
VALUES ('ac_xxx', 'theme', 'dark');

-- Get metadata
SELECT key, value
FROM alicia_meta
WHERE ref = 'ac_xxx'
  AND deleted_at IS NULL;

-- Get specific key
SELECT value
FROM alicia_meta
WHERE ref = 'ac_xxx'
  AND key = 'theme'
  AND deleted_at IS NULL;
```

## Example Use Cases

| ref | key | value |
|-----|-----|-------|
| `ac_xxx` | `theme` | `dark` |
| `ac_xxx` | `tts_speed` | `1.2` |
| `am_xxx` | `model` | `claude-3-opus` |
| `am_xxx` | `tokens_used` | `1523` |
