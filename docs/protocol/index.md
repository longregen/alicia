# Alicia Real-Time Binary Protocol Specification

This document provides an overview of the **Alicia binary communication protocol** for real-time streaming conversations between clients and servers. The protocol runs over **LiveKit**, a WebRTC-based platform that provides the underlying transport infrastructure.

The protocol uses **MessagePack** for message serialization, providing efficient binary encoding for all message types. LiveKit handles the transport layer through **data channels** (for protocol messages) and **audio tracks** (for voice streaming), allowing the protocol to focus on conversation semantics rather than low-level networking concerns.

## Protocol Documentation Structure

The protocol specification is divided into the following chapters:

1. [Introduction](./01-introduction.md) - Overview of the protocol's purpose and scope
2. [Conventions and Terminology](./02-conventions.md) - Key terms and conventions used throughout the specification
3. [Envelope Format](./03-envelope-format.md) - The common wrapper structure for all messages
4. [Message Types](./04-message-types/index.md) - Message types defined in the protocol:
   - Types 1-16: Core messages (fully documented)
   - Types 17-19: Reserved (17-18 used for Sync)
   - Types 20-27: Feedback & Memory messages
   - Type 28: Reserved
   - Types 29-31: Optimization messages
   - Types 40-43: Subscription messages (for multiplexed WebSocket)
   - [ErrorMessage](./04-message-types/01-error-message.md) (Type 1)
   - [UserMessage](./04-message-types/02-user-message.md) (Type 2)
   - [AssistantMessage](./04-message-types/03-assistant-message.md) (Type 3)
   - [AudioChunk](./04-message-types/04-audio-chunk.md) (Type 4)
   - [ReasoningStep](./04-message-types/05-reasoning-step.md) (Type 5)
   - [ToolUseRequest](./04-message-types/06-tool-use-request.md) (Type 6)
   - [ToolUseResult](./04-message-types/07-tool-use-result.md) (Type 7)
   - [Acknowledgement](./04-message-types/08-acknowledgement.md) (Type 8)
   - [Transcription](./04-message-types/09-transcription.md) (Type 9)
   - [ControlStop](./04-message-types/10-control-stop.md) (Type 10)
   - [ControlVariation](./04-message-types/11-control-variation.md) (Type 11)
   - [Configuration](./04-message-types/12-configuration.md) (Type 12)
   - [StartAnswer](./04-message-types/13-start-answer.md) (Type 13)
   - [MemoryTrace](./04-message-types/14-memory-trace.md) (Type 14)
   - [Commentary](./04-message-types/15-commentary.md) (Type 15)
   - [AssistantSentence](./04-message-types/16-assistant-sentence.md) (Type 16)
5. [Reconnection Semantics](./05-reconnection-semantics.md) - How to handle connection drops and resume conversations
6. [Database Alignment](./06-database-alignment.md) - How protocol messages map to database structures
7. [Implementation Guidelines](./07-implementation-guidelines.md) - Best practices for implementing the protocol
8. [Example Sessions](./08-example-sessions.md) - Narrative examples of protocol usage

## Transport Layer

The protocol leverages LiveKit's WebRTC-based infrastructure:

- **Data channels** provide reliable, ordered delivery of protocol messages serialized with MessagePack
- **Audio tracks** provide low-latency streaming of voice data between participants
- **Connection management** is handled by LiveKit, including NAT traversal, encryption, and resilience

This separation allows the protocol to focus on conversation semantics while LiveKit handles the complexities of real-time communication over the internet.

## Specification Style

This specification is written in an RFC-style format. Key words like **MUST**, **SHOULD**, and **MAY** are to be interpreted as described in RFC 2119, indicating requirement levels. Implementers are expected to follow these requirements to ensure interoperability between different Alicia clients and servers.

## See Also

- [Architecture Overview](../ARCHITECTURE.md) - System context
- [Agent Documentation](../AGENT.md) - Protocol implementation
- [Server Documentation](../SERVER.md) - HTTP API complement
- [Frontend Components](../FRONTEND_COMPONENTS.md) - Client implementation
