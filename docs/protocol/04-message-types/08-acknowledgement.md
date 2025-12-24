### 8. Acknowledgement (Type 8)

**Purpose:** Confirms receipt or processing of specific messages. This serves as a flow control mechanism over the LiveKit data channel, acknowledging specific messages or cumulative progress in the conversation. While the underlying transport is reliable, acknowledgements are useful for reconnection handshakes and confirming critical control operations.

**Typical Direction:** Bidirectional (via LiveKit data channel). The server sends acknowledgements to confirm receipt of user messages or control commands. The client sends acknowledgements to confirm receipt of assistant messages or to support flow control.

**Fields:**

* `conversationId` (Text): Conversation ID (maps to LiveKit room name).
* `acknowledgedStanzaId` (Int32, optional): The stanzaId of a specific message being acknowledged. For example, after a user sends stanzaId 5, the server sends an Acknowledgement where acknowledgedStanzaId = 5 to indicate "I received your message #5".
* `acknowledgedMessageId` (Text, NanoID, optional): Alternatively or additionally, the NanoID of the message being acknowledged. This is more explicit and useful when acknowledging a particular message out-of-order. In most cases, using stanzaId suffices since it's sequential, but NanoID can be used for more complex scenarios.
* `lastSeenStanzaId` (Int32, optional): A cumulative acknowledgement indicator, meaning "I have received everything up to this stanzaId". This is useful during reconnection or as periodic ACKs so the other side knows the highest message number that has been received. If this field is present, it implies all messages with absolute stanza ID ≤ that value have been received.
* `notes` (Text, optional): Additional information for debugging purposes, e.g. "ACK for user message" or context. Usually not needed in production.

**MessagePack Representation (Informative):**

```
{
  "conversationId": "conv_7H93k",
  "acknowledgedStanzaId": 5,
  "acknowledgedMessageId": "msg_u1A2B",
  "lastSeenStanzaId": 5,
  "notes": "Processing user query"
}
```

**Semantics:** An Acknowledgement is optional in this protocol because the underlying LiveKit transport is reliable. However, it becomes important in specific scenarios:

* **Flow control:** When the server sends a long stream of data (like many AssistantSentence messages), the client sends acknowledgements periodically to signal that it is keeping up or to adjust rate. If the client stops acknowledging and the server has a policy to wait or slow down, it uses this signal.
* **Reconnection:** When a client reconnects after a drop, it sends `lastSequenceSeen` in the Configuration handshake. That essentially serves a similar role to an acknowledgement: "I saw everything up through X". The server may also send an Acknowledgement once it resumes, to confirm the resumption point.
* **Confirm critical actions:** After a ControlStop (type 10) is sent by the client to stop generation, the server SHOULD send an Acknowledgement to confirm it received the stop command, since the user wants assurance it took effect. Similarly, after a ControlVariation (type 11) is processed, an acknowledgement confirms receipt.

**Usage:** The protocol does not mandate an acknowledgement for every message. Acknowledgements are typically used:

* After Configuration handshake: to confirm successful initialization.
* After Control messages (stop/variation): to confirm the server will act on them.
* For streaming flow control: the client acknowledges important milestones (like "I have rendered up to sentence 5" by acknowledging that stanzaId).
* As keepalive signals: in congested networks, acknowledgements can double as pings by sending the last seen stanzaId with no new information to keep the connection alive.

**Database Alignment:** Acknowledgements are not stored as part of conversation history since they are low-level protocol signals, not user or assistant content. They are not recorded in message tables like `alicia_user_messages`. They may be logged at a telemetry level for debugging purposes (perhaps in metadata or separate debug logs). The specification includes them for completeness of the protocol, but implementations do not persist them beyond runtime.
