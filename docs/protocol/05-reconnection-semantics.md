## Reconnection Semantics

One of the critical features of the Alicia protocol is the ability to resume a conversation seamlessly after a dropped connection. This section describes how reconnection is handled using LiveKit's built-in reconnection capabilities combined with the protocol-level `lastSequenceSeen` field in the Configuration message. The goals of reconnection support are:

* Avoid duplicating messages the client already received
* Continue an in-progress answer or operation when possible
* Maintain ordering and conversation integrity

## Initial Connection and Resume

**First-Time Connection:**

When a client first connects to Alicia (no prior conversation or starting fresh), it follows this flow:

1. The client obtains a LiveKit access token from the Alicia authentication service
2. The client joins the LiveKit room using the SDK with the provided token
3. Once the LiveKit room connection is established, the client sends a Configuration message over the LiveKit data channel with no `conversationId` (or with an explicit flag indicating it's new). The `lastSequenceSeen` in this case is `0` (meaning nothing seen yet)
4. The server creates a new conversation and responds, providing the assigned `conversationId`. Typically, the server sends an Acknowledgement or a Configuration response containing the new `conversationId`
5. After this handshake, the conversation proceeds normally

**Reconnection to Existing Conversation:**

When a client reconnects (after a network disruption, app reload, or LiveKit connection drop) and wants to continue an existing conversation:

1. The client rejoins the same LiveKit room using the original conversationId as the room identifier
2. LiveKit automatically restores the room connection and any active audio/video tracks
3. The client sends a Configuration message with `conversationId` set to the known ID of that conversation, and `lastSequenceSeen` set to the last stanza sequence number it has processed
4. The server verifies the conversation and the client's authorization, then compares `lastSequenceSeen` to the latest message sequence in that conversation
5. The server replays any messages the client missed

## Example Reconnection Scenario

Suppose conversation ID `abc123` has 30 server messages (server's lastServerStanzaID is -30) when the connection drops. The client only received through server message 25 (stanzaId -25).

**Reconnection flow:**

1. Client rejoins LiveKit room with room name derived from `abc123`
2. Client sends: `Configuration { conversationId: "abc123", lastSequenceSeen: -25 }`
3. Server responds: `Acknowledgement { acknowledgedStanzaId: -25, conversationId: "abc123" }` — confirming "you're caught up through -25"
4. Server detects messages -26 through -30 are missing on the client
5. Server retransmits messages -26 through -30 in order over the data channel
6. These retransmissions use the original content and IDs (they are not fresh messages, just delivered again)
7. The server ensures order is preserved
8. After resending, the client is caught up, and subsequent messages can flow

The server may send an Acknowledgement or special response after the Configuration handshake indicating the resume was successful and possibly listing how many messages will be resent or the current lastSequence. However, it can also directly start resending messages.

## LiveKit-Specific Reconnection Benefits

LiveKit provides several automatic reconnection features that complement the protocol-level reconnection:

**Automatic Track Recovery:**
* Audio and video tracks are automatically restored when the connection is re-established
* The client doesn't need to manually resubscribe to tracks
* This ensures voice conversations continue seamlessly

**Built-in Buffering:**
* LiveKit buffers data channel messages during brief disconnections
* Short network hiccups may not require protocol-level replay if LiveKit's buffer covers the gap

**Room State Synchronization:**
* LiveKit automatically synchronizes room state (participants, tracks) on reconnection
* The protocol focuses on message-level synchronization via `lastSequenceSeen`

## Use of `lastStanzaMap`

In typical linear conversation flow, `lastSequenceSeen` suffices to indicate the point up to which the client is synced. The `lastStanzaMap` is optional and supports advanced synchronization scenarios:

* **Multiple independent sequences:** The map can hold entries for different message types or channels if they progress independently, such as `{"text": 25, "audio": 10}`. In the base protocol, audio and text share a single sequence
* **Partial message delivery:** For extremely large AssistantMessages split below the sentence level, `lastStanzaMap` can map message IDs to delivery markers. Since the protocol chunks messages at the sentence level (each AssistantSentence has a stanza id), the sequence number typically covers synchronization needs

The client may omit `lastStanzaMap` in normal cases. If provided, the server uses it to refine which messages need retransmission. For example, if the client received messages 26 and 27 completely, but only got 3 of 5 sentence chunks from message 28, `lastStanzaMap` indicates messageId 28 → last sentence seq 3. The server then resends sentences 4 and 5 if possible.

Implementation of such granular synchronization is complex; a simpler approach resends the entire message 28 or continues from sentence 4 if the server tracks this internally. Since MessagePack messages don't inherently support mid-message resumption, the server typically resends all chunks for a given message.

The `lastStanzaMap` is an advanced hint mechanism. It can be left as an empty map or omitted unless both client and server implement partial message resumption.

## Handling Message Replay

**Duplicate Prevention:**

The server takes care not to duplicate IDs. It resends existing messages rather than generating new ones. Retransmitted message envelopes carry the original stanzaId (negative for server-sent messages) and original message ID. The client, which might still have these in its log, should recognize duplicates by comparing IDs. With correct use of lastSequenceSeen, duplicates should not occur (except potentially the next message if lastSequenceSeen was off by one).

**Large Gaps:**

If the gap is large (client missed many messages), the server might throttle how fast it resends to avoid overwhelming the client. It may send them as a burst though, since the client likely will just append them quickly.

**Expired Conversations:**

If the conversation has ended or expired on the server side (some systems archive or clean up archived conversations), and a client tries to resume, the server responds with an ErrorMessage indicating conversation not found or not resumable. The server should inform the client rather than silently starting a fresh conversation, as this could confuse the user.

**Security:**

The server ensures the user reconnecting is indeed authorized for that conversation. LiveKit access tokens tie the user to conversation rights, preventing impostors from guessing a conversationId and replaying lastSequence.

## Resuming In-Progress Activities

If the connection drops while the server is in the middle of generating an answer (i.e., has sent StartAnswer and some AssistantSentence chunks but not all):

**Server-Side Behavior:**

* The server likely pauses or stops generation when it detects the LiveKit connection dropped (or it keeps generating in background)
* If it keeps generating, it has the full answer ready or partially buffered
* On reconnection, the client's lastSequenceSeen indicates it only got, say, 2 out of 5 sentences of that answer
* The server detects from lastSequenceSeen that some of the sequence for that answer (with certain previousId) didn't finish
* If it has the sentences buffered, it sends the rest with their original stanza IDs
* If it didn't buffer, it might regenerate or decide to abort that answer and send an Error or simply not continue

**Best Practices:**

One possibility is that the server aborts the incomplete answer entirely upon reconnection, and lets the user re-ask if needed. However, this creates a suboptimal user experience. A better implementation caches the generation context so it can continue sending. If using a stateless model, it may be hard to resume mid-answer. Some advanced systems might re-prompt the model to finish or simply had the model's output already buffered (some models can produce output faster than it's sent).

For the specification, the server should attempt to continue the answer if feasible. If not, it should send a final piece marking the answer truncated or handle it gracefully (maybe sending an ErrorMessage explaining that the answer could not complete due to disconnect).

**Audio Track Continuity:**

LiveKit automatically restores audio tracks on reconnection. If the server was streaming audio response (TTS) when the connection dropped:

* The audio track is automatically resubscribed
* The server should resume audio streaming from where it left off
* Protocol-level synchronization via `lastSequenceSeen` helps identify which audio chunks were delivered

## Acknowledgements in Resume

After sending Configuration, the client might wait for an explicit acknowledgement. The server can send an Acknowledgement that includes `acknowledgedStanzaId = lastSequenceSeen` to confirm "I know where you were". Then it proceeds with any resends. This is optional, but some handshake confirmation is good practice.

## Complete Reconnection Flow Example

**Scenario:**

1. Client was in conversation ID `abc123`, last received server message had stanzaId -30 (the 30th server message)
2. LiveKit connection drops (network issue)
3. Client detects disconnection and attempts reconnect
4. Client rejoins LiveKit room for conversation `abc123`
5. LiveKit establishes connection, restores audio/video tracks
6. Client sends over data channel: `Configuration { conversationId: "abc123", lastSequenceSeen: -30 }`
7. Server responds: `Acknowledgement { acknowledgedStanzaId: -30, conversationId: "abc123" }`
8. Server checks and sees the last server message in that conversation has stanzaId -35 (the 35th server message, so messages 31-35 were missed)
9. Server resends messages with stanzaIds -31, -32, -33, -34, -35 in order over the data channel. These could include the remaining AssistantSentence chunks of the answer that was cut off
10. After resending, the server continues normally. If message -35 was the final part of an answer, it waits for the next user message
11. Client receives the missed messages, updates UI accordingly (completing an answer that was partially shown)
12. Conversation proceeds seamlessly

## Multiple Conversations

The Alicia protocol assumes one conversation per LiveKit room. If a client needs to handle multiple chats, it opens separate LiveKit room connections or handles them sequentially. The protocol does not multiplex conversations on one room connection. Therefore, the conversationId in the envelope remains constant for all messages after the initial handshake.

## Implementation Requirements

**Clients and servers must implement the reconnection handshake using Configuration.** Clients should always provide lastSequenceSeen (even if 0), and servers should honor it by not re-sending earlier messages except to fill gaps. Both sides should be robust to duplicates (e.g., if lastSequenceSeen was slightly behind and the client gets one or two messages it actually had, it should deduplicate or ignore gracefully by comparing IDs).
