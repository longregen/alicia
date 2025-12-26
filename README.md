# Alicia

A self-hosted, consumer-hardware voice assistant with real-time audio streaming, semantic memory, and extensible tool integration.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Clients                                    │
├───────────────┬───────────────────────────────┬─────────────────────────┤
│  Web Frontend │         CLI Tool              │     Android App         │
│  React 19     │         Cobra                 │     Kotlin/Compose      │
│  LiveKit SDK  │         Streaming Chat        │     LiveKit SDK         │
└───────┬───────┴───────────────┬───────────────┴────────────┬────────────┘
        │                       │                            │
        └───────────────────────┼────────────────────────────┘
                                │ MessagePack/HTTP
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           Go Backend                                    │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
│  │   LiveKit   │  │   HTTP API  │  │  LLM Client │  │   Tool System   │ │
│  │   Agent     │  │   (Chi)     │  │  (LiteLLM)  │  │   (MCP + Builtin│ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └────────┬────────┘ │
│         │                │                │                  │          │
│  ┌──────┴────────────────┴────────────────┴──────────────────┴────────┐ │
│  │                    Application Services                            │ │
│  │  Conversation │ Message │ Memory │ Tool │ Audio                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│         │                                                               │
│  ┌──────┴──────────────────────────────────────────────────────────────┐│
│  │                    PostgreSQL + pgvector                            ││
│  │  Conversations │ Messages │ Memory (embeddings) │ Tools │ MCP       ││
│  └─────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐       ┌───────────────┐       ┌───────────────┐
│   ASR Server  │       │   LLM Server  │       │   TTS Server  │
│   (Whisper)   │       │   (vLLM/Qwen) │       │   (Kokoro)    │
└───────────────┘       └───────────────┘       └───────────────┘
```

## Features

### Implemented

- **Voice Agent**: Real-time audio via LiveKit, Whisper ASR, Qwen LLM, Kokoro TTS
- **Web Frontend**: Chat interface, voice conversations, conversation management, MCP configuration
- **CLI**: Interactive chat, conversation commands, streaming responses
- **Tool System**: Calculator, DuckDuckGo search, memory query, MCP protocol support
- **Memory**: Semantic search with pgvector, automatic context augmentation
- **Android App**: Native Kotlin/Compose with LiveKit, Porcupine wake word, Room database

### Planned

- Multi-user conversations
- Video/screen sharing
- Enhanced memory consolidation

## Quick Start

### Prerequisites

- Nix (recommended) or Go 1.24+, Node.js 22+, PostgreSQL with pgvector
- LiveKit server
- LLM server (vLLM with Qwen or compatible)
- ASR/TTS server (speaches or compatible)

### Development

```bash
# Using Nix (recommended)
nix develop
go test ./...
npm --prefix frontend test

# Manual setup
cp .env.example .env
# Edit .env with your configuration
go build -o bin/alicia ./cmd/alicia
./bin/alicia serve
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ALICIA_POSTGRES_URL` | PostgreSQL connection string |
| `ALICIA_LLM_URL`, `ALICIA_LLM_MODEL` | LLM endpoint and model |
| `ALICIA_LIVEKIT_URL`, `ALICIA_LIVEKIT_API_KEY`, `ALICIA_LIVEKIT_API_SECRET` | LiveKit configuration |
| `ALICIA_ASR_URL`, `ALICIA_ASR_MODEL` | Speech recognition endpoint |
| `ALICIA_TTS_URL`, `ALICIA_TTS_MODEL`, `ALICIA_TTS_VOICE` | Text-to-speech endpoint |
| `ALICIA_EMBEDDING_URL`, `ALICIA_EMBEDDING_MODEL` | Embedding service |
| `ALICIA_SERVER_HOST`, `ALICIA_SERVER_PORT` | HTTP server binding |

See `.env.example` for full configuration.

## Project Structure

```
├── android/                 # Native Android app (Kotlin/Compose)
│   ├── app/                 # Main application module
│   ├── core/                # Shared modules (data, domain, network, database)
│   ├── feature/             # Feature modules (assistant, conversations, settings)
│   └── service/             # Background services (voice, hotkey)
├── cmd/alicia/              # CLI entry point
├── frontend/                # React web application
│   └── src/
│       ├── components/      # UI components
│       ├── hooks/           # React hooks (useSync, useLiveKit)
│       └── services/        # API and LiveKit integration
├── internal/                # Go backend
│   ├── adapters/            # External integrations (postgres, livekit, mcp, http)
│   ├── application/         # Business logic, services, use cases
│   ├── domain/              # Core entities and errors
│   ├── ports/               # Interface definitions
│   └── config/              # Configuration management
├── pkg/protocol/            # MessagePack protocol definitions
├── migrations/              # Database migrations
├── docs/                    # Documentation (mdbook)
└── extra/                   # Experiments and proof-of-concepts
```

## Technology Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24, Chi, pgx, LiveKit Server SDK |
| Frontend | React 19, TypeScript, Vite 7, LiveKit Components |
| Android | Kotlin, Jetpack Compose, Hilt, Room, LiveKit Android SDK |
| Database | PostgreSQL with pgvector |
| Protocol | MessagePack over LiveKit data channels |
| AI/ML | Whisper (ASR), Qwen (LLM), Kokoro (TTS), pgvector (embeddings) |
| Observability | OpenTelemetry, Prometheus |
| Build | Nix Flakes, sqlc |

## Protocol

Binary MessagePack protocol over LiveKit data channels:

- **Envelope**: StanzaID, ConversationID, Type, Body, Meta
- **Stanza IDs**: Client→Server positive (1, 2, 3...), Server→Client negative (-1, -2, -3...)
- **Reconnection**: Stanza tracking enables message recovery

See `docs/protocol/` for specification.

## Building

```bash
# Backend + Frontend
nix build .#alicia

# Android APK (hermetic)
nix build .#alicia-android

# Frontend only
cd frontend && npm run build

# Run tests
go test ./...
npm --prefix frontend test
npm --prefix frontend run test:e2e
```

### NixOS Android Development

On NixOS, the Android Gradle build requires an FHS environment to work around AAPT2 binary compatibility issues. Use the provided wrapper script:

```bash
# Enter the Android development shell
cd android

# Use gradlew-nix instead of gradlew for all Gradle commands
./gradlew-nix lint
./gradlew-nix build
./gradlew-nix assembleDebug

# Or use nix develop directly
nix develop .#android -c android-fhs-env -c './gradlew assembleDebug'
```

The FHS environment provides `/lib64/ld-linux-x86-64.so.2` and standard library paths needed by unpatched ELF binaries like AAPT2.

## CGO Requirement

The backend requires CGO for Opus audio codec (LiveKit audio processing):

```bash
# Ubuntu/Debian
sudo apt-get install gcc libc6-dev libopus-dev libopusfile-dev

# macOS
brew install opus opusfile
```

Used only in `internal/adapters/livekit/audio_converter.go`.

## Documentation

- `docs/` - mdbook documentation with architecture, protocol specs, and guides
- `extra/` - Proof-of-concept experiments (ElectricSQL, MessagePack, PGLite, etc.)

## License

See LICENSE file.
