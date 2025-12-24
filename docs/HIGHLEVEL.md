# Audio Alicia

Alicia is a self-hosted personal AI voice assistant that enables natural, real-time conversations through audio. Running entirely on consumer hardware, it combines speech recognition, language understanding, and voice synthesis to create a seamless conversational experience with complete privacy.

> **Current Status**: The backend voice assistant is fully implemented with LiveKit integration. The web frontend currently supports text-only conversations via REST API. Voice features in the web interface are planned for future releases. See [Multi-Platform Support](#multi-platform-support) for details.

## What Alicia Does

Alicia transforms how you interact with AI through voice, providing a fluid, human-like conversation experience:

### Implemented Features ✅

- **Backend Voice Pipeline**: Fully implemented real-time voice processing (Whisper STT → Qwen3 LLM → Kokoro TTS)
- **LiveKit Integration**: Backend agent supports real-time audio streaming through LiveKit
- **Text Conversations**: Web interface supports text-based conversations via REST API
- **Conversation Management**: Create, list, and manage multiple conversations
- **Message History**: Persistent storage of conversation history
- **Privacy-First**: All processing happens locally on your hardware
- **Extensible Architecture**: Hexagonal architecture with clean separation of concerns

### Planned Features 🚧

- **Web Voice Interface**: LiveKit integration in web frontend for voice conversations
- **Real-time Streaming**: Streaming responses as they're being generated
- **Multilingual Translation**: Speak in one language and receive responses in another
- **Mobile Apps**: Native Android app
- **CLI Tool**: Command-line interface for terminal-based interactions
- **Voice & Text Flexibility**: Switch seamlessly between voice and text input/output
- **Conversation Memory**: Context retrieval from previous conversations
- **Tool Integration**: Web search and other external tools (currently stubbed)

## Architecture

### Real-Time Communication

Alicia uses **LiveKit** as its real-time communication layer for audio/video streaming. Each conversation runs in a dedicated LiveKit room, providing:

- Low-latency bidirectional audio streaming
- Real-time video support (when needed)
- Reliable transport for voice data
- Secure peer-to-peer connections

### Conversation Protocol

Alicia implements a **MessagePack-based protocol** over LiveKit data channels to handle conversation semantics:

- Message exchange for conversation state
- Transcription events
- Assistant responses
- Control signals (pause, resume, regenerate)
- Translation settings and multilingual support

This separation keeps audio/video transport (LiveKit) independent from conversation logic (MessagePack protocol), enabling clean architecture and flexibility.

### Local AI Processing

All AI processing happens locally on your machine, with no cloud dependencies:

- **Speech Recognition**: Whisper (distil-large-v3) for fast real-time transcription across multiple languages
- **Language Understanding**: Qwen3-8B-AWQ for intelligent, contextual responses
- **Voice Synthesis**: Kokoro for natural-sounding speech output

These models work together to create a cohesive, responsive conversation experience that feels natural and engaging.

## The Alicia Experience

### Current Experience (Backend Agent) ✅

When using the backend LiveKit agent directly:

1. **Connect via LiveKit**: Join a LiveKit room with the Alicia agent
2. **Speak Naturally**: The agent listens and transcribes your speech in real-time using Whisper
3. **Immediate Response**: Alicia processes your message and begins responding
4. **Listen to Voice**: Hear Alicia's voice response synthesized with Kokoro TTS
5. **Real-time Audio**: Audio streams in real-time over LiveKit tracks

### Current Experience (Web Frontend) ✅

The web interface currently provides:

1. **Text-Based Chat**: Type your messages in the input field
2. **View Responses**: Read Alicia's text responses
3. **Manage Conversations**: Create, switch between, and delete conversations
4. **Message History**: Browse previous messages in each conversation

### Planned Experience (Web Frontend with Voice) 🚧

The enhanced web experience will include:

1. **Speak Naturally**: Click to activate microphone and start talking
2. **Watch Your Words Appear**: See your speech transcribed on screen as you speak
3. **Immediate Response**: Alicia begins responding immediately after you finish speaking
4. **Listen & Read Along**: Hear Alicia's voice response while following along with the text

### Planned Features 🚧

- **Translation Mode**: Speak in one language and receive responses in another
- **Voice Options**: Choose from multiple voice options for TTS
- **Control the Conversation**: Stop responses mid-stream, regenerate answers, or continue from any point

## Self-Hosted on Consumer Hardware

Alicia runs entirely on standard consumer devices - no cloud services required. This ensures:

- **Privacy**: Your conversations stay on your device
- **Reliability**: No dependency on external services
- **Performance**: Optimized for efficient resource usage on standard hardware
- **Control**: You own your data and your assistant

## Multi-Platform Support

### Current Platform Support

- **Desktop & Mobile Web** ✅: A responsive web interface (text-only) that works across desktop browsers and mobile devices
  - Implemented: Text conversations via REST API
  - Planned: LiveKit integration for voice conversations

### Planned Platform Support 🚧

- **Native Mobile**: Android application with voice assistant integration for a native mobile experience
  - Platform: Android (Kotlin/Java)
  - Status: Not yet implemented
  - Features: Native audio, background support, push notifications

- **Command Line**: Terminal interface for quick interactions and scripting
  - Language: Go
  - Status: Not yet implemented
  - Features: Voice and text modes, history export, script integration
