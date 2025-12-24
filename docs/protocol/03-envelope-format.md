## Envelope Format

All messages are wrapped in an **Envelope** structure that provides common metadata for routing, ordering, and interpreting the message payload. Envelopes are serialized using MessagePack and transmitted over LiveKit data channels using the `publishData()` API.

### Envelope Fields

#### `stanzaId` (Int32)

The stanza ID identifies the message's position in the conversation sequence and indicates the sender by its sign:

- **Client messages**: Start at 1 and increment positively (1, 2, 3, ...)
- **Server messages**: Start at -1 and decrement negatively (-1, -2, -3, ...)

This bidirectional monotonic ordering enables easy sender identification and maintains sequence within each direction. The server and client reject or ignore messages that violate monotonic ordering (for example, if a client sends a stanzaId that is not greater than all its previously sent IDs).

#### `conversationId` (String)

The unique identifier of the conversation to which this message belongs. The conversation ID follows the format `conv_{nanoid}` and directly maps to the LiveKit room name, enabling:

- **Multiplexing**: Multiple conversations can exist simultaneously
- **Resumption**: Clients can reconnect to existing conversations by rejoining the corresponding LiveKit room

For a newly initialized connection, the client leaves this field blank or null to request a new conversation. When resuming an existing conversation, the client sets this to the known conversation ID from a prior session. Once a conversation is established, this field remains constant for all messages in that session.

#### `type` (UInt16)

A numeric code indicating the message type. Each message type in the protocol is assigned a numeric ID (1 through 16, defined in the Message Types section). This field's value corresponds to one of the defined message types. The receiver uses this `type` to determine how to decode and interpret the `body` field.

On the wire, this is encoded as a small unsigned integer. If a receiver encounters an unknown `type` code, it ignores or safely skips that message, allowing forward compatibility when new message types are added.

#### `meta` (Map)

A container for metadata associated with the message. This is a map/dictionary where keys are strings and values can be strings, numbers, or structured types. The metadata is optional and can be empty.

Common metadata includes:

- `timestamp`: Client message creation time
- `clientVersion`: Version of the client software
- Request context flags
- OpenTelemetry tracing fields (see below)

The Alicia database's `alicia_meta` table stores these entries for auditing and debugging. Each key-value pair from `meta` is persisted and associated with the message or conversation.

**OpenTelemetry Fields:**

The `meta` map includes OpenTelemetry fields for distributed tracing following the [OpenTelemetry Messaging Spans specification](https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/):

- **`messaging.trace_id` (String, optional)**: The OpenTelemetry trace identifier for this message's processing, allowing correlation with backend tracing spans
- **`messaging.span_id` (String, optional)**: The OpenTelemetry span identifier, used with the trace_id to uniquely identify the span within a trace

These fields are optional. When not present, no specific tracing is associated with the message.

#### `body` (varies)

The actual content of the message, discriminated by the `type` field. The body is encoded as one of the specific message structures defined in the Message Types section.

For example, if `type` is 2 (UserMessage), the body contains a `UserMessage` structure with fields like the user's text. In MessagePack, this is represented as a map where the keys correspond to the fields of the specific message type.

Implementations populate the body with the correct structure matching the `type`. The receiver interprets the body according to the type.

### MessagePack Representation

In MessagePack, the envelope is encoded as:

```
{
  "stanzaId": <Int32>,
  "conversationId": <String>,
  "type": <UInt16>,
  "meta": {
    "messaging.trace_id": <String>,    # optional OpenTelemetry trace ID
    "messaging.span_id": <String>,     # optional OpenTelemetry span ID
    <String>: <Any>,                   # other arbitrary metadata
    ...
  },
  "body": {
    ... # fields specific to the message type
  }
}
```

### Transport over LiveKit

Envelopes are transmitted over LiveKit data channels:

1. **Serialization**: The envelope is serialized to MessagePack binary format
2. **Transmission**: The binary data is sent using LiveKit's `publishData()` method
3. **Delivery**: LiveKit data channels provide reliable, ordered delivery within the room

Each envelope is sent as a discrete data packet. The LiveKit data channel handles framing and delivery, so no additional framing protocol is required.

**Example conceptual flow:**

```go
// Sending an envelope
envelope := Envelope{
    StanzaID:       1,
    ConversationID: "conv_abc123def456",
    Type:           2,
    Meta: map[string]string{
        "timestamp":          "2025-12-20T10:30:00Z",
        "messaging.trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
    },
    Body: UserMessageBody{
        Text: "Hello, Alicia!",
    },
}

// Serialize to MessagePack
packedData, _ := msgpack.Marshal(envelope)

// Send over LiveKit data channel
room.LocalParticipant.PublishData(packedData, livekit.DataPacketKind_RELIABLE)
```

### Metadata Design

The envelope's `meta` field is intended for general-purpose metadata and can contain arbitrary keys. The `conversationId`, `stanzaId`, and `type` fields are separated out as top-level fields (rather than being keys in meta) because they are critical for protocol operation and need to be readily accessible for routing and parsing.

Implementations do not put these critical values inside `meta`. For custom metadata, the `meta` map is used. The field name `meta` emphasizes that these are not transport-layer headers but application metadata, potentially persisted in Alicia's `alicia_meta` table.

### OpenTelemetry Context Propagation

For proper context propagation, implementations follow the [OpenTelemetry Messaging Spans specification](https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/) for propagating trace context between producers and consumers. The `messaging.trace_id` and `messaging.span_id` fields in the `meta` map enable correlation of protocol messages with backend tracing spans for monitoring and debugging.
