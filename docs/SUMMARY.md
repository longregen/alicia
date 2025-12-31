# Alicia

[Introduction](HIGHLEVEL.md)

# Getting Started

- [Quick Start](QUICKSTART.md)
- [CLI Usage](CLI.md)

# System Design

- [Architecture Overview](ARCHITECTURE.md)
- [Components](COMPONENTS.md)
- [Database Schema](DATABASE.md)
- [ID Scheme](ID_SCHEME.md)

# Platform Guides

- [Go Server](SERVER.md)
- [Go Agent](AGENT.md)
- [React Frontend](FRONTEND_COMPONENTS.md)
- [Android App](ANDROID.md)

# Core Concepts

- [Conversation Workflow](CONVERSATION_WORKFLOW.md)
- [LiveKit Integration](LIVEKIT.md)
- [Prompt Optimization (GEPA)](GEPA_PRIMER.md)
- [Optimization System](OPTIMIZATION_SYSTEM.md)
- [MCP Client Orchestration Guide](MCP_CLIENT_PRIMER.md)

# Protocol Specification

- [Protocol Overview](protocol/index.md)
- [Introduction](protocol/01-introduction.md)
- [Conventions and Terminology](protocol/02-conventions.md)
- [Envelope Format](protocol/03-envelope-format.md)
- [Message Types](protocol/04-message-types/index.md)
  - [Error Message](protocol/04-message-types/01-error-message.md)
  - [User Message](protocol/04-message-types/02-user-message.md)
  - [Assistant Message](protocol/04-message-types/03-assistant-message.md)
  - [Audio Chunk](protocol/04-message-types/04-audio-chunk.md)
  - [Reasoning Step](protocol/04-message-types/05-reasoning-step.md)
  - [Tool Use Request](protocol/04-message-types/06-tool-use-request.md)
  - [Tool Use Result](protocol/04-message-types/07-tool-use-result.md)
  - [Acknowledgement](protocol/04-message-types/08-acknowledgement.md)
  - [Transcription](protocol/04-message-types/09-transcription.md)
  - [Control Stop](protocol/04-message-types/10-control-stop.md)
  - [Control Variation](protocol/04-message-types/11-control-variation.md)
  - [Configuration](protocol/04-message-types/12-configuration.md)
  - [Start Answer](protocol/04-message-types/13-start-answer.md)
  - [Memory Trace](protocol/04-message-types/14-memory-trace.md)
  - [Commentary](protocol/04-message-types/15-commentary.md)
  - [Assistant Sentence](protocol/04-message-types/16-assistant-sentence.md)
- [Reconnection Semantics](protocol/05-reconnection-semantics.md)
- [Database Alignment](protocol/06-database-alignment.md)
- [Implementation Guidelines](protocol/07-implementation-guidelines.md)
- [Example Sessions](protocol/08-example-sessions.md)

# Operations

- [Deployment](DEPLOYMENT.md)
- [Offline Sync](OFFLINE_SYNC.md)

# Reference

- [User Stories](USER_STORIES.md)
