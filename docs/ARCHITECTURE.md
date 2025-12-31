# Alicia Architecture

This document describes the architecture of Alicia, a real-time voice assistant that enables natural conversations through audio. Alicia uses LiveKit as its real-time communication layer to provide seamless, streaming conversations with AI across web, mobile, and command-line interfaces.

## System Overview

Alicia is a multi-platform voice assistant that provides real-time, streaming conversations with AI. The system combines speech recognition (Whisper), language understanding (Qwen2.5-7B-Instruct-AWQ), and voice synthesis (Kokoro) to create a seamless conversational experience. LiveKit serves as the central communication hub, handling all real-time audio streaming and protocol message delivery.

```mermaid
graph TD
    subgraph "Client Layer"
        WebClient[Web Client<br/>React/TypeScript]
        MobileClient[Mobile Client<br/>Android/Kotlin]
        CLIClient[CLI Client<br/>Go/Cobra]
    end

    subgraph "LiveKit Server"
        Room[Room: conv_123<br/>Audio Tracks + Data Channels]
    end

    subgraph "Alicia Agent"
        Agent[LiveKit Agent Process]
        Whisper[Whisper ASR]
        LLM[Qwen2.5-7B-Instruct-AWQ LLM]
        TTS[Kokoro TTS]
    end

    subgraph "Data Layer"
        DB[(PostgreSQL<br/>+ pgvector)]
    end

    WebClient -->|LiveKit SDK| Room
    MobileClient -->|LiveKit SDK| Room
    CLIClient -->|LiveKit SDK| Room

    Room <-->|Audio + Data| Agent

    Agent --> Whisper
    Agent --> LLM
    Agent --> TTS
    Agent --> DB

    style Room fill:#e1f5ff
    style Agent fill:#fff4e1
```

### Key Architecture Principles

1. **LiveKit as Communication Backbone**: All real-time communication flows through LiveKit, providing battle-tested media transport, NAT traversal, and multi-platform support.

2. **One Room Per Conversation**: Each conversation maps to exactly one LiveKit room (`conv_{conversation_id}`), providing natural isolation and lifecycle management.

3. **Dual Transport Channels**:
   - **Audio tracks** carry voice (user microphone → agent, agent TTS → user)
   - **Data channels** carry protocol messages (MessagePack-encoded)

4. **Agent as Participant**: The Alicia agent joins rooms as participant `alicia-agent`, processing audio and generating responses in real-time.

5. **Client Flexibility**: Clients can be voice-only, text-only, or multimodal, all using the same underlying infrastructure.

## Technology Stack

### Core Technologies

1. **Real-Time Communication**:
   - **LiveKit Server**: Central hub for WebRTC-based real-time audio and data transport
   - **LiveKit Client SDKs**: Web (TypeScript), Android (Kotlin), CLI (Go)
   - **LiveKit Agents Framework**: Framework for building AI agents

2. **Backend Language & Tools**:
   - Go 1.22+ (API services, CLI client, conversation management, agent worker)
   - golangci-lint (static code analysis)

3. **Database & Storage**:
   - PostgreSQL 15+ with pgvector extension (vector embeddings)
   - sqlc (type-safe database access for Go)
   - pgx (PostgreSQL driver)
   - migrate (database migrations)

4. **Protocol & Serialization**:
   - MessagePack (binary protocol implementation as specified in PROTOCOL.md)
   - Protocol messages flow over LiveKit data channels

5. **AI Models & Integration**:
   - Whisper (speech recognition via speaches server)
   - Qwen2.5-7B-Instruct-AWQ (language understanding via vLLM server)
   - Kokoro TTS (voice synthesis via speaches server)
   - LiteLLM (OpenAI-compatible API proxy)

6. **Observability**:
   - OpenTelemetry (distributed tracing)
   - Prometheus (metrics)
   - zap (structured logging)

### Audio Formats

- **Input/Output**: Opus codec at 48kHz (LiveKit negotiated)
- **Internal Processing**: 16kHz PCM for Whisper
- **TTS Output**: 24kHz mono for Kokoro (resampled to 48kHz for LiveKit)

## System Architecture

### High-Level Component View

```mermaid
graph TB
    subgraph "Client Applications"
        Web[Web Interface<br/>React - Full Voice]
        Mobile[Android App<br/>Kotlin/Compose]
        CLI[CLI Tool<br/>Go/Cobra]
    end

    subgraph "API Services Layer"
        API[API Service<br/>Go]
        Auth[Authentication<br/>Token Generation]
    end

    subgraph "LiveKit Infrastructure"
        LKServer[LiveKit Server<br/>Media Router]
        Rooms[Rooms<br/>conv_* namespaced]
    end

    subgraph "Agent Worker Layer"
        AgentPool[Agent Worker Pool<br/>Go]
        VAD[Voice Activity<br/>Detection]
        Pipeline[STT → LLM → TTS<br/>Pipeline]
    end

    subgraph "AI Components"
        ASR[Whisper ASR<br/>speaches]
        LLM[Qwen2.5-7B-Instruct-AWQ LLM<br/>vLLM]
        TTSEngine[Kokoro TTS<br/>speaches]
        LiteLLM[LiteLLM Proxy<br/>OpenAI API]
    end

    subgraph "Data Layer"
        PostgreSQL[(PostgreSQL)]
        Vectors[(pgvector<br/>Embeddings)]
    end

    Web --> API
    Mobile --> API
    CLI --> API

    API --> Auth
    Auth -->|Access Token| Web
    Auth -->|Access Token| Mobile
    Auth -->|Access Token| CLI

    Web -->|Connect| LKServer
    Mobile -->|Connect| LKServer
    CLI -->|Connect| LKServer

    LKServer --> Rooms
    Rooms -->|Dispatch| AgentPool

    AgentPool --> VAD
    AgentPool --> Pipeline

    Pipeline --> ASR
    Pipeline --> LLM
    Pipeline --> TTSEngine

    AgentPool --> PostgreSQL
    LLM --> Vectors

    style LKServer fill:#e1f5ff
    style AgentPool fill:#fff4e1
    style PostgreSQL fill:#e8f5e9
```

## LiveKit Communication Layer

### Room Model

Each Alicia conversation corresponds to exactly one LiveKit room. This provides:

- **Isolation**: Conversations cannot interfere with each other
- **Security**: Access controlled via per-room JWT tokens
- **Lifecycle Management**: Rooms automatically clean up when empty
- **Reconnection**: Clients can rejoin rooms with state preserved

**Room Naming Convention**:
```
Room Name: conv_{conversation_id}
Example:   conv_ac_k9mX2pL7qR
```

Where `conversation_id` is a random ID from the `alicia_conversations.id` column (generated using `generate_random_id('ac')`).

**Room Configuration**:
```go
room, err := livekitAPI.Room.CreateRoom(ctx, &livekit.CreateRoomRequest{
    Name:            fmt.Sprintf("conv_%s", conversationID),
    EmptyTimeout:    300,        // 5 min after last participant leaves
    MaxParticipants: 2,          // User + Alicia agent
    Metadata: string(mustMarshal(map[string]string{
        "conversation_id": conversationID,
        "created_at":      time.Now().UTC().Format(time.RFC3339),
    })),
})
```

### Participants

#### Client Participant

The user connecting from a frontend application.

- **Identity**: `user_{user_id}` (e.g., `user_u7k2m9p3`)
- **Permissions**: Can publish audio, subscribe to agent audio, publish/receive data messages
- **Tracks**: Publishes microphone audio (Opus, 48kHz)

Example client connection:
```typescript
const room = new Room();
await room.connect(livekitUrl, accessToken, {
  autoSubscribe: true,
});

// Publish microphone
const audioTrack = await createLocalAudioTrack({
  echoCancellation: true,
  noiseSuppression: true,
  autoGainControl: true,
});
await room.localParticipant.publishTrack(audioTrack);
```

#### Alicia Agent Participant

The AI assistant running as a LiveKit Agent worker.

- **Identity**: `alicia-agent` (with optional conversation-specific suffix)
- **Permissions**: Can publish TTS audio, subscribe to user audio, publish/receive data messages
- **Tracks**: Publishes synthesized speech audio (Opus, 48kHz)

The agent is automatically dispatched when a user joins a room.

### Transport Channels

#### Audio Tracks

| Track | Source | Direction | Format |
|-------|--------|-----------|--------|
| User Audio | Client Microphone | Client → Agent | Opus, 48kHz |
| Agent Audio | Kokoro TTS | Agent → Client | Opus, 48kHz |

LiveKit handles codec negotiation, adaptive bitrate, packet loss recovery, and jitter buffering automatically.

#### Data Channels

Alicia protocol messages flow over LiveKit data channels:

- **Encoding**: MessagePack (binary)
- **Reliability**: Uses LiveKit's RELIABLE data channel (guaranteed delivery, ordered)
- **Bidirectional**: Both client and agent can send messages

Example message flow:
```typescript
// Client sends UserMessage
const envelope = {
  stanzaId: nextStanzaId(),
  conversationId: conversationId,
  type: 2, // UserMessage
  body: {
    id: nanoid(),
    content: "What's the weather like?",
    previousId: lastMessageId,
  }
};

room.localParticipant.publishData(
  msgpack.encode(envelope),
  DataPacket_Kind.RELIABLE
);
```

```go
// Agent receives data message
room.OnDataReceived(func(packet *livekit.DataPacket) {
    var envelope Envelope
    msgpack.Unmarshal(packet.Data, &envelope)
    handleProtocolMessage(envelope)
})
```

### Access Control

Access to rooms is controlled via JWT tokens generated by the API service:

```go
func CreateRoomToken(conversationID, userID string, isAgent bool) (string, error) {
    token := auth.NewAccessToken(livekitAPIKey, livekitAPISecret)

    identity := fmt.Sprintf("user_%s", userID)
    if isAgent {
        identity = "alicia-agent"
    }

    token.SetIdentity(identity)
    token.AddGrant(&auth.VideoGrant{
        RoomJoin: true,
        Room:     fmt.Sprintf("conv_%s", conversationID),
        CanPublish: true,
        CanSubscribe: true,
        CanPublishData: true,
    })
    token.SetValidFor(6 * time.Hour)

    return token.ToJWT()
}
```

Tokens are scoped to a specific room and expire after 6 hours (configurable).

## Component Descriptions

### 1. Client Applications

Alicia supports multiple client implementations across different platforms.

#### Web Interface

- **Framework**: React with TypeScript
- **Features**:
  - Text message input and display
  - Conversation management (create, list, delete)
  - Message history
  - REST API integration
  - LiveKit real-time voice integration (`useLiveKit.ts`)
  - Voice activity detection and microphone input (`MicrophoneVAD.tsx`)
  - Real-time streaming via data channels
  - MessagePack protocol implementation
- **Deployment**: Static site (Vite build)

#### Mobile App

- **Platform**: Android (Kotlin/Jetpack Compose)
- **Features**:
  - Native audio capture and playback via LiveKit SDK
  - Porcupine wake word detection
  - Room database for local storage
  - Background voice service
  - Hilt dependency injection

#### CLI Tool

- **Language**: Go
- **Features**:
  - Interactive chat with streaming responses (`chat` command)
  - Conversation management (create, list, delete)
  - Text-only mode for terminal sessions
  - Database integration for persistence

### 2. API Service

A Go service that manages conversation lifecycle and authentication:

**Responsibilities**:
- Create new conversations in the database
- Generate LiveKit access tokens for clients and agents
- Handle REST API endpoints for conversation management
- Manage user authentication and authorization
- Provide conversation history queries

**Key Endpoints**:
- `POST /conversations` - Create new conversation, returns LiveKit token
- `GET /conversations/{id}` - Retrieve conversation metadata and message history
- `GET /conversations/{id}/token` - Generate new access token for existing conversation
- `DELETE /conversations/{id}` - End conversation and cleanup resources

**Implementation Notes**:
- Uses `pgx` for database access with `sqlc`-generated type-safe queries
- Implements OpenTelemetry tracing for request tracking
- Minimal latency focus for token generation (< 50ms target)

### 3. LiveKit Server

The central real-time communication hub:

**Responsibilities**:
- Route audio tracks between participants (SFU architecture)
- Manage data channel delivery
- Handle WebRTC negotiation (STUN/TURN)
- Enforce room access control
- Trigger agent dispatch via webhooks

**Deployment**:
- Self-hosted for privacy
- Configured with local TURN server for NAT traversal
- Redis for state management (optional, for multi-instance deployments)

**Configuration**:
```yaml
# livekit.yaml
port: 7880
rtc:
  port_range_start: 50000
  port_range_end: 60000
  use_external_ip: true
turn:
  enabled: true
  tls_port: 5349
keys:
  api_key: ${LIVEKIT_API_KEY}
  api_secret: ${LIVEKIT_API_SECRET}
webhook:
  urls:
    - http://alicia-api:8080/webhooks/livekit
```

### 4. Alicia Agent

The AI assistant implementation, running as a LiveKit Agent worker:

**Architecture** (Conceptual):

The agent processes voice conversations through a pipeline that integrates speech recognition, language understanding, and speech synthesis. The actual implementation is in `/internal/adapters/livekit/agent.go` and uses the Go ports/adapters pattern with services for ASR, LLM, and TTS.

**Voice Pipeline Flow**:

```mermaid
graph LR
    Audio[User Audio Track] --> VAD[Voice Activity<br/>Detection]
    VAD --> Buffer[Audio Buffer]
    Buffer --> Whisper[Whisper ASR]
    Whisper --> Trans[Transcription]
    Trans --> Qwen[Qwen2.5-7B-Instruct-AWQ LLM]
    Qwen --> Resp[Response Text]
    Resp --> Kokoro[Kokoro TTS]
    Kokoro --> AudioOut[Agent Audio Track]

    Trans -.->|Transcription msg| Data[Data Channel]
    Qwen -.->|StartAnswer,<br/>AssistantSentence| Data
    Qwen -.->|ToolUseRequest| Data

    style VAD fill:#e1f5ff
    style Whisper fill:#fff4e1
    style Qwen fill:#ffe1e1
    style Kokoro fill:#e1ffe1
```

**Component Integration**:

#### Whisper ASR (Speech-to-Text)

- **Implementation**: speaches server (OpenAI-compatible API)
- **Model**: `whisper-large-v3` or `whisper-medium` (configurable)
- **Output**: Streaming transcription with partial and final results
- **Protocol**: Sends `Transcription` messages (Type 9) via data channel
- **Service Interface**: Implements `ports.ASRService` defined in `/internal/ports/`

#### Qwen2.5-7B-Instruct-AWQ LLM (Language Understanding)

- **Implementation**: OpenAI-compatible API (vLLM or LiteLLM)
- **Model**: Configurable (commonly `Qwen2.5-7B-Instruct-AWQ` quantized for efficiency)
- **Context**: Full conversation history + memory context
- **Protocol**: Sends `StartAnswer` (Type 13) and `AssistantSentence` (Type 16) messages
- **Service Implementation**: See `/internal/llm/service.go` for the actual implementation

#### Kokoro TTS (Text-to-Speech)

- **Implementation**: speaches server (OpenAI-compatible API)
- **Voice**: Configurable (af_sarah, am_adam, etc.)
- **Output**: 24kHz stereo audio, resampled to 48kHz Opus for LiveKit
- **Streaming**: Sentence-by-sentence synthesis for low latency
- **Service Interface**: Implements `ports.TTSService` defined in `/internal/ports/`

**Protocol Message Handling**:

The agent processes incoming protocol messages through a message router that dispatches to appropriate handlers based on message type. See `/internal/adapters/livekit/message_router.go` for the actual implementation.

**Conversation State Management**:

The agent loads conversation history and memory context from PostgreSQL and persists all messages to the database. The implementation uses repository patterns defined in `/internal/adapters/postgres/` with type-safe queries generated by sqlc.

### 5. Database Layer

PostgreSQL serves as the persistent storage for all conversation data:

**Schema Overview** (Simplified):

The database schema includes tables for:
- **alicia_conversations**: Conversation metadata and state
- **alicia_messages**: User and assistant messages with linked-list structure
- **alicia_sentences**: Individual sentences within messages (for streaming)
- **alicia_audio**: Audio data and transcriptions
- **alicia_memory**: Memory entries with vector embeddings (1024 dimensions)
- **alicia_memory_used**: Tracking which memories were used in responses
- **alicia_tools**: Tool definitions and configurations
- **alicia_tool_uses**: Tool execution logs
- **alicia_reasoning_steps**: Chain-of-thought reasoning traces
- **alicia_votes**: User feedback on messages, tools, memories, and reasoning
- **alicia_notes**: User annotations and corrections
- **alicia_mcp_servers**: MCP (Model Context Protocol) server configurations
- **prompt_optimization_runs**, **prompt_candidates**, **prompt_evaluations**: GEPA optimization infrastructure

Key features:
- Vector similarity search using pgvector with `vector(1024)` embeddings
- Linked-list message structure via `previous_id` for branching support
- Comprehensive audit trails with `created_at`, `updated_at`, `deleted_at`
- JSONB columns for flexible metadata storage

See `/migrations/001_init.up.sql` for the complete schema definition.

**Access Patterns**:

- **Conversation Load**: Agent loads message history on room join
- **Message Persistence**: Every message stored immediately upon receipt/generation
- **Memory Retrieval**: Vector similarity search for relevant context
- **Tool Logging**: All tool requests and results logged for debugging

## Data Flow

### Voice Conversation Flow

```mermaid
sequenceDiagram
    participant Client
    participant LiveKit
    participant Agent
    participant Whisper
    participant Qwen
    participant Kokoro
    participant DB

    Client->>LiveKit: Publish Audio Track (user speaking)
    LiveKit->>Agent: Route Audio Stream

    Agent->>Whisper: Process Audio Chunks
    Whisper->>Agent: Transcription (streaming)
    Agent->>LiveKit: Data: Transcription (partial)
    LiveKit->>Client: Data: Transcription (partial)

    Note over Agent,Whisper: User finishes speaking

    Whisper->>Agent: Transcription (final)
    Agent->>LiveKit: Data: Transcription (final)
    LiveKit->>Client: Data: Transcription (final)

    Agent->>DB: Store UserMessage
    Agent->>Qwen: Generate Response (with context)

    Agent->>LiveKit: Data: StartAnswer
    LiveKit->>Client: Data: StartAnswer

    loop For each response chunk
        Qwen->>Agent: Response Chunk
        Agent->>LiveKit: Data: AssistantSentence
        LiveKit->>Client: Data: AssistantSentence
        Agent->>Kokoro: Synthesize Chunk
        Kokoro->>Agent: Audio PCM
        Agent->>LiveKit: Publish Audio Track (agent speaking)
        LiveKit->>Client: Route Audio Stream
    end

    Agent->>DB: Store AssistantMessage
```

### Tool Usage Flow

```mermaid
sequenceDiagram
    participant Client
    participant LiveKit
    participant Agent
    participant Qwen
    participant Tool
    participant DB

    Client->>LiveKit: Data: UserMessage
    LiveKit->>Agent: Data: UserMessage

    Agent->>Qwen: Process with Tools Available
    Qwen->>Agent: Tool Request (e.g., web_search)

    Agent->>LiveKit: Data: ToolUseRequest
    LiveKit->>Client: Data: ToolUseRequest (for transparency)

    Agent->>Tool: Execute Tool
    Tool->>Agent: Tool Result

    Agent->>LiveKit: Data: ToolUseResult
    LiveKit->>Client: Data: ToolUseResult

    Agent->>DB: Log Tool Usage

    Agent->>Qwen: Continue with Result
    Qwen->>Agent: Final Response

    Agent->>LiveKit: Data: StartAnswer + AssistantSentence
    LiveKit->>Client: Data: Messages
```

### Text-Only Mode

Clients can operate in text-only mode without audio:

```typescript
// Send text message
const envelope = {
  stanzaId: nextStanzaId(),
  conversationId: conversationId,
  type: 2, // UserMessage
  body: {
    id: nanoid(),
    content: "What's the weather?",
  }
};
room.localParticipant.publishData(msgpack.encode(envelope));

// Receive text responses
room.on("dataReceived", (payload) => {
  const envelope = msgpack.decode(payload);

  if (envelope.type === 16) {  // AssistantSentence
    displayText(envelope.body.text);
  }
});
```

The agent still generates TTS audio, but the client can choose not to subscribe to the audio track.

## Protocol Implementation

### MessagePack Encoding

All protocol messages use MessagePack for efficient binary serialization:

```go
// Example envelope structure (Go)
type Envelope struct {
    StanzaID       int32                  `msgpack:"stanzaId"`
    ConversationID string                 `msgpack:"conversationId"`
    Type           uint16                 `msgpack:"type"`
    Meta           map[string]interface{} `msgpack:"meta,omitempty"`
    Body           interface{}            `msgpack:"body"`
}

// Common meta keys for OpenTelemetry tracing
const (
    MetaKeyTraceID = "messaging.trace_id"
    MetaKeySpanID  = "messaging.span_id"
)

// Serialize
data, err := msgpack.Marshal(envelope)

// Send via LiveKit data channel
room.LocalParticipant.PublishData(data, lksdk.DataPacket_RELIABLE)
```

### Message Types

Core protocol message types flow over LiveKit data channels:

| Type | Name | Direction | Purpose |
|------|------|-----------|---------|
| 1 | ErrorMessage | Both | Error reporting |
| 2 | UserMessage | Client → Agent | User input |
| 3 | AssistantMessage | Agent → Client | Complete assistant response (non-streaming) |
| 4 | AudioChunk | Both | Audio metadata (audio flows on tracks) |
| 5 | ReasoningStep | Agent → Client | Chain-of-thought steps |
| 6 | ToolUseRequest | Agent → Client | Tool execution request |
| 7 | ToolUseResult | Both | Tool execution result |
| 8 | Acknowledgement | Both | Message receipt confirmation |
| 9 | Transcription | Agent → Client | Speech recognition results |
| 10 | ControlStop | Client → Agent | Stop current response |
| 11 | ControlVariation | Client → Agent | Request response variation |
| 12 | Configuration | Both | Capability negotiation |
| 13 | StartAnswer | Agent → Client | Begin streaming response |
| 14 | MemoryTrace | Agent → Client | Memory retrieval events |
| 15 | Commentary | Agent → Client | Meta-commentary |
| 16 | AssistantSentence | Agent → Client | Streaming response chunk |
| 20 | Feedback | Client → Agent | User votes on messages/tools/memories |
| 22 | UserNote | Client → Agent | User annotations and corrections |
| 24 | MemoryAction | Both | Memory CRUD operations |
| 26 | ServerInfo | Agent → Client | Server status and configuration |
| 29-31 | GEPA Messages | Both | Optimization dimension preferences and elite selection |

See [PROTOCOL.md](./protocol/index.md) for complete specifications and all message types.

### Audio Track vs AudioChunk Messages

With LiveKit, audio primarily flows over audio tracks (WebRTC), not protocol messages:

- **Audio Tracks**: Primary transport for voice (low-latency, adaptive bitrate)
- **AudioChunk Messages (Type 4)**: Optional metadata for synchronization, format info, or debugging

Example AudioChunk usage:
```go
// Send metadata about audio being streamed on track
envelope := Envelope{
    Type: 4, // AudioChunk
    Body: AudioChunkBody{
        Format:     "audio/opus",
        Sequence:   42,
        DurationMs: 500,
        TrackSID:   audioTrack.SID(), // Reference to LiveKit track
    },
}
publishData(envelope)
```

### Reconnection Handling

LiveKit provides automatic reconnection with state preservation:

```typescript
room.on(RoomEvent.Reconnecting, () => {
  console.log("Reconnecting...");
});

room.on(RoomEvent.Reconnected, () => {
  console.log("Reconnected");
  // Tracks and data channels automatically restored
});

room.on(RoomEvent.Disconnected, (reason) => {
  if (reason !== DisconnectReason.CLIENT_INITIATED) {
    // Offer to rejoin
    showRejoinDialog();
  }
});
```

The Alicia protocol's `Configuration` message with `lastSequenceSeen` provides additional message-level recovery:

```go
// Client sends Configuration on reconnect
envelope := Envelope{
    Type: 12, // Configuration
    Body: ConfigurationBody{
        LastSequenceSeen: 42, // Last stanzaId received
        Features:         []string{"streaming", "partial_responses"},
    },
}

// Agent resends any missed messages
missed := getMessagesAfterStanza(conversationID, 42)
for _, msg := range missed {
    publishData(msg)
}
```


## Deployment Architecture

Alicia is designed to run on a single consumer device with sufficient compute resources (GPU for AI models).

Deployment uses Nix for reproducible builds and environment management. See the Nix configuration for details.

## Observability

### Distributed Tracing

All components integrate with OpenTelemetry for end-to-end tracing:

```go
// Agent: Trace conversation flow
import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("alicia-agent")

func (a *Agent) handleUserMessage(ctx context.Context, envelope Envelope) {
    ctx, span := tracer.Start(ctx, "handle_user_message")
    defer span.End()

    span.SetAttributes(
        attribute.String("conversation_id", envelope.ConversationID),
        attribute.String("message_id", envelope.Body.(UserMessageBody).ID),
    )

    // Process message...
    a.processWithSpeaches(ctx)
    a.generateResponse(ctx)
}
```

Traces flow through the envelope's `meta` map using the `messaging.trace_id` and `messaging.span_id` keys.

### Metrics

Key metrics collected via Prometheus:

- **LiveKit**: Room count, participant count, track bitrate, packet loss
- **Agent**: Message processing latency, model inference time, active conversations
- **API**: Request rate, error rate, token generation time
- **Database**: Query latency, connection pool usage

### Logging

Structured logging with context:

```go
import "go.uber.org/zap"

logger := zap.L()

logger.Info("user_message_received",
    zap.String("conversation_id", conversationID),
    zap.String("message_id", messageID),
    zap.Int("content_length", len(content)),
)
```

## Security Considerations

### 1. Access Control

- LiveKit rooms isolated per conversation
- JWT tokens with 6-hour expiry (configurable)
- User identity verified via API authentication

### 2. Data Privacy

- All data stored locally (no external services)
- Conversation history encrypted at rest (PostgreSQL TDE)
- Audio never persisted (ephemeral streaming only)

### 3. Network Security

- TLS required for all connections (API and LiveKit)
- Self-hosted LiveKit server (no external media routing)
- TURN server with authentication

### 4. Rate Limiting

- API endpoints rate-limited per user
- Agent limits concurrent conversations per worker
- LiveKit enforces max participants per room

## System Capabilities

### Core Infrastructure

- PostgreSQL database with migrations
- Go backend service architecture (hexagonal/ports & adapters)
- REST API for conversation management
- LiveKit server integration
- MessagePack protocol implementation
- OpenTelemetry tracing support

### Backend Agent

- LiveKit agent worker
- Voice pipeline (STT → LLM → TTS)
- Whisper ASR integration (via speaches server)
- Qwen3 LLM integration (via LiteLLM/vLLM)
- Kokoro TTS integration (via speaches server)
- Real-time audio streaming
- Protocol message routing
- Conversation state management
- Message persistence

### Web Frontend

- React + TypeScript + Vite setup
- Basic UI components (ChatWindow, MessageList, InputBar, Sidebar)
- REST API integration for text messages
- Conversation management
- Message history display
- LiveKit client integration (`useLiveKit.ts` hook)
- Voice activity detection with microphone input (`MicrophoneVAD.tsx` component)
- Real-time streaming via data channels
- MessagePack protocol handling (core message types)

### Tools & Features

- Tool execution framework
- Tool coordinator
- Web search tool (full DuckDuckGo HTML integration)

### Memory & Context

- Database schema for memory storage
- pgvector integration for embeddings
- Memory retrieval use cases (integrated in `generate_response.go:71-90`)
- Memory storage from conversations
- Context augmentation with memories

### Multi-Platform Support

- Web frontend (full voice support)
- Android app (Kotlin/Compose, LiveKit, Porcupine wake word, Room database)
- CLI tool (interactive chat, conversation management, streaming responses)

### Advanced Features

#### DSPy + GEPA Prompt Optimization

The system integrates Stanford's DSPy framework with GEPA optimizer for automatic prompt improvement.

Key capabilities:
- **Automatic Prompt Optimization**: Uses GEPA's reflective mutation to evolve better prompts
- **Tool Usage Optimization**: Improves tool descriptions, argument generation, and result formatting
- **Memory-Aware Learning**: Leverages conversation memories as few-shot demonstrations
- **Feedback Loop**: User votes and notes feed directly into optimization metrics

#### Silero VAD (Voice Activity Detection)

Silero VAD is implemented in the web frontend (`frontend/src/utils/sileroVAD.ts`) to automatically detect when users start and stop speaking:

- Browser-based speech detection using Silero VAD
- Configurable sensitivity threshold
- Automatic microphone activation on speech
- Integrated with MicrophoneVAD component

## Conclusion

This architecture provides a robust foundation for Alicia's real-time voice assistant capabilities. LiveKit handles the complex real-time communication layer, allowing the system to focus on the AI pipeline and conversational semantics. The design supports multiple client platforms, scales from single-device to cloud deployment, and maintains clear separation of concerns between media transport and protocol messaging.

Key strengths:

- **Battle-tested infrastructure**: LiveKit provides production-grade WebRTC
- **Multi-platform support**: Web, mobile, and CLI with consistent SDKs
- **Flexible deployment**: Self-hosted for privacy or cloud for scale
- **Clean architecture**: Clear separation between transport (LiveKit) and semantics (Alicia protocol)
- **Extensible**: Easy to add new message types, tools, or AI models

The modular design enables incremental development with early testing and validation of core functionality.

## See Also

- [Components](COMPONENTS.md) - Detailed component breakdown
- [Database Schema](DATABASE.md) - Data model and schema
- [Protocol Specification](protocol/index.md) - Binary protocol details
- [Server Documentation](SERVER.md) - HTTP server implementation
- [Agent Documentation](AGENT.md) - LiveKit agent implementation
