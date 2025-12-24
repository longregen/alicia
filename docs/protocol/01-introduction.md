# Alicia Real-Time Binary Protocol Specification

## Introduction

This document specifies the **Alicia binary communication protocol** for real-time conversational AI. The protocol enables streaming interactions between clients and AI agents over [LiveKit](https://livekit.io)'s real-time communication infrastructure. It defines how messages, audio, and control signals flow within a conversation, using LiveKit's data channels for reliable message transport and audio tracks for voice streaming.

### Purpose and Scope

The Alicia protocol supports interactive conversational AI with the following capabilities:

- **Streaming Responses**: Assistant messages stream token-by-token to clients as they are generated
- **Voice Input/Output**: Real-time audio capture and playback through LiveKit audio tracks
- **Tool Integration**: Assistants invoke external functions and services during conversations
- **Memory and Context**: Conversations maintain persistent state with database-backed message history
- **Multi-Modal Interaction**: Text, audio, and structured data flow seamlessly within a single conversation

### LiveKit as the Communication Layer

The protocol operates within LiveKit rooms, where:

- **Data Channels** carry MessagePack-encoded binary messages for text, tool invocations, control signals, and metadata
- **Audio Tracks** stream voice input from users and voice output from assistants
- **Room** represents a single conversation session
- **Participants** represent the client and agent(s) involved in the conversation

Each message is wrapped in a common **envelope format** that includes metadata and identifiers for ordering, tracing, and conversation management. The envelope structure, combined with LiveKit's reliable transport, ensures messages arrive in order and can be correlated with the Alicia database schema.

### Serialization

All messages use [MessagePack](https://msgpack.org) for binary serialization. MessagePack provides compact, efficient encoding while remaining schema-less and language-neutral. LiveKit data channels carry these MessagePack-encoded messages as binary payloads.

### Specification Format

This specification follows RFC/W3C conventions. Key words such as **MUST**, **SHOULD**, and **MAY** are interpreted as described in RFC 2119, indicating requirement levels. Implementers must follow these requirements to ensure interoperability between different Alicia clients and agents.
