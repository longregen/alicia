## Implementation Guidelines

This section provides guidelines and best practices for implementing the Alicia protocol using LiveKit as the real-time communication layer.

## 1. LiveKit Integration

### Client Implementation

Clients use the LiveKit SDK to establish real-time connections and send/receive protocol messages over data channels.

**Basic Setup:**

```typescript
import { Room, RoomEvent, DataPacket_Kind } from 'livekit-client';

// Initialize LiveKit room
const room = new Room({
  adaptiveStream: true,
  dynacast: true,
});

// Obtain token from your auth service
const token = await fetchLiveKitToken(conversationId);

// Connect to room
await room.connect(livekitUrl, token);

// Listen for data channel messages
room.on(RoomEvent.DataReceived, (payload: Uint8Array, participant) => {
  const message = msgpack.decode(payload);
  handleProtocolMessage(message);
});

// Send protocol messages over data channel
function sendMessage(message: any) {
  const encoded = msgpack.encode(message);
  room.localParticipant.publishData(encoded, DataPacket_Kind.RELIABLE);
}
```

**Audio Track Handling:**

```typescript
// Subscribe to audio tracks for assistant responses
room.on(RoomEvent.TrackSubscribed, (track, publication, participant) => {
  if (track.kind === 'audio' && participant.identity === 'assistant') {
    const audioElement = track.attach();
    document.body.appendChild(audioElement);
  }
});

// Publish microphone input
const audioTrack = await createLocalAudioTrack();
await room.localParticipant.publishTrack(audioTrack);
```

### Server Implementation

Servers use the LiveKit SDK to handle room connections and implement protocol logic.

**Basic Agent Structure:**

```go
import (
    "github.com/livekit/protocol/livekit"
    "github.com/vmihailenco/msgpack/v5"
)

type AliciaAgent struct {
    room           *livekit.Room
    conversationID string
    lastSequence   int32
}

func (a *AliciaAgent) OnParticipantConnected(participant *livekit.Participant) {
    // Handle new participant joining the room
    log.Info("Participant connected", "identity", participant.Identity)
}

func (a *AliciaAgent) OnDataReceived(data []byte, participant *livekit.Participant) {
    // Handle protocol messages from data channel
    var message map[string]interface{}
    if err := msgpack.Unmarshal(data, &message); err != nil {
        log.Error("Error handling message", "error", err)
        a.sendError(participant, err.Error())
        return
    }
    a.handleProtocolMessage(message, participant)
}

func (a *AliciaAgent) handleProtocolMessage(message map[string]interface{}, participant *livekit.Participant) {
    // Route protocol messages to handlers
    msgType := message["type"].(uint16)

    switch msgType {
    case 12: // Configuration
        a.handleConfiguration(message, participant)
    case 2: // UserMessage
        a.handleUserMessage(message, participant)
    case 11: // ControlVariation
        a.handleControlVariation(message, participant)
    // ... other message types
    }
}

func (a *AliciaAgent) sendProtocolMessage(message interface{}) error {
    // Send protocol message to all participants
    encoded, _ := msgpack.Marshal(message)
    return a.room.LocalParticipant.PublishData(encoded, livekit.DataPacketKind_RELIABLE)
}
```

**Streaming Response Pattern:**

```go
func (a *AliciaAgent) streamAssistantResponse(userMessageID, prompt string) {
    // Create assistant message in database
    assistantID := nanoid.New()

    // Send StartAnswer
    a.sendProtocolMessage(Envelope{
        StanzaID:       -a.nextSequence(),
        Type:           13, // StartAnswer
        ConversationID: a.conversationID,
        Body: StartAnswerBody{
            ID:         assistantID,
            PreviousID: userMessageID,
        },
    })

    // Stream response from LLM (via LiteLLM)
    sequence := 1
    for sentence := range a.llm.StreamResponse(prompt) {
        isFinal := sentence.IsLast

        a.sendProtocolMessage(Envelope{
            StanzaID:       -a.nextSequence(),
            Type:           16, // AssistantSentence
            ConversationID: a.conversationID,
            Body: AssistantSentenceBody{
                PreviousID: assistantID,
                Sequence:   sequence,
                Text:       sentence.Text,
                IsFinal:    isFinal,
            },
        })

        sequence++
    }
}
```

## 2. MessagePack Serialization

### Schema Definition

Define clear MessagePack schemas for all message types to ensure consistent serialization.

**TypeScript Example:**

```typescript
interface ProtocolEnvelope {
  stanzaId: number;
  type: number;
  conversationId: string;
  meta?: Record<string, string>;
}

interface UserMessage {
  id: string;
  previousId: string | null;
  content: string;
}

interface ProtocolMessage extends ProtocolEnvelope {
  // Message-specific fields based on type
  [key: string]: any;
}
```

**Go Example:**

```go
type ProtocolEnvelope struct {
    StanzaID       int32             `msgpack:"stanzaId"`
    Type           uint16            `msgpack:"type"`
    ConversationID string            `msgpack:"conversationId"`
    Meta           map[string]string `msgpack:"meta,omitempty"`
    Body           interface{}       `msgpack:"body"`
}

type UserMessage struct {
    ID         string `msgpack:"id"`
    PreviousID string `msgpack:"previousId,omitempty"`
    Content    string `msgpack:"content"`
}
```

### Encoding Best Practices

**Efficient Binary Encoding:**
* Use MessagePack's binary type for audio data, not base64 strings
* Keep field names short but readable (MessagePack encodes keys as strings)
* Use integer constants for message types instead of string names

**Error Handling:**

```typescript
function decodeMessage(data: Uint8Array): ProtocolMessage | null {
  try {
    const message = msgpack.decode(data);

    // Validate required envelope fields
    if (!message.stanzaId || !message.type) {
      logger.error('Invalid message: missing required fields');
      return null;
    }

    return message as ProtocolMessage;
  } catch (error) {
    logger.error('MessagePack decode error:', error);
    return null;
  }
}
```

## 3. Security Considerations

### LiveKit Token-Based Authentication

LiveKit uses JWT tokens for authentication and authorization.

**Token Generation (Server-Side):**

```go
import (
    "github.com/livekit/protocol/auth"
    "time"
)

func createToken(userID, conversationID string) (string, error) {
    // Create LiveKit access token for a conversation
    token := auth.NewAccessToken(livekitAPIKey, livekitAPISecret)

    token.SetIdentity(userID)
    token.SetName("User " + userID)
    token.AddGrant(&auth.VideoGrant{
        RoomJoin:       true,
        Room:           conversationID,
        CanPublish:     true,
        CanSubscribe:   true,
        CanPublishData: true,
    })

    // Token expires in 6 hours
    token.SetValidFor(6 * time.Hour)

    return token.ToJWT()
}
```

**Token Verification:**

The LiveKit server automatically verifies tokens. Your application server should:
* Only issue tokens to authenticated users
* Include user identity in token claims
* Set appropriate permissions (publish/subscribe/data)
* Use reasonable TTL values

### Transport Security

**TLS Encryption:**
* LiveKit connections use WebRTC with DTLS/SRTP for media encryption
* Data channels use SCTP over DTLS for encrypted message transport
* Always use `wss://` (secure WebSocket) for LiveKit signaling

**Input Validation:**

```go
func validateUserMessage(message UserMessage) bool {
    // Check content length
    if len(message.Content) > MaxMessageLength {
        return false
    }

    // Validate ID format (NanoID)
    if !isValidNanoID(message.ID) {
        return false
    }

    // Check for injection attacks
    if containsMaliciousContent(message.Content) {
        return false
    }

    return true
}
```

**Rate Limiting:**

```go
import (
    "sync"
    "time"
)

type RateLimiter struct {
    maxMessages  int
    window       time.Duration
    userMessages map[string][]time.Time
    mu           sync.Mutex
}

func NewRateLimiter(maxMessages int, windowSeconds int) *RateLimiter {
    return &RateLimiter{
        maxMessages:  maxMessages,
        window:       time.Duration(windowSeconds) * time.Second,
        userMessages: make(map[string][]time.Time),
    }
}

func (r *RateLimiter) IsAllowed(userID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-r.window)

    // Remove old messages
    var valid []time.Time
    for _, ts := range r.userMessages[userID] {
        if ts.After(cutoff) {
            valid = append(valid, ts)
        }
    }
    r.userMessages[userID] = valid

    // Check limit
    if len(r.userMessages[userID]) >= r.maxMessages {
        return false
    }

    r.userMessages[userID] = append(r.userMessages[userID], now)
    return true
}
```

## 4. Performance Optimization

### Data Channel Efficiency

**Batching Small Messages:**

For rapid updates like typing indicators, batch multiple small messages:

```typescript
class MessageBatcher {
  private queue: any[] = [];
  private timeout: NodeJS.Timeout | null = null;

  queue(message: any) {
    this.queue.push(message);

    if (!this.timeout) {
      this.timeout = setTimeout(() => this.flush(), 10);
    }
  }

  flush() {
    if (this.queue.length > 0) {
      const batch = msgpack.encode(this.queue);
      room.localParticipant.publishData(batch, DataPacket_Kind.RELIABLE);
      this.queue = [];
    }
    this.timeout = null;
  }
}
```

**Reliable vs. Lossy Delivery:**

Choose the appropriate delivery guarantee:

```typescript
// Use RELIABLE for critical protocol messages
function sendCriticalMessage(message: any) {
  const encoded = msgpack.encode(message);
  room.localParticipant.publishData(encoded, DataPacket_Kind.RELIABLE);
}

// Use LOSSY for real-time updates that can be dropped
function sendTranscriptionUpdate(text: string) {
  const message = { type: 9, text, isFinal: false };
  const encoded = msgpack.encode(message);
  room.localParticipant.publishData(encoded, DataPacket_Kind.LOSSY);
}
```

### Audio Streaming Optimization

**Chunk Size:**

Balance latency and overhead:

```go
const (
    ChunkSizeMs     = 20    // 20ms chunks for low latency (common for voice)
    SampleRate      = 16000 // Hz
    SamplesPerChunk = (SampleRate * ChunkSizeMs) / 1000 // 320 samples
)

func streamAudio(audioData []byte) {
    // Stream audio in optimally-sized chunks
    chunkSize := SamplesPerChunk * 2 // 2 bytes per sample (16-bit)

    for i := 0; i < len(audioData); i += chunkSize {
        end := i + chunkSize
        if end > len(audioData) {
            end = len(audioData)
        }
        chunk := audioData[i:end]
        publishAudioFrame(chunk)
        time.Sleep(time.Duration(ChunkSizeMs) * time.Millisecond) // Maintain real-time pacing
    }
}
```

**Adaptive Streaming:**

LiveKit handles adaptive bitrate automatically for audio tracks. For data channel messages:

```go
type AdaptiveMessageSender struct {
    pendingAcks int
    maxPending  int
    mu          sync.Mutex
    cond        *sync.Cond
}

func NewAdaptiveMessageSender() *AdaptiveMessageSender {
    s := &AdaptiveMessageSender{maxPending: 10}
    s.cond = sync.NewCond(&s.mu)
    return s
}

func (s *AdaptiveMessageSender) SendWithBackpressure(message interface{}) {
    s.mu.Lock()
    // Wait if too many unacknowledged messages
    for s.pendingAcks >= s.maxPending {
        s.cond.Wait()
    }
    s.pendingAcks++
    s.mu.Unlock()

    s.sendMessage(message)
}

func (s *AdaptiveMessageSender) OnAcknowledgement(stanzaID int32) {
    s.mu.Lock()
    // Decrease pending count when acknowledged
    if s.pendingAcks > 0 {
        s.pendingAcks--
    }
    s.cond.Signal()
    s.mu.Unlock()
}
```

### Memory Management

**Buffering Strategy:**

```go
type BufferedMessage struct {
    StanzaID  int32
    Data      interface{}
    Timestamp time.Time
}

type MessageBuffer struct {
    buffer  []BufferedMessage
    maxSize int
    mu      sync.RWMutex
}

func NewMessageBuffer(maxSize int) *MessageBuffer {
    return &MessageBuffer{
        buffer:  make([]BufferedMessage, 0, maxSize),
        maxSize: maxSize,
    }
}

func (b *MessageBuffer) Add(message Envelope) {
    b.mu.Lock()
    defer b.mu.Unlock()

    // Add message to buffer for potential replay
    b.buffer = append(b.buffer, BufferedMessage{
        StanzaID:  message.StanzaID,
        Data:      message,
        Timestamp: time.Now(),
    })

    // Trim to max size
    if len(b.buffer) > b.maxSize {
        b.buffer = b.buffer[len(b.buffer)-b.maxSize:]
    }
}

func (b *MessageBuffer) GetMessagesSince(lastSeen int32) []interface{} {
    b.mu.RLock()
    defer b.mu.RUnlock()

    var result []interface{}
    for _, msg := range b.buffer {
        if abs(msg.StanzaID) > lastSeen {
            result = append(result, msg.Data)
        }
    }
    return result
}
```

## 5. Error Handling

### Connection Errors

**Automatic Reconnection:**

```typescript
room.on(RoomEvent.Disconnected, async () => {
  logger.warn('Disconnected from room, attempting reconnection...');

  let attempts = 0;
  const maxAttempts = 5;

  while (attempts < maxAttempts) {
    try {
      await room.connect(livekitUrl, token);
      logger.info('Reconnected successfully');

      // Resume conversation
      await sendConfiguration(conversationId, lastSequenceSeen);
      break;
    } catch (error) {
      attempts++;
      const delay = Math.min(1000 * Math.pow(2, attempts), 10000);
      await sleep(delay);
    }
  }

  if (attempts >= maxAttempts) {
    logger.error('Failed to reconnect after maximum attempts');
    showUserError('Connection lost. Please refresh the page.');
  }
});
```

### Protocol Errors

**Graceful Degradation:**

```go
func (a *AliciaAgent) handleUserMessage(message UserMessage) {
    // Handle UserMessage with error recovery
    defer func() {
        if r := recover(); r != nil {
            log.Error("Error processing user message", "error", r)

            // Send error to client
            a.sendErrorMessage(
                "Sorry, I encountered an error processing your message.",
                "error",
                "PROCESSING_ERROR",
            )
        }
    }()

    // Validate message
    if !validateUserMessage(message) {
        a.sendErrorMessage(
            "Invalid message format",
            "warning",
            "",
        )
        return
    }

    // Process message
    response := a.processMessage(message)
    a.sendAssistantResponse(response)
}
```

**Error Message Protocol:**

```go
func (a *AliciaAgent) sendErrorMessage(content, severity, errorCode string) {
    // Send ErrorMessage to client
    a.sendProtocolMessage(Envelope{
        StanzaID:       -a.nextSequence(),
        Type:           1, // ErrorMessage
        ConversationID: a.conversationID,
        Body: ErrorMessageBody{
            ID:        nanoid.New(),
            Content:   content,
            Severity:  severity,
            ErrorCode: errorCode,
        },
    })
}
```

## 6. Ordering and Idempotency

### Sequence Number Management

**Client-Side:**

```typescript
class SequenceManager {
  private nextClientSequence = 1;

  getNextSequence(): number {
    const seq = this.nextClientSequence;
    this.nextClientSequence += 2;  // Client uses positive odd numbers
    return seq;
  }

  resetForNewConversation() {
    this.nextClientSequence = 1;
  }
}
```

**Server-Side:**

```go
type SequenceManager struct {
    nextServerSequence int32
    mu                 sync.Mutex
}

func NewSequenceManager() *SequenceManager {
    return &SequenceManager{nextServerSequence: 2} // Server uses negative even numbers
}

func (s *SequenceManager) GetNextSequence() int32 {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Get next server sequence (negative)
    seq := -s.nextServerSequence
    s.nextServerSequence += 2
    return seq
}
```

### Duplicate Detection

**Message Deduplication:**

```typescript
class MessageDeduplicator {
  private seen = new Set<string>();

  isDuplicate(message: ProtocolMessage): boolean {
    const key = `${message.conversationId}-${message.stanzaId}`;

    if (this.seen.has(key)) {
      logger.debug(`Duplicate message detected: ${key}`);
      return true;
    }

    this.seen.add(key);

    // Clean up old entries (keep last 1000)
    if (this.seen.size > 1000) {
      const iterator = this.seen.values();
      this.seen.delete(iterator.next().value);
    }

    return false;
  }
}
```

## 7. Testing Best Practices

### Unit Tests

Test individual message handlers:

```go
func TestUserMessageHandling(t *testing.T) {
    agent := NewAliciaAgent()

    var sentMessages []Envelope
    agent.sendProtocolMessage = func(msg Envelope) error {
        sentMessages = append(sentMessages, msg)
        return nil
    }

    userMessage := UserMessage{
        ID:         "msg-user-1",
        PreviousID: "",
        Content:    "Hello!",
    }

    agent.handleUserMessage(userMessage)

    // Verify StartAnswer was sent
    require.NotEmpty(t, sentMessages)
    assert.Equal(t, uint16(13), sentMessages[0].Type) // StartAnswer
}
```

### Integration Tests

Test full conversation flows:

```typescript
describe('Voice Conversation Flow', () => {
  it('should handle complete voice interaction', async () => {
    const room = await connectToRoom(testConversationId);

    // Send audio chunks
    const audioStream = await getMicrophoneStream();
    await publishAudioTrack(room, audioStream);

    // Wait for transcription
    const transcription = await waitForMessage(room, msg => msg.type === 9);
    expect(transcription.isFinal).toBe(true);

    // Wait for assistant response
    const startAnswer = await waitForMessage(room, msg => msg.type === 13);
    const sentences = await collectMessages(
      room,
      msg => msg.type === 16 && msg.previousId === startAnswer.id
    );

    expect(sentences.length).toBeGreaterThan(0);
    expect(sentences[sentences.length - 1].isFinal).toBe(true);
  });
});
```

### Reconnection Tests

Test reconnection scenarios:

```go
func TestReconnectionReplay(t *testing.T) {
    // Setup: conversation with 10 messages
    agent := setupConversationWithMessages(t, 10)

    // Simulate client disconnect and reconnect
    client := NewMockClient(7) // lastSeen=7
    client.Connect(agent.RoomName)

    // Send Configuration with lastSequenceSeen=7
    client.SendConfiguration(ConfigurationBody{
        ConversationID:   agent.ConversationID,
        LastSequenceSeen: 7,
    })

    // Verify messages 8-10 are replayed
    replayed := client.CollectMessages(5 * time.Second)
    assert.Len(t, replayed, 3)
    for _, msg := range replayed {
        assert.GreaterOrEqual(t, abs(msg.StanzaID), int32(8))
    }
}
```

## 8. Monitoring and Observability

### OpenTelemetry Integration

Add tracing to protocol messages:

```go
import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("alicia-agent")

func (a *AliciaAgent) handleUserMessage(ctx context.Context, message Envelope) {
    // Handle UserMessage with tracing
    ctx, span := tracer.Start(ctx, "handle_user_message")
    defer span.End()

    span.SetAttributes(
        attribute.String("conversation_id", message.ConversationID),
        attribute.String("message_id", message.Body.(UserMessageBody).ID),
    )

    // Add trace ID to meta
    traceID := span.SpanContext().TraceID().String()
    if message.Meta == nil {
        message.Meta = make(map[string]string)
    }
    message.Meta["messaging.trace_id"] = traceID

    a.processMessage(ctx, message)
}
```

### Metrics Collection

Track key metrics:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "time"
)

var (
    messageCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "alicia_messages_total",
            Help: "Total protocol messages",
        },
        []string{"type", "direction"},
    )

    messageLatency = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "alicia_message_latency_seconds",
            Help: "Message processing latency",
        },
        []string{"type"},
    )
)

func (a *AliciaAgent) handleProtocolMessage(message Envelope) {
    // Handle message with metrics
    messageCounter.WithLabelValues(
        fmt.Sprintf("%d", message.Type),
        "received",
    ).Inc()

    startTime := time.Now()
    a.routeMessage(message)

    messageLatency.WithLabelValues(
        fmt.Sprintf("%d", message.Type),
    ).Observe(time.Since(startTime).Seconds())
}
```

## Summary

Key implementation requirements:

* **LiveKit SDK** for client and server connectivity
* **LiveKit Agents framework** for server-side protocol handling
* **MessagePack** for efficient binary serialization
* **Token-based authentication** via LiveKit JWT tokens
* **Data channels** for protocol messages (RELIABLE for critical, LOSSY for real-time updates)
* **Audio tracks** for voice streaming (automatically managed by LiveKit)
* **Proper error handling** with reconnection logic
* **Sequence number management** for ordering guarantees
* **Observability** with tracing and metrics

The protocol works seamlessly with LiveKit's infrastructure while adding application-level semantics for conversations, memory, tools, and more.
