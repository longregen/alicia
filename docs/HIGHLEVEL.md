# Alicia

Alicia is a self-hosted personal AI voice assistant that enables natural, real-time conversations through audio. Running entirely on consumer hardware, it combines speech recognition, language understanding, and voice synthesis to create a seamless conversational experience with complete privacy.

## What Alicia Does

Alicia transforms how you interact with AI through voice, providing a fluid, human-like conversation experience:

### Implemented Features ✅

- **Real-time Voice Conversations**: Full voice pipeline with Whisper STT → Qwen3 LLM → Kokoro TTS
- **LiveKit Integration**: Real-time audio streaming through LiveKit for all clients
- **Web Frontend**: React-based interface with voice and text conversations, conversation management
- **Android App**: Native Kotlin/Compose app with LiveKit integration and Porcupine wake word detection
- **CLI Tool**: Interactive command-line chat with streaming responses and conversation management
- **Semantic Memory**: Context retrieval from previous conversations using pgvector embeddings
- **Tool Integration**: Calculator, DuckDuckGo web search, memory query, and MCP protocol support
- **Streaming Responses**: Real-time streaming of assistant responses as they're generated
- **Conversation Management**: Create, list, archive, and manage multiple conversations
- **Message History**: Persistent storage of conversation history with offline sync support
- **Privacy-First**: All processing happens locally on your hardware
- **Extensible Architecture**: Hexagonal architecture with clean separation of concerns

### Planned Features 🚧

- **Silero VAD (Voice Activity Detection)**: Automatic speech detection in the web frontend using Silero VAD, eliminating the need for push-to-talk buttons
- **Debugging & Evals**: Prompt optimization for messages, tool use, and memory
- **Personal Knowledge Integration**: Full integration with personal knowledge database systems

## Architecture

### Real-Time Communication

Alicia uses **LiveKit** as its real-time communication layer for audio streaming. Each conversation runs in a dedicated LiveKit room (`conv_{conversation_id}`), providing:

- Low-latency bidirectional audio streaming
- Reliable data channels for protocol messages
- Secure token-based access control
- Automatic reconnection with state preservation

### Conversation Protocol

Alicia implements a **MessagePack-based protocol** over LiveKit data channels with 16 message types:

- User and assistant messages
- Audio chunks and transcriptions
- Tool use requests and results
- Reasoning steps and memory traces
- Control signals (stop, variation)
- Configuration and acknowledgements

This separation keeps audio transport (LiveKit tracks) independent from conversation logic (MessagePack protocol), enabling clean architecture and flexibility.

### Local AI Processing

All AI processing happens locally on your machine, with no cloud dependencies:

- **Speech Recognition**: Whisper for real-time transcription across multiple languages
- **Language Understanding**: Qwen3 via vLLM for intelligent, contextual responses
- **Voice Synthesis**: Kokoro for natural-sounding speech output
- **Embeddings**: pgvector for semantic memory search

## The Alicia Experience

### Voice Conversation Flow

1. **Connect**: Join a conversation via web, mobile, or CLI
2. **Speak Naturally**: Your speech is transcribed in real-time using Whisper
3. **Context Retrieval**: Relevant memories are retrieved to augment context
4. **Immediate Response**: Alicia processes your message and begins responding
5. **Streaming Audio**: Hear Alicia's voice response as it's generated
6. **Memory Storage**: Important information is stored for future context

### Text Mode

All platforms support text-only mode where you can type messages and receive text responses, with optional TTS playback.

## Multi-Platform Support

### Web Frontend ✅

A React-based web interface with full feature support:

- Voice conversations via LiveKit SDK
- Text input with streaming responses
- Conversation management and history
- MCP server configuration
- Audio input/output controls

### Android App ✅

A native Kotlin/Jetpack Compose application:

- LiveKit SDK for real-time audio
- Porcupine wake word detection ("Hey Alicia")
- Room database for local storage
- Background voice service
- Hilt dependency injection

### CLI Tool ✅

A Go-based command-line interface:

- Interactive chat with streaming responses
- Conversation management commands (new, list, show, delete)
- Database integration for persistence
- Configuration via environment variables

## Self-Hosted on Consumer Hardware

Alicia runs entirely on standard consumer devices - no cloud services required. This ensures:

- **Privacy**: Your conversations stay on your device
- **Reliability**: No dependency on external services
- **Performance**: Optimized for efficient resource usage on standard hardware
- **Control**: You own your data and your assistant

## Technology Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.25, Chi, pgx, LiveKit Server SDK |
| Frontend | React 19, TypeScript, Vite 7, LiveKit Components |
| Android | Kotlin, Jetpack Compose, Hilt, Room, LiveKit SDK |
| Database | PostgreSQL with pgvector |
| Protocol | MessagePack over LiveKit data channels |
| AI/ML | Whisper (ASR), Qwen (LLM), Kokoro (TTS) |
| Build | Nix Flakes, sqlc |
