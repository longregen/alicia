# Command-Line Interface (CLI)

The Alicia CLI provides text-based interaction with the AI assistant, along with server management, agent worker control, and prompt optimization tools.

## Overview

The CLI is the primary entry point for running Alicia services and managing conversations. It's built with Go using the Cobra framework for command structure and configuration via environment variables.

**Entry Point**: `/cmd/alicia/main.go`

## Available Commands

### `alicia new`

Create a new conversation.

**Usage**:
```bash
# Create with auto-generated title
alicia new

# Create with custom title
alicia new --title "My Conversation"
alicia new -t "Planning Session"
```

**Source**: `/cmd/alicia/conversation.go`

**Example**:
```bash
$ alicia new --title "Project Planning"
Created conversation: conv_abc123xyz
Title: Project Planning
```

### `alicia list`

List all conversations.

**Usage**:
```bash
# List active conversations (default: 50)
alicia list

# Include archived conversations
alicia list --all
alicia list -a

# Limit number of results
alicia list --limit 20
alicia list -l 20
```

**Source**: `/cmd/alicia/conversation.go`

**Example**:
```bash
$ alicia list
ID                            Title                                    Status     Created
----------------------------------------------------------------------------------------------------
conv_abc123xyz                Project Planning                         active     2025-12-30 14:32
conv_def456uvw                Debug Session                            active     2025-12-29 09:15
conv_ghi789rst                Code Review                              active     2025-12-28 16:45
```

### `alicia show`

Display all messages in a conversation.

**Usage**:
```bash
alicia show <conversation-id>
```

**Source**: `/cmd/alicia/conversation.go`

**Example**:
```bash
$ alicia show conv_abc123xyz
Conversation: Project Planning
ID: conv_abc123xyz
Status: active
Created: 2025-12-30 14:32:15

[14:32:18] user:
What's the best way to structure this project?
--------------------------------------------------------------------------------

[14:32:22] assistant:
I'd recommend organizing it into these modules...
--------------------------------------------------------------------------------
```

### `alicia delete`

Delete a conversation (soft delete).

**Usage**:
```bash
alicia delete <conversation-id>
```

**Source**: `/cmd/alicia/conversation.go`

**Example**:
```bash
$ alicia delete conv_abc123xyz
Deleted conversation: Project Planning
ID: conv_abc123xyz
```

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
- Registers built-in tools (calculator, web search always available; memory query available when embedding service is configured)
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

**Optional Configuration (for voice functionality)**:
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

ASR (Speech Recognition):
  URL:     http://localhost:5000
  Model:   whisper-1
  API Key: ***
  Status:  Optional (for voice functionality)

TTS (Text-to-Speech):
  URL:     http://localhost:5001
  Model:   tts-1
  Voice:   alloy
  API Key: ***
  Status:  Optional (for voice functionality)

Database:
  PostgreSQL:    postgresql://...***
  SQLite Path:   /home/user/.local/share/alicia/alicia.db (not used by CLI commands)

Environment variables:
  ALICIA_LLM_URL, ALICIA_LLM_API_KEY, ALICIA_LLM_MODEL
  ALICIA_LIVEKIT_URL, ALICIA_LIVEKIT_API_KEY, ALICIA_LIVEKIT_API_SECRET
  ALICIA_ASR_URL, ALICIA_ASR_API_KEY, ALICIA_ASR_MODEL
  ALICIA_TTS_URL, ALICIA_TTS_API_KEY, ALICIA_TTS_MODEL, ALICIA_TTS_VOICE
  ALICIA_DB_PATH, ALICIA_POSTGRES_URL
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

**Note**: Version displays "dev" unless built with ldflags (e.g., `-ldflags "-X main.version=v0.3.0"`)

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
- `ALICIA_POSTGRES_URL` - PostgreSQL connection string (required for all CLI commands: `serve`, `agent`, `chat`, `new`, `list`, `show`, `delete`)
- `ALICIA_DB_PATH` - SQLite database path (legacy/reserved for future use; not currently used by CLI commands)

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
