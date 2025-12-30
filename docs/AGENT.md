# LiveKit Agent Worker

The Alicia agent worker handles real-time voice conversations by connecting to LiveKit rooms and managing the complete voice interaction pipeline from audio input to synthesized speech output.

## Overview

The agent worker is a long-running process that monitors LiveKit for active conversation rooms, dispatches agent instances to handle voice interactions, and coordinates the flow between speech recognition, language model processing, tool execution, and speech synthesis.

**Entry Point**: `/cmd/alicia/agent.go`

## Architecture

### Worker Lifecycle

The worker operates in a continuous polling loop:

1. **Startup**: Connect to PostgreSQL, initialize services and repositories
2. **Polling**: Every 5 seconds, query LiveKit for active rooms matching `conv_` prefix
3. **Dispatch**: For each room with participants but no assigned agent, create and dispatch an agent instance
4. **Monitoring**: Track agent health and room participant count
5. **Cleanup**: Disconnect agents when rooms close or all users leave
6. **Shutdown**: Graceful shutdown on SIGTERM/SIGINT

**Source**: `/internal/adapters/livekit/worker.go`

### Agent Instance

Each agent instance represents a single conversation session:

- One agent per LiveKit room
- Maintains WebRTC connection for audio streaming
- Handles MessagePack data channel for protocol messages
- Manages voice pipeline state machine
- Coordinates between ASR, LLM, TTS, and tool execution

**Source**: `/internal/adapters/livekit/agent.go`

## Room Connection Flow

### Discovery

```
Worker polls LiveKit every 5s
  ↓
Lists all active rooms
  ↓
Filters for rooms starting with "conv_"
  ↓
Checks if room has participants
  ↓
Verifies no agent already assigned
  ↓
Dispatches new agent instance
```

### Connection

```
Agent creates access token (24h validity)
  ↓
Establishes WebRTC connection to room
  ↓
Subscribes to participant audio tracks
  ↓
Publishes local audio track for responses
  ↓
Opens data channel for MessagePack protocol
  ↓
Enters active state, ready for voice input
```

### Room Naming

Rooms follow the convention: `conv_{conversation_id}`

Example: `conv_abc123xyz` maps to conversation ID `abc123xyz`

## Message Handling Pipeline

**Source**: `/internal/adapters/livekit/message_router.go`

### MessageRouter

Routes incoming protocol messages to appropriate handlers:

- **User Messages**: Routes to ProcessUserMessage use case
- **Tool Acknowledgements**: Updates tool execution status
- **Metadata Updates**: Syncs conversation state
- **Control Messages**: Handles pause/resume/stop commands

### MessageDispatcher

Sends outgoing messages to clients via data channel:

- Serializes domain messages to MessagePack binary format
- Manages acknowledgement tracking with stanza IDs
- Implements retry logic for unacknowledged messages
- Handles message ordering and reliability

## Voice Pipeline

**Source**: `/internal/adapters/livekit/voice_pipeline.go`

### Input Pipeline: Speech → Text

```
User speaks into microphone
  ↓
WebRTC audio track (Opus encoded)
  ↓
Opus decoder → PCM audio buffer
  ↓
Silence detection (VAD - Voice Activity Detection)
  ↓
Accumulate audio segments until silence
  ↓
Send accumulated audio to ASR service
  ↓
Receive transcription text
  ↓
Create UserMessage in database
  ↓
Trigger response generation
```

**Key Components**:
- Voice Activity Detection (VAD) for natural turn-taking
- Configurable silence threshold and duration
- Chunked audio processing for efficiency
- Automatic punctuation and formatting via ASR

### Output Pipeline: Text → Speech

```
LLM generates response text (streaming)
  ↓
Sentence boundary detection
  ↓
Send complete sentence to TTS service
  ↓
Receive audio bytes (Opus or MP3)
  ↓
Opus encoder (if needed)
  ↓
Publish to agent's audio track
  ↓
User hears response
```

**Key Components**:
- Streaming sentence-by-sentence for low latency
- Parallel TTS requests for multiple sentences
- Audio buffering and synchronization
- Configurable voice and speaking rate

## Response Generation Flow

**Source**: `/internal/application/usecases/generate_response.go`

### Complete Pipeline

```
User message created
  ↓
Retrieve conversation history (last 20 messages)
  ↓
Search for relevant memories (if enabled)
  ↓
Build LLM context with history + memories
  ↓
Fetch enabled tools from database
  ↓
Stream LLM response with tool calling support
  ↓
For each tool call:
  ├─ Execute tool via ToolService
  ├─ Store ToolUse in database
  ├─ Append tool result to context
  └─ Continue LLM generation
  ↓
For each streamed sentence:
  ├─ Store Sentence in database
  ├─ Send to TTS for synthesis
  └─ Broadcast via MessagePack
  ↓
Complete AssistantMessage saved
  ↓
Notify clients via SSE/WebSocket
```

### Tool Execution

When the LLM requests a tool:

1. **Tool Call Detection**: Parse tool name and arguments from LLM response
2. **Tool Lookup**: Find tool definition in database
3. **Validation**: Verify tool is enabled and arguments match schema
4. **Execution**: Call tool executor (built-in or MCP)
5. **Result Capture**: Store execution result, status, and metadata
6. **Context Update**: Append tool result to conversation context
7. **Continuation**: Resume LLM generation with tool result

**Built-in Tools**:
- `calculator`: Mathematical expression evaluation
- `web_search`: Web search via external API
- `memory_query`: Semantic search over conversation memory

**MCP Tools**: Dynamically loaded from connected MCP servers

## Voice Activity Detection (VAD)

The agent uses energy-based VAD to detect when users are speaking:

- **Energy Threshold**: Minimum audio energy to consider as speech
- **Silence Duration**: Time of silence before ending utterance (default: 1.5s)
- **Minimum Duration**: Minimum speech duration to process (default: 0.3s)
- **Debouncing**: Prevents false positives from background noise

Configuration tuning affects responsiveness vs accuracy:
- Lower threshold = more sensitive, may capture noise
- Longer silence = more complete sentences, higher latency
- Shorter silence = faster response, may cut off sentences

## Agent Configuration

### Worker Pool

Each agent uses a worker pool for concurrent event processing:

- **WorkerCount**: Number of goroutines per agent (default: 10)
- **WorkQueueSize**: Buffered queue size (default: 100)
- Prevents blocking on slow operations
- Enables parallel tool execution and TTS synthesis

### Token Validity

Agent access tokens have configurable validity:

- Default: 24 hours
- Determines max session duration
- After expiry, agent must reconnect with new token

### Room Prefix

Worker only handles rooms matching prefix:

- Default: `conv_`
- Allows multiple workers with different prefixes
- Enables A/B testing and staged rollouts

## Key Source Files

- **Agent Command**: `/cmd/alicia/agent.go`
- **Worker**: `/internal/adapters/livekit/worker.go`
- **Agent Instance**: `/internal/adapters/livekit/agent.go`
- **Voice Pipeline**: `/internal/adapters/livekit/voice_pipeline.go`
- **Message Router**: `/internal/adapters/livekit/message_router.go`
- **Message Dispatcher**: `/internal/adapters/livekit/message_dispatcher.go`
- **Generate Response Use Case**: `/internal/application/usecases/generate_response.go`
- **Process User Message Use Case**: `/internal/application/usecases/process_user_message.go`
- **Handle Tool Call Use Case**: `/internal/application/usecases/handle_tool_call.go`
- **Agent Factory**: `/internal/adapters/livekit/agent_factory.go`
- **Codec**: `/internal/adapters/livekit/codec.go`
- **Audio Converter**: `/internal/adapters/livekit/audio_converter.go`

## Monitoring & Debugging

### Logging

The agent logs key events:

- Room discovery and dispatch
- Agent connection and disconnection
- Audio processing pipeline events
- ASR transcription results
- LLM response streaming
- Tool execution results
- TTS synthesis completion
- Error conditions and retries

### Health Checks

Worker performs periodic health checks:

- Every 10 seconds, verify agent still connected
- Check if room still exists
- Verify room has non-agent participants
- Auto-cleanup if conditions not met

### Metrics

Key metrics to monitor (via Prometheus):

- Active agent count
- Room join/leave rate
- ASR transcription latency
- LLM response latency
- TTS synthesis latency
- Tool execution success rate
- Audio packet loss rate

## Error Handling

The agent implements robust error handling:

- **Connection Failures**: Retry with exponential backoff
- **ASR Errors**: Log and notify user, continue listening
- **LLM Errors**: Return error message to user
- **TTS Errors**: Fall back to text-only response
- **Tool Errors**: Capture error, report to LLM for recovery

## See Also

- [CLI.md](CLI.md) - Running the agent worker
- [SERVER.md](SERVER.md) - HTTP API for client interaction
- [ANDROID.md](ANDROID.md) - Mobile client integration
- [PROTOCOL.md](PROTOCOL.md) - MessagePack protocol specification
