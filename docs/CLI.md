# Command-Line Interface (CLI)

The Alicia CLI provides text-based interaction with the AI assistant, along with server management, agent worker control, and prompt optimization tools.

## Overview

The CLI is the primary entry point for running Alicia services and managing conversations. It's built with Go using the Cobra framework for command structure and configuration via environment variables.

**Entry Point**: `/cmd/alicia/main.go`

## Available Commands

### `alicia chat`

Interactive text-based chat with Alicia.

**Usage**:
```bash
# Start a new conversation
alicia chat --title "My Conversation"

# Continue an existing conversation
alicia chat <conversation-id>
```

**Features**:
- Real-time streaming responses from LLM
- Conversation history persistence to database
- Automatic conversation creation
- Type 'exit' or 'quit' to end session

**Source**: `/cmd/alicia/chat.go`

**Example**:
```bash
$ alicia chat
Started new conversation: Chat 2025-12-30 14:32
ID: conv_abc123xyz

Type your message and press Enter. Type 'exit' or 'quit' to end the conversation.
--------------------------------------------------------------------------------

You: What's the weather like?
Alicia: I don't have access to real-time weather data, but I can help you find...
```

### `alicia serve`

Start the HTTP API server for REST endpoints and real-time communication.

**Usage**:
```bash
alicia serve
```

**What it does**:
- Initializes PostgreSQL connection pool
- Sets up all repositories (conversations, messages, tools, memory, etc.)
- Configures LLM, ASR, TTS, and embedding services
- Registers built-in tools (calculator, web search, memory query)
- Starts HTTP server with REST API and WebSocket endpoints
- Enables LiveKit token generation for voice clients

**Required Configuration**:
- `ALICIA_POSTGRES_URL` - PostgreSQL connection string
- `ALICIA_LLM_URL` - LLM endpoint URL
- `ALICIA_LLM_API_KEY` - LLM API key

**Optional Configuration**:
- `ALICIA_LIVEKIT_*` - For voice features
- `ALICIA_ASR_*` / `ALICIA_TTS_*` - For speech services
- `ALICIA_EMBEDDING_*` - For memory/RAG features

**Source**: `/cmd/alicia/serve.go`

**Example**:
```bash
$ alicia serve
Starting Alicia API server...
  HTTP:     http://localhost:8080
  LLM:      https://api.openai.com/v1
  LiveKit:  ws://localhost:7880

Database connection established
LLM service initialized
Tool service initialized
Built-in tools registered
HTTP server listening on localhost:8080
```

### `alicia agent`

Start the LiveKit agent worker for handling voice conversations.

**Usage**:
```bash
alicia agent
```

**What it does**:
- Connects to LiveKit server
- Polls for conversation rooms with `conv_` prefix every 5 seconds
- Dispatches agent instances to rooms with active participants
- Manages voice pipeline (audio input → ASR → LLM → TTS → audio output)
- Handles MessagePack protocol for data channel communication
- Automatically cleans up agents when rooms close

**Required Configuration**:
- `ALICIA_POSTGRES_URL` - PostgreSQL connection string
- `ALICIA_LIVEKIT_URL` - LiveKit server WebSocket URL
- `ALICIA_LIVEKIT_API_KEY` - LiveKit API key
- `ALICIA_LIVEKIT_API_SECRET` - LiveKit API secret
- `ALICIA_ASR_URL` - ASR service endpoint
- `ALICIA_TTS_URL` - TTS service endpoint

**Source**: `/cmd/alicia/agent.go`

**Example**:
```bash
$ alicia agent
Starting Alicia agent worker...
  LiveKit:  ws://localhost:7880
  LLM:      https://api.openai.com/v1
  ASR:      http://localhost:5000
  TTS:      http://localhost:5001

Database connection established
Agent factory initialized
Agent worker started
Dispatching agent to room: conv_abc123 (participants: 2)
Agent connected to room: conv_abc123 (conversation: abc123)
```

### `alicia optimize`

Manage DSPy/GEPA prompt optimization runs.

**Usage**:
```bash
# List optimization runs
alicia optimize list [--status running|completed|failed] [--limit 20]

# Show run details
alicia optimize show <run-id> [--json]

# Start new optimization run
alicia optimize run --name "My Optimization" --type "assistant_prompt" [--baseline "..."]

# List candidates for a run
alicia optimize candidates <run-id> [--json]

# Show best candidate
alicia optimize best <run-id> [--prompt]
```

**Source**: `/cmd/alicia/optimize.go`

**Example**:
```bash
$ alicia optimize list
ID        NAME              STATUS     ITERATIONS  BEST SCORE  STARTED          COMPLETED
--        ----              ------     ----------  ----------  -------          ---------
a1b2c3d4  Response Quality  completed  100/100     0.8532      2025-12-29 10:15 2025-12-29 12:43
e5f6g7h8  Tool Selection    running    47/100      0.7891      2025-12-30 08:00 N/A
```

### `alicia config`

Display current configuration from environment variables.

**Usage**:
```bash
alicia config
```

**Shows**:
- LLM configuration (URL, model, max tokens, temperature)
- LiveKit configuration and status
- ASR configuration and status
- TTS configuration and status
- Database paths
- List of environment variables

**Example**:
```bash
$ alicia config
Current configuration:

LLM:
  URL:         https://api.openai.com/v1
  Model:       gpt-4
  Max Tokens:  4096
  Temperature: 0.70
  API Key:     sk-...***

LiveKit:
  URL:        ws://localhost:7880
  API Key:    API...***
  API Secret: ***
  Status:     Configured
```

### `alicia version`

Show version information.

**Usage**:
```bash
alicia version
```

**Example**:
```bash
$ alicia version
Alicia v0.3.0
  Commit:     a1b2c3d4
  Build Date: 2025-12-30T10:00:00Z
```

## Configuration

All configuration is done via environment variables with `ALICIA_` prefix:

### LLM Configuration
- `ALICIA_LLM_URL` - LLM API endpoint (default: OpenAI-compatible)
- `ALICIA_LLM_API_KEY` - API authentication key
- `ALICIA_LLM_MODEL` - Model name (e.g., "gpt-4", "claude-3-opus")
- `ALICIA_LLM_MAX_TOKENS` - Maximum response tokens
- `ALICIA_LLM_TEMPERATURE` - Sampling temperature (0.0-2.0)

### LiveKit Configuration
- `ALICIA_LIVEKIT_URL` - LiveKit WebSocket URL
- `ALICIA_LIVEKIT_API_KEY` - LiveKit API key
- `ALICIA_LIVEKIT_API_SECRET` - LiveKit API secret

### Speech Services
- `ALICIA_ASR_URL` - Automatic Speech Recognition endpoint
- `ALICIA_ASR_API_KEY` - ASR API key (if required)
- `ALICIA_ASR_MODEL` - ASR model name (default: "whisper-1")
- `ALICIA_TTS_URL` - Text-to-Speech endpoint
- `ALICIA_TTS_API_KEY` - TTS API key (if required)
- `ALICIA_TTS_MODEL` - TTS model name (default: "tts-1")
- `ALICIA_TTS_VOICE` - Voice name (default: "alloy")

### Database
- `ALICIA_POSTGRES_URL` - PostgreSQL connection string
- `ALICIA_DB_PATH` - SQLite database path (for local CLI chat)

### Embedding (for Memory/RAG)
- `ALICIA_EMBEDDING_URL` - Embedding service endpoint
- `ALICIA_EMBEDDING_API_KEY` - Embedding API key
- `ALICIA_EMBEDDING_MODEL` - Model name (default: "text-embedding-ada-002")
- `ALICIA_EMBEDDING_DIMENSIONS` - Vector dimensions

## Key Source Files

- **Main Entry**: `/cmd/alicia/main.go`
- **Serve Command**: `/cmd/alicia/serve.go`
- **Agent Command**: `/cmd/alicia/agent.go`
- **Chat Command**: `/cmd/alicia/chat.go`
- **Optimize Commands**: `/cmd/alicia/optimize.go`
- **Config Loading**: `/internal/config/config.go`
- **LLM Client**: `/internal/llm/client.go`

## See Also

- [SERVER.md](SERVER.md) - HTTP API endpoints and architecture
- [AGENT.md](AGENT.md) - LiveKit agent worker details
- [CONFIGURATION.md](CONFIGURATION.md) - Detailed configuration guide
