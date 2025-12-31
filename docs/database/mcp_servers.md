# alicia_mcp_servers

Configuration for Model Context Protocol (MCP) servers providing external tools.

## Schema

```sql
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
```

**Note**: This table has no `deleted_at` column. MCP server configurations are permanently deleted when removed.

## Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | TEXT | Primary key with `amcp_` prefix |
| `name` | TEXT | Unique server name |
| `transport_type` | TEXT | Transport: `stdio`, `sse`, or `http` |
| `command` | TEXT | Command for stdio transport |
| `args` | TEXT[] | Command arguments |
| `env` | JSONB | Environment variables |
| `url` | TEXT | URL for SSE/HTTP transports |
| `api_key` | TEXT | Authentication key |
| `auto_reconnect` | BOOLEAN | Auto-reconnect on disconnect |
| `reconnect_delay` | INTEGER | Seconds between reconnect attempts |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |

## Indexes

```sql
CREATE INDEX idx_mcp_servers_name ON alicia_mcp_servers(name);
```

## Transport Types

| Type | Fields Used | Description |
|------|-------------|-------------|
| `stdio` | `command`, `args`, `env` | Local process with stdin/stdout |
| `sse` | `url`, `api_key` | Server-Sent Events |
| `http` | `url`, `api_key` | HTTP requests |

## Examples

### stdio Transport

```sql
INSERT INTO alicia_mcp_servers (name, transport_type, command, args, env)
VALUES (
    'filesystem',
    'stdio',
    '/usr/local/bin/mcp-filesystem',
    ARRAY['--root', '/home/user/documents'],
    '{"MCP_LOG_LEVEL": "info"}'
);
```

### SSE Transport

```sql
INSERT INTO alicia_mcp_servers (name, transport_type, url, api_key)
VALUES (
    'weather-api',
    'sse',
    'https://api.weather.example.com/mcp',
    'sk-xxx'
);
```

## Tool Registration

When an MCP server connects, its tools are registered in [alicia_tools](./tools.md) and become available for use.

## Example Queries

```sql
-- List all MCP servers
SELECT name, transport_type, auto_reconnect
FROM alicia_mcp_servers;

-- Get stdio server config
SELECT command, args, env
FROM alicia_mcp_servers
WHERE name = 'filesystem';
```

## See Also

- [MCP Client Primer](../MCP_CLIENT_PRIMER.md)
