# Alicia

AI assistant with voice, chat, and tool capabilities.

## Services

### Agent (`/agent`)
Go AI assistant backend (~3,800 LOC)

- Receives conversation requests via WebSocket, generates AI responses using LLM APIs
- Manages tool/function calling through Model Context Protocol (MCP)
- Features Pareto-efficient multi-objective response optimization (evolutionary strategy)
- Vector-based semantic memory retrieval with pgvector
- Integrates with OpenTelemetry (SigNoz) and Langfuse for prompts

### API (`/api`)
REST/WebSocket backend for the AI assistant

- Manages conversations, messages, and real-time communication
- WebSocket hub for agents, voice services, and clients
- Memory system with vector embeddings (pgvector) for semantic search
- LiveKit integration for voice/video
- Feedback tracking for RLHF
- User preferences with real-time sync to agent via WebSocket

### User Preferences

User preferences are stored in the `user_preferences` table and synced to the agent in real-time via `TypePreferencesUpdate` WebSocket messages. The agent stores preferences in memory (`agent/preferences.go`) keyed by user ID. Preferences control memory thresholds (importance, historical, personal, factual rated 1-5), memory retrieval count, and max tokens. The frontend (`web/src/stores/preferencesStore.ts`) uses Zustand with debounced saves. Backend validation in `api/server/handlers/preferences.go` enforces value ranges.

### MCP Services (`/mcp`)

| Service | Purpose | Tools |
|---------|---------|-------|
| **deno-calc** | Sandboxed JS/TS execution | `calculate` |
| **garden** | Database exploration + SQL queries | `describe_table`, `execute_sql`, `schema_explore` |
| **web** | Web browsing + content extraction | `read`, `fetch_raw`, `fetch_structured`, `search`, `extract_links`, `extract_metadata`, `screenshot` |

All implement JSON-RPC 2.0 over stdio. Garden has LLM-powered SQL error hints. Web has SSRF protection and headless browser support via Rod.

MCP services have OpenTelemetry instrumentation with distributed tracing via the `_meta` field (W3C Trace Context format).

### Voice (`/voice`)
Real-time voice bridge (~2,300 LOC)

- Transcribes speech to text (ASR/Whisper) and synthesizes responses (TTS/Kokoro)
- LiveKit WebRTC integration for audio streaming
- Voice activity detection using RMS energy threshold
- Sentence-by-sentence TTS queuing

## Infrastructure

Services are deployed via NixOS at /persist/colmena/colmena/hosts/america/giga/services{alicia.nix,ai-llm-voice.nix,langfuse.nix,signoz.nix}

## Telemetry

OpenTelemetry traces are sent to [SigNoz](https://signoz.io/) at OTEL_EXPORTER_OTLP_ENDPOINT=https://alicia-data.hjkl.lol

**Docs:**
- [OpenTelemetry Tracing Guide](https://signoz.io/blog/opentelemetry-tracing/)
- [APM & Distributed Tracing](https://signoz.io/docs/instrumentation/overview/)
- [Frontend Tracing](https://signoz.io/docs/frontend-monitoring/sending-traces-with-opentelemetry/)
- [Collector Configuration](https://signoz.io/docs/opentelemetry-collection-agents/opentelemetry-collector/configuration/)

## Prompts

Prompts are managed with [Langfuse](https://langfuse.com/) (org: `decent`, project: `alicia`) at `langfuse.hjkl.lol`. Credentials are in `.env`: LANGFUSE_HOST, LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY

Langfuse is an open-source LLM engineering platform for tracing, prompt management, and evaluation. Prompts follow the pattern `alicia/<component>/<name>`. [Docs](https://langfuse.com/docs/prompts)

Traces include a `langfuse.trace.name` attribute for easier identification in Langfuse. The agent propagates trace context to MCP services via the `_meta` field in tool call parameters.

### Using prompts in the agent

Use `getPromptText()` to fetch prompts with automatic caching and fallback:

```go
const fallbackPrompt = `Your fallback prompt with {{variable}} placeholders.`

vars := map[string]string{
    "variable": "value",
}
promptText := getPromptText("alicia/component/prompt-name", fallbackPrompt, vars)
```

When adding a new prompt, please create the prompt in langfuse by using the API and the credentials in `.env`. Traces include prompt metadata as span attributes for analysis by prompt version: langfuse.prompt.{name,version}.

This is added automatically when using `getSystemPrompt()` in the agent. The attributes appear on `agent.tool_loop` span (parent) and `llm.chat` spans (each LLM call)

**Docs:**
- [Link Prompts to Traces](https://langfuse.com/docs/prompt-management/features/link-to-traces)
- [Prompt Versioning](https://langfuse.com/docs/prompts/get-started)

