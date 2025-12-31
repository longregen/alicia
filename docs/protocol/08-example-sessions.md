## Example Sessions

This section illustrates how the Alicia protocol operates in practice using LiveKit as the transport layer. Each example includes narrative description and sequence diagrams.

## Example 1: Basic Text Q&A

This example shows a simple text conversation with streaming response.

### Narrative

**Setup:**
* User has already authenticated and received a LiveKit access token
* Client joins LiveKit room for a new conversation

**Flow:**

1. **Client joins LiveKit room** using the SDK with the access token
2. **Client sends Configuration** message over the data channel with no `conversationId` (indicating new conversation), `lastSequenceSeen: 0`
3. **Server creates conversation**, assigns `conversationId = "conv_7H93k..."`, and responds with Acknowledgement
4. **User types message**: "What is the capital of France?"
5. **Client sends UserMessage** with `stanzaId: 1`, content, generates NanoID `msg_u1A2B`, sets `previousId: null` (first message)
6. **Server acknowledges** receipt with Acknowledgement message
7. **Server begins generating response**, sends StartAnswer with `stanzaId: -1`, `id: "msg_a9X8Y"`, `previousId: "msg_u1A2B"`
8. **Server streams response** as AssistantSentence messages:
   - Sentence 1 (stanzaId: -2): "The capital of France is Paris."
   - Sentence 2 (stanzaId: -3): "It is located in the north-central part of the country." (isFinal: true)
9. **Conversation continues** or awaits next user message

### Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant L as LiveKit Room
    participant S as Server/Agent

    Note over C: User authenticates, gets token
    C->>L: Join room (new conversation)
    L->>S: Participant connected

    C->>S: Configuration {conversationId: "", lastSequenceSeen: 0}
    Note over S: Create conversation "conv_7H93k..."
    S->>C: Acknowledgement {conversationId: "conv_7H93k..."}

    Note over C: User types message
    C->>S: UserMessage {id: "msg_u1A2B", stanzaId: 1, content: "What is the capital of France?"}
    S->>C: Acknowledgement {acknowledgedStanzaId: 1}

    Note over S: Process query
    S->>C: StartAnswer {id: "msg_a9X8Y", stanzaId: -1, previousId: "msg_u1A2B"}
    S->>C: AssistantSentence {stanzaId: -2, sequence: 1, text: "The capital of France is Paris."}
    S->>C: AssistantSentence {stanzaId: -3, sequence: 2, text: "It is located in the north-central...", isFinal: true}
```

## Example 2: Voice Conversation

This example demonstrates a complete voice interaction with audio tracks and transcription.

### Narrative

**Setup:**
* Client has joined LiveKit room for existing conversation
* Microphone and speaker are configured

**Flow:**

1. **User speaks** into microphone
2. **Client publishes audio track** to LiveKit room (microphone input)
3. **Server receives audio** via subscribed track
4. **Server transcribes** in real-time, sends Transcription messages:
   - Partial (stanzaId: -1): "What's the..." (isFinal: false)
   - Partial (stanzaId: -2): "What's the weather..." (isFinal: false)
   - Final (stanzaId: -3): "What's the weather in Tokyo?" (isFinal: true)
5. **Server creates UserMessage** from final transcription with `stanzaId: -4`, `id: "msg_u5Z9P"`
6. **Server retrieves memory** about user's timezone preference, sends MemoryTrace (stanzaId: -5)
7. **Server calls weather tool**, sends ToolUseRequest (stanzaId: -6) and ToolUseResult (stanzaId: -7)
8. **Server generates response**, sends StartAnswer with `stanzaId: -8`, `id: "msg_aB3K7"`
9. **Server streams text response** as AssistantSentence messages (stanzaId: -9, -10)
10. **Server streams audio response** via TTS over audio track
11. **Client plays audio** from subscribed assistant audio track

### Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant L as LiveKit Room
    participant S as Server/Agent
    participant T as External Tools

    Note over C: User starts speaking
    C->>L: Publish audio track (microphone)
    L->>S: Audio frames

    S->>C: Transcription {stanzaId: -1, text: "What's the...", isFinal: false}
    S->>C: Transcription {stanzaId: -2, text: "What's the weather...", isFinal: false}
    S->>C: Transcription {stanzaId: -3, text: "What's the weather in Tokyo?", isFinal: true}

    S->>C: UserMessage {stanzaId: -4, id: "msg_u5Z9P", content: "What's the weather in Tokyo?"}

    Note over S: Retrieve user memory
    S->>C: MemoryTrace {stanzaId: -5, memoryType: "profile", content: "User timezone: JST"}

    Note over S: Need weather data
    S->>C: ToolUseRequest {stanzaId: -6, toolName: "get_weather", parameters: {city: "Tokyo"}}
    S->>T: Call weather API
    T->>S: Weather data
    S->>C: ToolUseResult {stanzaId: -7, success: true, data: {...}}

    S->>C: StartAnswer {stanzaId: -8, id: "msg_aB3K7", previousId: "msg_u5Z9P"}
    S->>C: AssistantSentence {stanzaId: -9, sequence: 1, text: "The current weather in Tokyo..."}
    S->>C: AssistantSentence {stanzaId: -10, sequence: 2, text: "...is sunny with a temperature of 22°C.", isFinal: true}

    S->>L: Publish audio track (TTS)
    L->>C: Audio frames
    Note over C: Speaker plays response
```

## Example 3: Reconnection Mid-Answer

This example shows how reconnection works when the connection drops during a streaming response.

### Narrative

**Initial State:**
* Conversation `conv_abc123` is active
* Server has sent messages up to stanzaId -6 (second sentence of an answer)
* Client receives up to stanzaId -6, then connection drops

**Reconnection Flow:**

1. **Connection drops** (network issue)
2. **Client detects disconnection**, stores `lastSequenceSeen: 6`
3. **Client rejoins LiveKit room** with same conversation identifier
4. **LiveKit restores connection** and resubscribes to audio tracks
5. **Client sends Configuration** with `conversationId: "conv_abc123"`, `lastSequenceSeen: 6`
6. **Server acknowledges** with Acknowledgement
7. **Server checks database** and finds messages with stanzaId -7, -8, -9 were not delivered:
   - Message -7: AssistantSentence (sequence 3)
   - Message -8: AssistantSentence (sequence 4, final)
   - Message -9: Commentary
8. **Server replays** missing messages in order
9. **Client receives** remaining sentences and completes the answer display
10. **Conversation continues** seamlessly

### Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant L as LiveKit Room
    participant S as Server/Agent
    participant D as Database

    Note over C,S: Initial conversation in progress
    S->>C: AssistantSentence {stanzaId: -5, sequence: 1}
    S->>C: AssistantSentence {stanzaId: -6, sequence: 2}

    Note over C,L: Connection drops
    Note over C: Stores lastSequenceSeen=6

    Note over C: Detect disconnect, attempt reconnect
    C->>L: Rejoin room "conv_abc123"
    L->>S: Participant reconnected

    C->>S: Configuration {conversationId: "conv_abc123", lastSequenceSeen: 6}
    S->>C: Acknowledgement {acknowledgedStanzaId: 6}

    S->>D: Query messages WHERE abs(stanza_id) > 6
    D->>S: Messages -7, -8, -9

    Note over S: Replay missed messages
    S->>C: AssistantSentence {stanzaId: -7, sequence: 3}
    S->>C: AssistantSentence {stanzaId: -8, sequence: 4, isFinal: true}
    S->>C: Commentary {stanzaId: -9, commentType: "system_note"}

    Note over C: Answer completed, UI updated
```

## Example 4: Message Editing with ControlVariation

This example demonstrates how users can edit their messages and trigger a new response.

### Narrative

**Initial State:**
* User has sent a message asking about Paris weather
* Server has started generating a response

**Edit Flow:**

1. **User sends message**: "What's the weather in Paris?" (UserMessage `id: "U100"`, stanzaId 1)
2. **Server sends StartAnswer** (`id: "A200"`, stanzaId -1, previousId: "U100")
3. **User realizes mistake** and wants to ask about London instead
4. **Client sends ControlVariation** with `targetId: "U100"`, `mode: "edit"` (stanzaId 2)
5. **Client sends new UserMessage**: "What's the weather in London?" (UserMessage `id: "U101"`, stanzaId 3)
6. **Server receives ControlVariation**, marks message U100 as replaced, cancels generation for A200
7. **Server receives new UserMessage U101**, begins processing
8. **Server sends new StartAnswer** for London weather (`id: "A201"`, stanzaId -2, previousId: "U101")
9. **Server streams response** about London weather
10. **Database reflects** U100 is replaced by U101, A200 is cancelled, A201 is active

### Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server/Agent
    participant D as Database

    Note over C: User types and sends
    C->>S: UserMessage {id: "U100", stanzaId: 1, content: "What's the weather in Paris?"}
    S->>D: Insert message U100

    S->>C: StartAnswer {id: "A200", stanzaId: -1, previousId: "U100"}
    S->>D: Insert message A200 (streaming)

    Note over C: User realizes mistake, clicks edit
    C->>S: ControlVariation {stanzaId: 2, targetId: "U100", mode: "edit"}
    Note over S: Cancel generation for A200
    S->>D: Update A200 status=cancelled

    C->>S: UserMessage {id: "U101", stanzaId: 3, content: "What's the weather in London?"}
    S->>D: Insert U101, mark U100 as replaced_by=U101

    Note over S: Process new message
    S->>C: StartAnswer {id: "A201", stanzaId: -2, previousId: "U101"}
    S->>C: AssistantSentence {stanzaId: -3, sequence: 1, text: "The weather in London..."}
    S->>C: AssistantSentence {stanzaId: -4, sequence: 2, text: "...is cloudy with light rain.", isFinal: true}
```

## Example 5: Multi-Turn Conversation with Memory

This example shows how memory is retrieved and used across multiple turns.

### Narrative

**Turn 1:**
1. User: "My favorite color is blue"
2. Server stores this as a memory preference
3. Server responds: "I'll remember that your favorite color is blue"

**Turn 2 (later in conversation):**
1. User: "What colors should I use for my website?"
2. Server retrieves memory about favorite color
3. Server sends MemoryTrace indicating the preference was retrieved
4. Server responds: "Since blue is your favorite color, I recommend using shades of blue..."

### Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server/Agent
    participant M as Memory Store

    Note over C: Turn 1
    C->>S: UserMessage {content: "My favorite color is blue"}

    Note over S: Extract preference
    S->>M: Store memory {type: "preference", content: "favorite color: blue"}

    S->>C: StartAnswer
    S->>C: AssistantSentence {text: "I'll remember that your favorite color is blue.", isFinal: true}

    Note over C: Turn 2 (later)
    C->>S: UserMessage {content: "What colors should I use for my website?"}

    Note over S: Semantic search
    S->>M: Query relevant memories
    M->>S: Return preference: "favorite color: blue"

    S->>C: MemoryTrace {memoryType: "preference", content: "favorite color: blue", usage: "retrieved"}

    S->>C: StartAnswer
    S->>C: AssistantSentence {text: "Since blue is your favorite color..."}
    S->>C: AssistantSentence {text: "I recommend using shades of blue...", isFinal: true}
```

## Example 6: Error Handling and Retry

This example demonstrates error handling when a tool fails.

### Narrative

1. **User asks**: "What's the latest news?"
2. **Server calls news API tool**, but the API is down
3. **Server sends ToolUseRequest** and **ToolUseResult** with `success: false`
4. **Server sends ErrorMessage** explaining the tool failure
5. **Server attempts fallback**: sends response without real-time news
6. **User retries**: sends ControlVariation with `mode: "retry"`
7. **Server tries again**, this time the API succeeds
8. **Server sends successful response** with news

### Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server/Agent
    participant T as News API

    C->>S: UserMessage {content: "What's the latest news?"}

    S->>C: ToolUseRequest {toolName: "get_news"}
    S->>T: API call
    T--xS: Error: Service unavailable

    S->>C: ToolUseResult {success: false, error: "API unavailable"}
    S->>C: ErrorMessage {severity: "warning", content: "News service temporarily unavailable"}

    S->>C: StartAnswer
    S->>C: AssistantSentence {text: "I'm unable to fetch the latest news right now...", isFinal: true}

    Note over C: User clicks retry
    C->>S: ControlVariation {targetId: "msg_u...", mode: "retry"}

    S->>C: ToolUseRequest {toolName: "get_news"}
    S->>T: API call (retry)
    T->>S: Success: News data

    S->>C: ToolUseResult {success: true, data: {...}}
    S->>C: StartAnswer
    S->>C: AssistantSentence {text: "Here are the latest headlines...", isFinal: true}
```

## Implementation Notes

### Message Flow over LiveKit

All protocol messages in these examples flow through LiveKit:

**Data Channel:**
* Configuration, UserMessage, AssistantSentence, ToolUseRequest, ToolUseResult, MemoryTrace, Commentary, ControlVariation, ErrorMessage, Acknowledgement
* Sent as MessagePack-encoded binary over LiveKit's reliable data channel

**Audio Tracks:**
* User microphone input: Published as audio track by client
* Assistant TTS output: Published as audio track by server
* Automatically managed by LiveKit (subscription, buffering, playback)

**Room Management:**
* Each conversation maps to one LiveKit room
* Room name typically matches conversationId
* Access controlled via JWT tokens

### StanzaId Sequencing

In all examples:
* **Client messages** use positive numbers starting at 1: 1, 2, 3, 4...
* **Server messages** use negative numbers starting at -1: -1, -2, -3, -4...
* **Monotonically increasing** in absolute value
* Used for ordering and reconnection tracking

### Database Persistence

Each protocol message corresponds to database operations:
* UserMessage → Insert into `alicia_messages` with `role='user'`
* AssistantSentence → Insert into `alicia_sentences`, aggregate to `alicia_messages`
* MemoryTrace → Insert into `alicia_memory_used`
* ToolUseRequest/Result → Insert/update in `alicia_tool_uses`
* Commentary → Insert into `alicia_commentaries`

The database maintains a complete record while LiveKit provides real-time delivery.

## Testing These Scenarios

Implementers should test each example scenario to ensure:

1. **Basic Q&A**: Text messages flow correctly, streaming works
2. **Voice**: Audio tracks publish/subscribe correctly, transcription integrates
3. **Reconnection**: Message replay works, no duplicates, audio tracks restore
4. **Editing**: ControlVariation properly cancels and replaces
5. **Memory**: Retrieval and logging work across turns
6. **Errors**: Graceful degradation, retry mechanisms function

These examples cover the core protocol patterns and should serve as templates for building comprehensive test suites.
