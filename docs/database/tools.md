# alicia_tools

Registry of available tools with their schemas and configurations.

## Schema

```sql
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
```

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `at_` prefix |
| `name` | TEXT | Unique tool name |
| `description` | TEXT | Human-readable description |
| `schema` | JSONB | JSON Schema for input parameters |
| `enabled` | BOOLEAN | Whether tool is available for use |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft deletion timestamp |

## Indexes

```sql
CREATE INDEX idx_tools_name ON alicia_tools(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_enabled ON alicia_tools(enabled) WHERE deleted_at IS NULL;
```

## Relationships

- **Has many** [alicia_tool_uses](./tool_uses.md) via `tool_name`
- **Has many** [optimized_tools](./optimized_tools.md) via `tool_id`

## Schema Format

The `schema` field follows JSON Schema format:

```json
{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Search query"
    },
    "limit": {
      "type": "integer",
      "default": 10
    }
  },
  "required": ["query"]
}
```

## MCP Integration

Tools can be provided by MCP servers. When an MCP server connects, its tools are registered in this table. See [alicia_mcp_servers](./mcp_servers.md).

## Example Queries

```sql
-- List all enabled tools
SELECT name, description
FROM alicia_tools
WHERE enabled = true
  AND deleted_at IS NULL
ORDER BY name;

-- Get tool schema
SELECT schema
FROM alicia_tools
WHERE name = 'web_search'
  AND deleted_at IS NULL;
```

## See Also

- [ToolUseRequest Protocol](../protocol/04-message-types/06-tool-use-request.md)
- [ToolUseResult Protocol](../protocol/04-message-types/07-tool-use-result.md)
