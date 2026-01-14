# HTTP API Server

The Alicia HTTP server provides RESTful endpoints for conversation management, real-time messaging, LiveKit integration, and MCP tool orchestration.

## Overview

The server acts as the central hub for all Alicia clients (web, mobile, CLI), handling conversation persistence, LLM interaction orchestration, memory management, and real-time communication via SSE and WebSocket.

**Entry Point**: `/cmd/alicia/serve.go`

## Architecture

### Initialization Sequence

When you run `alicia serve`, the following initialization occurs:

1. **Configuration Loading**: Environment variables parsed into config struct
2. **Database Connection**: PostgreSQL connection pool created with UTC timezone
3. **Repository Layer**: All domain repositories initialized (conversations, messages, tools, memory, etc.)
4. **Services Layer**: Business logic services created (LLM, memory, optimization, tools)
5. **Adapters**: External service adapters initialized (LiveKit, ASR, TTS, embedding, MCP)
6. **Use Cases**: Application use cases wired with dependencies
7. **HTTP Server**: Chi router configured with middleware and handlers
8. **Graceful Shutdown**: Signal handlers for clean shutdown

### Layered Architecture

```
HTTP Handlers (adapters/http/handlers/)
         ↓
  Use Cases (application/usecases/)
         ↓
   Services (application/services/)
         ↓
Repositories (adapters/postgres/)
         ↓
    PostgreSQL
```

## HTTP API Endpoints

**Server Setup**: `/internal/adapters/http/server.go`

### Health & Monitoring

- `GET /health` - Simple health check (always returns 200 OK)
- `GET /health/detailed` - Detailed health with dependency checks (DB, LLM, ASR, TTS, LiveKit)
- `GET /metrics` - Prometheus metrics endpoint

### Configuration

- `GET /api/v1/config` - Public configuration (no auth required)
  - Returns frontend-safe config (available voices, server capabilities)
  - Used by web/mobile clients during initialization

### Conversations

All require authentication via `Authorization` header.

- `POST /api/v1/conversations` - Create new conversation
- `GET /api/v1/conversations` - List user's conversations
- `GET /api/v1/conversations/{id}` - Get conversation by ID
- `PATCH /api/v1/conversations/{id}` - Update conversation (title, metadata)
- `DELETE /api/v1/conversations/{id}` - Delete conversation

### Messages

- `GET /api/v1/conversations/{id}/messages` - List messages in conversation
  - Supports pagination with `?limit=` and `?offset=`
  - Returns messages with sentences, tool uses, and reasoning steps

- `POST /api/v1/conversations/{id}/messages` - Send user message
  - Request body: `{"content": "user message text", "enable_tools": true}`
  - Triggers LLM response generation
  - Returns assistant message
  - Broadcasts to SSE/WebSocket subscribers

- `GET /api/v1/messages/{id}/siblings` - Get sibling messages (alternate responses)
- `POST /api/v1/conversations/{id}/switch-branch` - Switch to a different message branch

### Sync Protocol (MessagePack)

- `POST /api/v1/conversations/{id}/sync` - HTTP-based message sync
  - Accepts MessagePack binary payload
  - Used for bulk message synchronization

- `GET /api/v1/conversations/{id}/sync/status` - Get sync status
  - Returns last sync timestamp and pending message count

- `GET /api/v1/ws` - Multiplexed WebSocket endpoint
  - Single connection handles multiple conversations
  - Subscribe/unsubscribe to conversations dynamically
  - Message types: Subscribe (40), Unsubscribe (41), SubscribeAck (42), UnsubscribeAck (43)
  - Binary MessagePack protocol
  - Used by web and mobile clients

### Real-Time Events (SSE)

- `GET /api/v1/conversations/{id}/events` - Server-Sent Events stream
  - Long-lived connection for real-time updates
  - Event types: `message.created`, `sentence.streamed`, `tool.executed`
  - Used by web frontend for live updates

### LiveKit Integration

- `POST /api/v1/conversations/{id}/token` - Generate LiveKit access token
  - Creates JWT token for joining conversation room
  - Room name format: `conv_{conversation_id}`
  - Token validity: 6 hours (configurable)

### MCP (Model Context Protocol)

Only available if MCP adapter is configured.

- `GET /api/v1/mcp/servers` - List connected MCP servers
- `POST /api/v1/mcp/servers` - Add new MCP server
  - Request body: `{"name": "server-name", "type": "sse|stdio", "url": "..."}`
- `DELETE /api/v1/mcp/servers/{name}` - Remove MCP server
- `GET /api/v1/mcp/tools` - List all tools from all MCP servers

### Memory & Feedback Voting

Fine-grained feedback system for prompt optimization:

**Message Voting**:
- `POST /api/v1/messages/{id}/vote` - Vote on message quality
- `DELETE /api/v1/messages/{id}/vote` - Remove vote
- `GET /api/v1/messages/{id}/votes` - Get vote statistics

**Tool Use Voting**:
- `POST /api/v1/tool-uses/{id}/vote` - Vote on tool execution
- `POST /api/v1/tool-uses/{id}/quick-feedback` - Quick thumbs up/down
- `DELETE /api/v1/tool-uses/{id}/vote` - Remove vote
- `GET /api/v1/tool-uses/{id}/votes` - Get vote statistics

**Memory Voting**:
- `POST /api/v1/memories/{id}/vote` - Vote on memory relevance
- `POST /api/v1/memories/{id}/irrelevance-reason` - Report irrelevant memory
- `DELETE /api/v1/memories/{id}/vote` - Remove vote

**Memory Usage Voting**:
- `POST /api/v1/memory-usages/{id}/vote` - Vote on memory usage relevance
- `DELETE /api/v1/memory-usages/{id}/vote` - Remove vote from memory usage
- `GET /api/v1/memory-usages/{id}/votes` - Get memory usage vote statistics
- `POST /api/v1/memory-usages/{id}/irrelevance-reason` - Report why memory was irrelevant

**Memory Extraction Voting**:
- `POST /api/v1/messages/{messageId}/extracted-memories/{memoryId}/vote` - Vote on extraction quality
- `DELETE /api/v1/messages/{messageId}/extracted-memories/{memoryId}/vote` - Remove extraction vote
- `GET /api/v1/messages/{messageId}/extracted-memories/{memoryId}/votes` - Get extraction vote stats
- `POST /api/v1/messages/{messageId}/extracted-memories/{memoryId}/quality-feedback` - Quality feedback

**Reasoning Voting**:
- `POST /api/v1/reasoning/{id}/vote` - Vote on reasoning quality
- `POST /api/v1/reasoning/{id}/issue` - Report reasoning issue
- `DELETE /api/v1/reasoning/{id}/vote` - Remove vote

### Notes

Contextual notes for debugging and analysis:

- `POST /api/v1/messages/{id}/notes` - Add note to message
- `GET /api/v1/messages/{id}/notes` - Get message notes
- `POST /api/v1/tool-uses/{id}/notes` - Add note to tool use
- `POST /api/v1/reasoning/{id}/notes` - Add note to reasoning step
- `PUT /api/v1/notes/{id}` - Update note
- `DELETE /api/v1/notes/{id}` - Delete note

### Memory Management

Only available if embedding service is configured.

- `POST /api/v1/memories` - Create memory
- `GET /api/v1/memories` - List all memories
- `POST /api/v1/memories/search` - Vector similarity search
- `GET /api/v1/memories/by-tags` - Get memories by tags
- `GET /api/v1/memories/{id}` - Get memory by ID
- `PUT /api/v1/memories/{id}` - Update memory
- `DELETE /api/v1/memories/{id}` - Delete memory
- `POST /api/v1/memories/{id}/tags` - Add tag to memory
- `DELETE /api/v1/memories/{id}/tags/{tag}` - Remove tag
- `PUT /api/v1/memories/{id}/importance` - Set importance level
- `POST /api/v1/memories/{id}/pin` - Pin/unpin memory
- `POST /api/v1/memories/{id}/archive` - Archive/unarchive memory

### Prompt Optimization

DSPy/GEPA optimization endpoints (only if optimization service is configured):

- `POST /api/v1/optimizations` - Create optimization run
- `GET /api/v1/optimizations` - List optimization runs
- `GET /api/v1/optimizations/{id}` - Get run details
- `GET /api/v1/optimizations/{id}/candidates` - List prompt candidates
- `GET /api/v1/optimizations/{id}/best` - Get best candidate
- `GET /api/v1/optimizations/{id}/program` - Get optimized program
- `GET /api/v1/optimizations/{id}/stream` - SSE stream of optimization progress
- `GET /api/v1/optimizations/candidates/{id}/evaluations` - Get candidate evaluations

**Feedback Integration**:
- `POST /api/v1/feedback` - Submit feedback for optimization
- `GET /api/v1/feedback/dimensions` - Get dimension weights
- `PUT /api/v1/feedback/dimensions` - Update dimension weights

**Deployment**:
- `POST /api/v1/deployments` - Deploy optimized prompt
- `GET /api/v1/deployments/{prompt_type}/active` - Get active deployment
- `GET /api/v1/deployments/{prompt_type}/history` - List deployment history
- `DELETE /api/v1/deployments/{run_id}` - Rollback deployment

### Training

Training and prompt version management endpoints for GEPA optimization.

#### Training Operations
- `GET /api/v1/training/stats` - Get training statistics
- `POST /api/v1/training/optimize` - Trigger optimization run

#### Prompt Version Management
- `GET /api/v1/prompts/versions` - List all prompt versions
- `POST /api/v1/prompts/versions/{id}/activate` - Activate a specific prompt version

### Server Info & Statistics

- `GET /api/v1/server/info` - Server version, capabilities, MCP status
- `GET /api/v1/server/stats` - Global statistics (total conversations, messages, tool uses)
- `GET /api/v1/conversations/{id}/stats` - Session-specific statistics

### Text-to-Speech (OpenAI-compatible)

No authentication required (for voice preview feature):

- `POST /v1/audio/speech` - Generate speech from text
  - Request: `{"model": "tts-1", "voice": "alloy", "input": "text"}`
  - Response: Audio bytes (MP3 or Opus)

## Service Layer Architecture

**Location**: `/internal/application/services/`

Key services initialized during server startup:

- **ToolService**: Manages tool registration, listing, and execution
- **MemoryService**: Handles memory creation, search, and retrieval (requires embedding)
- **OptimizationService**: DSPy/GEPA prompt optimization orchestration
- **DeploymentService**: Prompt deployment and rollback management

## Configuration Options

All configuration via `ALICIA_` environment variables:

### Server
- `ALICIA_SERVER_HOST` - Bind address (default: "0.0.0.0")
- `ALICIA_SERVER_PORT` - HTTP port (default: 8080)
- `ALICIA_CORS_ORIGINS` - Allowed CORS origins (comma-separated)
- `ALICIA_STATIC_DIR` - Static file directory for frontend serving

### Database
- `ALICIA_POSTGRES_URL` - PostgreSQL connection string (required)
- `ALICIA_DB_PATH` - SQLite database path (CLI mode)

### LLM
- `ALICIA_LLM_URL` - LLM endpoint (required)
- `ALICIA_LLM_API_KEY` - API key (required)
- `ALICIA_LLM_MODEL` - Model name
- `ALICIA_LLM_MAX_TOKENS` - Max completion tokens
- `ALICIA_LLM_TEMPERATURE` - Sampling temperature

### Reflection LLM (optional)
- `ALICIA_REFLECTION_LLM_URL` - Stronger LLM for GEPA reflection
- `ALICIA_REFLECTION_LLM_API_KEY` - API key
- `ALICIA_REFLECTION_LLM_MODEL` - Model name
- `ALICIA_REFLECTION_LLM_MAX_TOKENS` - Max completion tokens
- `ALICIA_REFLECTION_LLM_TEMPERATURE` - Sampling temperature

### LiveKit (optional)
- `ALICIA_LIVEKIT_URL` - WebSocket URL
- `ALICIA_LIVEKIT_API_KEY` - API key
- `ALICIA_LIVEKIT_API_SECRET` - API secret
- `ALICIA_LIVEKIT_WORKER_COUNT` - Number of worker goroutines (default: 10)
- `ALICIA_LIVEKIT_WORK_QUEUE_SIZE` - Work queue buffer size (default: 100)

### ASR/TTS (optional)
- `ALICIA_ASR_URL` - ASR service endpoint
- `ALICIA_ASR_MODEL` - Model name
- `ALICIA_TTS_URL` - TTS service endpoint
- `ALICIA_TTS_MODEL` - Model name
- `ALICIA_TTS_VOICE` - Voice name

### Embedding (optional, for memory)
- `ALICIA_EMBEDDING_URL` - Embedding endpoint
- `ALICIA_EMBEDDING_API_KEY` - API key
- `ALICIA_EMBEDDING_MODEL` - Model name
- `ALICIA_EMBEDDING_DIMENSIONS` - Vector dimensions

## Key Source Files

- **Server Entry**: `/cmd/alicia/serve.go`
- **HTTP Server**: `/internal/adapters/http/server.go`
- **Handlers**: `/internal/adapters/http/handlers/`
- **Middleware**: `/internal/adapters/http/middleware/`
- **Services**: `/internal/application/services/`
- **Use Cases**: `/internal/application/usecases/`
- **Repositories**: `/internal/adapters/postgres/`
- **Config**: `/internal/config/config.go`

## See Also

- [CLI.md](CLI.md) - Command-line interface for server management
- [AGENT.md](AGENT.md) - LiveKit agent worker architecture
- [COMPONENTS.md](COMPONENTS.md) - System architecture and component documentation
- [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment guides and configuration
