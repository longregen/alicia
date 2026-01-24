# MCP Garden

Database exploration and query execution service. Enables AI agents to safely inspect schemas and run SQL queries against PostgreSQL databases with LLM-powered error hints.

## Tools

### `describe_table`

Get detailed schema information about a database table.

**Parameters:**
- `table` (string, required) - Table name to describe

**Returns:** Column definitions (name, type, nullable, default), primary keys, foreign keys, and row count.

### `execute_sql`

Execute SQL queries with safety controls and smart error hints.

**Parameters:**
- `sql` (string, required) - SQL query to execute
- `allow_mutation` (bool, default: false) - Whether INSERT/UPDATE/DELETE are permitted

**Safety features:**
- Read-only by default (blocks mutations unless explicitly allowed)
- Results capped at 500 rows
- Response size limited to 10KB (configurable)
- UUIDs shortened to `$1`, `$2` format for readability

**Error hints:** When a query fails, the service generates an actionable hint using:
1. LLM via Langfuse prompts (`alicia/garden/sql-debug-system`, `alicia/garden/sql-debug-user`)
2. Fallback rule-based patterns if LLM is unavailable

### `schema_explore`

Ask natural language questions about the database schema.

**Parameters:**
- `question` (string, required) - Natural language question about the schema
- `max_tokens` (int, default: 2048) - Max tokens for response

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `GARDEN_DATABASE_URL` | PostgreSQL connection URL | - |
| `DATABASE_URL` | Fallback if GARDEN_DATABASE_URL not set | - |
| `DATABASE_DOC_PATH` | Path to schema documentation file | - |
| `MCP_MAX_CHARACTER_RESPONSE_SIZE` | Max response size in characters | 10000 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | `https://alicia-data.hjkl.lol` |
| `ENVIRONMENT` | Environment label for telemetry | - |
| `LANGFUSE_HOST` | Langfuse API host | - |
| `LANGFUSE_PUBLIC_KEY` | Langfuse public key | - |
| `LANGFUSE_SECRET_KEY` | Langfuse secret key | - |

## Architecture

```
Agent
  | JSON-RPC 2.0 over stdio
  v
Garden MCP Server (main.go)
  |
  v
PostgreSQL (pgx connection pool)
  |
  v (on error)
LLM API (Langfuse prompts) --> error hint
```

The service implements MCP protocol version `2024-11-05` with these methods:
- `initialize` - Handshake and capability declaration
- `tools/list` - Returns available tools with JSON schemas
- `tools/call` - Executes a tool; accepts `_meta` field for W3C trace context

## Observability

OpenTelemetry traces are sent to SigNoz. Each tool call creates a span with:
- Tool name and parameters
- Execution status and result length
- Distributed trace context propagated from the agent via `_meta`

## Dependencies

- Go 1.24+
- PostgreSQL (via `pgx/v5`)
- OpenAI-compatible LLM API (optional, for error hints and schema_explore)
- Langfuse (optional, for prompt management)
