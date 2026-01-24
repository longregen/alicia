# MCP Deno Calc

Sandboxed JavaScript/TypeScript code execution for calculations and data transformations. Uses Deno's permission system to run code in a completely isolated environment with no network or filesystem access.

## Tools

### `calculate`

Execute JavaScript/TypeScript code in a sandboxed Deno runtime.

**Parameters:**
- `code` (string, required) - JavaScript/TypeScript code to execute

The result of the last expression is returned automatically. Use `console.log()` for additional output. Objects are serialized as formatted JSON.

**Examples:**
```javascript
// Math
Math.sqrt(144) + Math.PI

// Data transformation
const data = [1, 2, 3, 4, 5];
data.map(x => x * x).reduce((a, b) => a + b, 0)

// Async supported
await Promise.resolve(42)
```

## Sandboxing

Code runs with strict Deno permission flags:

| Flag | Effect |
|------|--------|
| `--no-remote` | No fetching code from URLs |
| `--no-npm` | No npm package installation |
| `--no-config` | Ignore deno.json config files |
| `--allow-read=<tmpfile>` | Only read the temp script file |

No other permissions are granted - no network, no filesystem write, no environment variable access.

**Timeout:** 30 seconds per execution, enforced via context deadline.

**Execution flow:**
1. User code wrapped in async IIFE to capture last expression
2. Written to a temporary `.ts` file
3. Executed with restricted Deno permissions
4. Temp file cleaned up after execution
5. Combined stdout/stderr returned as result

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | `https://alicia-data.hjkl.lol` |
| `ENVIRONMENT` | Environment label for telemetry | - |

## Architecture

```
Agent
  | JSON-RPC 2.0 over stdio
  v
Deno Calc MCP Server (main.go)
  |
  v
Deno Runtime (sandboxed subprocess)
  | --no-remote --no-npm --no-config
  v
Isolated code execution (30s timeout)
```

MCP protocol version `2024-11-05`. Methods: `initialize`, `tools/list`, `tools/call`.

## Runtime Dependencies

- Go 1.24+
- **Deno** must be available in `PATH`
