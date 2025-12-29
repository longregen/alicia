# Alicia Frontend Migration Plan

## Executive Summary

This document outlines the comprehensive migration strategy for incorporating the new-components library into the Alicia frontend. The migration enables enhanced features including:

- **Rich tool usage display** with status indicators and expandable details
- **Reasoning steps visualization** with inline rendering
- **Original audio preservation** with sentence-level playback
- **Streaming responses** with sentence-by-sentence appearance
- **Memory traces display** with relevance scores per message
- **Silero VAD integration** with LiveKit audio streaming
- **Normalized state management** with ConversationStore

---

## Table of Contents

1. [Architecture Comparison](#1-architecture-comparison)
2. [Protocol Changes](#2-protocol-changes)
3. [Backend Changes](#3-backend-changes)
4. [Frontend Changes](#4-frontend-changes)
5. [Migration Phases](#5-migration-phases)
6. [Feature Implementation Details](#6-feature-implementation-details)

---

## 1. Architecture Comparison

### 1.1 State Management

| Aspect | Current Architecture | New Architecture |
|--------|---------------------|------------------|
| **Message Storage** | SQLite (sql.js) + REST API | Normalized ConversationStore + SQLite |
| **Streaming State** | `Map<sequence, content>` in Context | `Record<SentenceId, MessageSentence>` |
| **Tool State** | Array of `{request, result}` pairs | `Record<ToolCallId, ToolCall>` with discriminated unions |
| **Audio State** | AudioChunks via LiveKit | AudioManager with IndexedDB |
| **ID Types** | Plain strings | Branded types (MessageId, SentenceId, etc.) |

### 1.2 Protocol Differences

| Message Type | Current | New Components Expectation | Gap |
|--------------|---------|---------------------------|-----|
| **Sentences** | Server sends `AssistantSentence` with sequence | Client infers from `.!?\n` delimiters | Adapter needed |
| **Audio** | Server sends `AudioChunk` via LiveKit | Client generates TTS from text | Keep server TTS, add storage |
| **Tools** | Separate `ToolUseRequest`/`ToolUseResult` | Single `ToolCall` with status | Status mapping layer |
| **Reasoning** | `ReasoningStep` messages | Embedded in content as `<reasoning>` tags | Already compatible |
| **Transcription** | `Transcription` with `final` flag | User message with `transcription` field | Store final transcription |

### 1.3 Component Hierarchy

```
Current:                              New (Using Our Hooks):
ChatWindow                            ChatWindow (organism)
├── MessageList                       ├── MessageList (organism)
│   └── MessageBubble                 │   ├── UserMessage (organism)
│       └── ToolUsageDisplay          │   │   └── ChatBubble (molecule)
├── ProtocolDisplay                   │   ├── AssistantMessage (organism)
│   ├── ReasoningSteps               │   │   └── ChatBubble (molecule)
│   ├── ToolUsages                    │   │       └── ComplexAddons (atom)
│   ├── Errors                        │   └── StreamingMessage (organism)
│   └── MemoryTraces                  ├── InputArea (organism)
├── AudioInput                        │   ├── MicrophoneVAD (molecule)
├── AudioOutput                       │   ├── ResizableBarTextInput (atom)
└── InputBar                          │   └── InputSendButton (atom)
                                      └── ResponseControls (organism)

Kept Infrastructure:
├── contexts/MessageContext.tsx    # Streaming state, tools, errors
├── hooks/useLiveKit.ts            # LiveKit connection + audio
├── hooks/useMessages.ts           # SQLite message persistence
├── hooks/useWebSocketSync.ts      # Real-time sync
└── hooks/useDatabase.ts           # SQLite access
```

---

## 2. Protocol Changes

### 2.1 No Breaking Changes Required

The current backend protocol is **more feature-complete** than what new-components expects. We will:
- Keep existing protocol messages unchanged
- Build adapter layer to transform protocol → new state format
- Extend new-components to handle additional message types

### 2.2 Protocol Adapter Layer

Create `/frontend/src/adapters/protocolAdapter.ts`:

```typescript
import { Envelope, MessageType } from '../types/protocol';
import { ConversationStore, ToolCall, MessageSentence } from '../new-components/types/streaming';

export function handleProtocolMessage(
  envelope: Envelope,
  store: ConversationStore
): ConversationStore {
  switch (envelope.type) {
    case MessageType.StartAnswer:
      return handleStartAnswer(store, envelope.body);

    case MessageType.AssistantSentence:
      return handleAssistantSentence(store, envelope.body);

    case MessageType.ToolUseRequest:
      return handleToolUseRequest(store, envelope.body);

    case MessageType.ToolUseResult:
      return handleToolUseResult(store, envelope.body);

    case MessageType.ReasoningStep:
      return handleReasoningStep(store, envelope.body);

    case MessageType.AudioChunk:
      return handleAudioChunk(store, envelope.body);

    case MessageType.Transcription:
      return handleTranscription(store, envelope.body);

    case MessageType.MemoryTrace:
      return handleMemoryTrace(store, envelope.body);

    default:
      return store;
  }
}
```

### 2.3 Message Type Mappings

#### AssistantSentence → MessageSentence
```typescript
// Protocol (from server)
interface AssistantSentence {
  id?: string;
  previousId: string;
  conversationId: string;
  sequence: number;
  text: string;
  isFinal?: boolean;
  audio?: Uint8Array;
}

// Adapter transforms to:
interface MessageSentence {
  id: SentenceId;              // = sentence.id || generate
  messageId: MessageId;        // = currentStreamingMessageId
  content: string;             // = sentence.text
  sequence: number;            // = sentence.sequence
  audioRefId?: AudioRefId;     // = store audio, get ref
  isComplete: boolean;         // = sentence.isFinal || false
}
```

#### ToolUseRequest/Result → ToolCall
```typescript
// Protocol messages
interface ToolUseRequest {
  id: string;
  messageId: string;
  toolName: string;
  parameters: Record<string, unknown>;
  execution: 'server' | 'client' | 'either';
}

interface ToolUseResult {
  requestId: string;
  success: boolean;
  result?: unknown;
  errorMessage?: string;
}

// Adapter transforms to:
type ToolCall =
  | { status: 'pending', id: ToolCallId, toolName, arguments, messageId, startTimeMs }
  | { status: 'executing', ... }
  | { status: 'success', resultContent: string, endTimeMs: number, ... }
  | { status: 'error', error: string, endTimeMs: number, ... };

// Mapping:
// ToolUseRequest → { status: 'pending', ... }
// ToolUseRequest (executing) → { status: 'executing', ... }
// ToolUseResult (success=true) → { status: 'success', resultContent: JSON.stringify(result), ... }
// ToolUseResult (success=false) → { status: 'error', error: errorMessage, ... }
```

### 2.4 Audio Storage Integration

Current: Audio auto-plays via LiveKit audio track
New: Store audio for replay, attach to sentences

```typescript
// When receiving AudioChunk:
async function handleAudioChunk(
  store: ConversationStore,
  chunk: AudioChunk,
  audioManager: AudioManager
): Promise<ConversationStore> {
  // 1. Store the audio data
  const audioRefId = await audioManager.store(chunk.data);

  // 2. Create AudioRef metadata
  const audioRef: AudioRef = {
    id: audioRefId,
    sizeBytes: chunk.data.byteLength,
    durationMs: chunk.durationMs,
    sampleRate: 16000, // from format
  };

  // 3. Associate with sentence (by sequence)
  const sentenceId = findSentenceBySequence(store, chunk.sequence);

  return {
    ...store,
    audioRefs: { ...store.audioRefs, [audioRefId]: audioRef },
    sentences: {
      ...store.sentences,
      [sentenceId]: {
        ...store.sentences[sentenceId],
        audioRefId,
      },
    },
  };
}
```

---

## 3. Backend Changes

### 3.1 No Immediate Changes Required

The backend already provides all necessary features:
- Sentence-level streaming via `AssistantSentence`
- Tool use via `ToolUseRequest`/`ToolUseResult`
- Reasoning via `ReasoningStep`
- Audio via `AudioChunk`
- Transcription via `Transcription`
- Memory traces via `MemoryTrace`

### 3.2 Optional Enhancements

#### 3.2.1 Audio Chunk Metadata
Add sentence ID to AudioChunk for easier association:

```go
// pkg/protocol/messages.go
type AudioChunk struct {
    ConversationID string
    Format         string
    Sequence       int32
    DurationMs     int32
    TrackSID       string
    Data           []byte
    IsLast         bool
    Timestamp      int64
    SentenceID     string  // NEW: Link to specific sentence
}
```

#### 3.2.2 Reasoning Format Option
Support inline reasoning tags for clients that prefer them:

```go
// Configuration option
Features: []string{
    "streaming",
    "reasoning_inline",  // Send reasoning as <reasoning> tags in content
}
```

### 3.3 API Changes (None Required)

Current REST and WebSocket APIs remain unchanged:
- `POST /api/v1/conversations/{id}/messages` - Send message, triggers response
- `GET /api/v1/conversations/{id}/sync/ws` - WebSocket sync
- `GET /api/v1/conversations/{id}/events` - SSE events

---

## 4. Frontend Changes

### 4.1 New Dependencies

```bash
cd frontend
npm i tailwindcss @tailwindcss/vite  idb react-virtuoso zustand
```

### 4.2 Directory Structure

```
frontend/src/
├── adapters/
│   ├── protocolAdapter.ts      # Protocol → ConversationStore mapping
│   └── vadLiveKitBridge.ts     # VAD Float32Array → LiveKit MediaStreamTrack
├── stores/
│   ├── conversationStore.ts    # Zustand normalized store
│   ├── audioStore.ts           # Audio refs and playback state
│   └── connectionStore.ts      # LiveKit connection state
├── components/
│   ├── atoms/                  # COPY from new-components (presentation only)
│   │   ├── AudioAddon.tsx
│   │   ├── ComplexAddons.tsx
│   │   ├── MemoryTraceAddon.tsx # NEW: Memory trace display
│   │   ├── MessageBubble.tsx
│   │   ├── InputSendButton.tsx
│   │   ├── LanguageFlag.tsx
│   │   ├── RecordingButtonForInput.tsx
│   │   ├── ResizableBarTextInput.tsx
│   │   └── ToggleSwitch.tsx
│   ├── molecules/              # COPY from new-components (presentation only)
│   │   ├── ChatBubble.tsx
│   │   ├── LanguageSelector.tsx
│   │   └── MicrophoneVAD.tsx
│   ├── organisms/              # NEW: Built using hooks + stores
│   │   ├── ChatWindow.tsx      # Uses useLiveKit, useMessages, useSync
│   │   ├── MessageList.tsx     # Uses ConversationStore, renders ChatBubble
│   │   ├── InputArea.tsx       # Uses VAD + LiveKit bridge for voice
│   │   ├── AssistantMessage.tsx # Wraps ChatBubble with tools + memory traces
│   │   ├── UserMessage.tsx     # Wraps ChatBubble with transcription
│   │   └── ResponseControls.tsx # Uses useLiveKit for stop/regenerate
│   └── legacy/                 # KEEP temporarily during migration
│       └── ...
├── contexts/
│   └── MessageContext.tsx      # KEEP during migration, remove at end
├── hooks/
│   ├── useAsync.ts
│   ├── useConversations.ts
│   ├── useDatabase.ts
│   ├── useLiveKit.ts
│   ├── useLiveQuery.ts
│   ├── useMessages.ts
│   ├── useSSE.ts
│   ├── useSync.ts
│   ├── useWebSocketSync.ts
│   ├── useConversationStore.ts # NEW: Zustand store hook
│   ├── useAudioManager.ts      # NEW: Audio storage with IndexedDB
│   └── useVAD.ts               # NEW: Silero VAD manager
├── utils/
│   ├── cls.ts                  # COPY from new-components
│   ├── sileroVAD.ts            # COPY from new-components
│   └── audioWorklet.ts         # NEW: AudioWorklet for VAD → LiveKit bridge
└── types/
    ├── components.ts           # COPY from new-components (component props)
    └── streaming.ts            # Branded types (MessageId, SentenceId, etc.)
```

### 4.3 State Management

**Migration approach**: Gradually migrate from `MessageContext` to Zustand normalized stores.

#### ConversationStore (Zustand)

```typescript
// stores/conversationStore.ts
import { create } from 'zustand';

interface ConversationStore {
  // Normalized entities
  messages: Record<MessageId, Message>;
  sentences: Record<SentenceId, MessageSentence>;
  toolCalls: Record<ToolCallId, ToolCall>;
  audioRefs: Record<AudioRefId, AudioRef>;
  memoryTraces: Record<MemoryTraceId, MemoryTrace>;

  // Relationships
  messagesBySentence: Record<MessageId, SentenceId[]>;
  toolCallsByMessage: Record<MessageId, ToolCallId[]>;
  memoryTracesByMessage: Record<MessageId, MemoryTraceId[]>;

  // Streaming state
  currentStreamingMessageId: MessageId | null;

  // Actions
  addMessage: (message: Message) => void;
  addSentence: (sentence: MessageSentence) => void;
  addToolCall: (toolCall: ToolCall) => void;
  updateToolCall: (id: ToolCallId, update: Partial<ToolCall>) => void;
  addMemoryTrace: (trace: MemoryTrace) => void;
}
```

The new organisms will consume data from:
- `useConversationStore()` - normalized state with selectors
- `useMessages()` - persisted messages from SQLite (hydration source)
- `useLiveKit()` - connection state, audio tracks
- `useVAD()` - voice activity detection state

---

## 5. Migration Phases

### Phase 1: Setup (Week 1)

**Goal**: Install dependencies, configure Tailwind, copy utilities

#### 1.1 Install Dependencies
```bash
cd frontend
npm install tailwindcss @tailwindcss/vite idb
```

#### 1.2 Configure Tailwind
1. Copy `tailwind.config.js` from `new-components/`
2. Copy theme variables from `new-components/src/index.css`
3. Update `vite.config.ts`:
```typescript
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  // ...
});
```
4. Import Tailwind in `main.tsx`:
```typescript
import './index.css'; // Add Tailwind directives here
```

#### 1.3 Copy Utilities
- Copy `new-components/src/utils/cls.ts` → `frontend/src/utils/cls.ts`
- Copy `new-components/src/types/components.ts` → `frontend/src/types/components.ts`

**Deliverables**:
- [ ] Tailwind working (test with a `className="bg-red-500"`)
- [ ] `cls()` utility available
- [ ] Component types available

#### 1.4 Copy flake.nix fetching of silero utility and onnx files

- Check out flake.nix on new-components and how it woks; make sure this also exists in the root repository's flake.nix and sets it up correctly on the working directory

### Phase 2: Component Migration (Week 3-5)

**Goal**: Replace current components with new architecture using existing hooks

**Strategy**: Copy presentation primitives (atoms/molecules), build NEW organisms that integrate with our existing hooks (useLiveKit, useWebSocketSync, etc.)

#### 2.1 Setup Tailwind CSS
```bash
npm install tailwindcss @tailwindcss/vite
```
- Copy `tailwind.config.js` from new-components
- Copy theme CSS variables from `new-components/src/index.css`
- Add Tailwind plugin to `vite.config.ts`

#### 2.2 Copy Presentation Components (Atoms)
Copy from `new-components/src/components/atoms/`:
- `AudioAddon.tsx` - Audio playback controls
- `ComplexAddons.tsx` - Tool icons + expandable details
- `MessageBubble.tsx` - Base bubble styling
- `InputSendButton.tsx` - Send button with states
- `LanguageFlag.tsx` - Language indicator
- `RecordingButtonForInput.tsx` - Voice recording button
- `ResizableBarTextInput.tsx` - Auto-resizing input
- `ToggleSwitch.tsx` - Toggle component

Also copy:
- `utils/cls.ts` - Class name utility

#### 2.3 Copy Presentation Components (Molecules)
Copy from `new-components/src/components/molecules/`:
- `ChatBubble.tsx` - Extends MessageBubble with addons
- `LanguageSelector.tsx` - Language picker dropdown
- `MicrophoneVAD.tsx` - Voice activity detection

#### 2.4 Build NEW Organisms (Using Our Hooks + Context)
Create from scratch, integrating existing hooks and MessageContext:

**MessageList.tsx**
```typescript
// Uses: useMessages (SQLite), useMessageContext (streaming)
// Renders: AssistantMessage, UserMessage
export function MessageList() {
  const { messages } = useMessages(conversationId);
  const { streamingMessages } = useMessageContext();

  return (
    <div className="flex flex-col gap-4 p-4">
      {messages.map(msg => (
        msg.role === 'assistant'
          ? <AssistantMessage key={msg.id} message={msg} />
          : <UserMessage key={msg.id} message={msg} />
      ))}
      {/* Render streaming message if active */}
      {Object.keys(streamingMessages).length > 0 && (
        <StreamingMessage />
      )}
    </div>
  );
}
```

**AssistantMessage.tsx**
```typescript
// Uses: useMessageContext (toolUsages, reasoningSteps)
// Renders: ChatBubble with ComplexAddons
export function AssistantMessage({ message }: { message: Message }) {
  const { toolUsages, reasoningSteps } = useMessageContext();

  // Filter tools for this message
  const messageTools = toolUsages.filter(t => t.request.messageId === message.id);

  return (
    <ChatBubble
      role="assistant"
      content={message.contents}
      addons={<ComplexAddons tools={messageTools} />}
      timestamp={message.createdAt}
    />
  );
}
```

**UserMessage.tsx**
```typescript
// Renders: ChatBubble with transcription indicator
export function UserMessage({ message }: { message: Message }) {
  return (
    <ChatBubble
      role="user"
      content={message.contents}
      timestamp={message.createdAt}
    />
  );
}
```

**InputArea.tsx**
```typescript
// Uses: useLiveKit (for audio), useMessageContext (isGenerating)
// Renders: ResizableBarTextInput, RecordingButtonForInput, InputSendButton
export function InputArea({ onSend }: { onSend: (text: string) => void }) {
  const { publishAudioTrack, isConnected } = useLiveKit();
  const { isGenerating } = useMessageContext();
  const [text, setText] = useState('');

  return (
    <div className="flex gap-2 p-4 border-t">
      <RecordingButtonForInput onStartRecording={() => publishAudioTrack()} />
      <ResizableBarTextInput
        value={text}
        onChange={setText}
        onSubmit={() => { onSend(text); setText(''); }}
        disabled={isGenerating}
      />
      <InputSendButton
        onClick={() => { onSend(text); setText(''); }}
        disabled={!isConnected || isGenerating || !text.trim()}
      />
    </div>
  );
}
```

**ChatWindow.tsx**
```typescript
// Uses: useLiveKit, useMessages, useWebSocketSync
// Renders: MessageList, InputArea, ResponseControls
export function ChatWindow({ conversationId }: { conversationId: string }) {
  const liveKit = useLiveKit(conversationId);
  const { syncStatus } = useWebSocketSync(conversationId);
  const { sendMessage } = useMessages(conversationId);

  return (
    <div className="flex flex-col h-full bg-surface-950">
      <MessageList conversationId={conversationId} />
      <ResponseControls />
      <InputArea onSend={sendMessage} />
    </div>
  );
}
```

#### 2.5 Delete Old Components
After organisms are working, delete:
- `components/MessageBubble.tsx` (old)
- `components/ToolUsageDisplay.tsx`
- `components/ProtocolDisplay.tsx`
- `components/MessageList.tsx` (old)

**Deliverables**:
- [ ] Tailwind configured and working
- [ ] All atoms copied and imports fixed
- [ ] All molecules copied and imports fixed
- [ ] MessageList organism with virtualization
- [ ] AssistantMessage with tool display
- [ ] UserMessage with transcription
- [ ] InputArea with voice support
- [ ] ChatWindow integrating all pieces
- [ ] Old components deleted

### Phase 3: Polish & Integration (Week 3-4)

**Goal**: Wire up all features, handle edge cases

#### 3.1 Streaming Display
- `StreamingMessage` component shows sentences as they arrive
- Typing cursor animation on incomplete response
- "Generating..." status indicator

#### 3.2 Tool Display
- Wire `ComplexAddons` to show tool status icons
- Expandable tool details (parameters, results)
- Status: pending → executing → success/error

#### 3.3 Reasoning Display
- Render `reasoningSteps` from MessageContext
- Blue-bordered collapsible blocks
- Show before main response content

#### 3.4 Error Display
- Keep existing `ErrorNotification` component
- Add error badges to messages if needed

**Deliverables**:
- [ ] Streaming messages render correctly
- [ ] Tools show with status icons
- [ ] Reasoning blocks visible
- [ ] Errors display properly

### Phase 4: Voice & Cleanup (Week 4-5)

**Goal**: Silero VAD integration with LiveKit streaming, memory traces display, legacy code removed

#### 4.1 VAD → LiveKit Audio Bridge

**Architecture**:
```
Microphone → Silero VAD (Float32Array frames)
           → AudioWorkletNode (process & forward)
           → MediaStreamAudioDestinationNode
           → MediaStreamTrack
           → LiveKit publishTrack()
           → Server-side transcription (unchanged)
```

**Implementation**:

```typescript
// adapters/vadLiveKitBridge.ts
export class VADLiveKitBridge {
  private audioContext: AudioContext;
  private destination: MediaStreamAudioDestinationNode;
  private workletNode: AudioWorkletNode | null = null;

  async initialize(): Promise<MediaStreamTrack> {
    this.audioContext = new AudioContext({ sampleRate: 16000 });
    this.destination = this.audioContext.createMediaStreamDestination();

    // Load AudioWorklet for efficient Float32Array → MediaStream conversion
    await this.audioContext.audioWorklet.addModule('/vad-processor.js');
    this.workletNode = new AudioWorkletNode(this.audioContext, 'vad-processor');
    this.workletNode.connect(this.destination);

    return this.destination.stream.getAudioTracks()[0];
  }

  // Called by Silero VAD onFrameProcessed callback
  pushAudioFrame(audioData: Float32Array): void {
    this.workletNode?.port.postMessage({ type: 'audio', data: audioData });
  }

  // Called by Silero VAD onSpeechEnd callback
  pushSpeechSegment(audioData: Float32Array): void {
    this.workletNode?.port.postMessage({ type: 'speech', data: audioData });
  }
}
```

```typescript
// hooks/useVAD.ts
export function useVAD(onTrackReady: (track: MediaStreamTrack) => void) {
  const bridgeRef = useRef<VADLiveKitBridge | null>(null);
  const vadRef = useRef<SileroVADManager | null>(null);

  const startVAD = useCallback(async () => {
    // Initialize bridge
    bridgeRef.current = new VADLiveKitBridge();
    const track = await bridgeRef.current.initialize();

    // Initialize Silero VAD with callbacks
    vadRef.current = new SileroVADManager({
      onFrameProcessed: (frame) => bridgeRef.current?.pushAudioFrame(frame),
      onSpeechStart: () => console.log('Speech started'),
      onSpeechEnd: (audio) => bridgeRef.current?.pushSpeechSegment(audio),
    });

    await vadRef.current.start();
    onTrackReady(track);
  }, [onTrackReady]);

  return { startVAD, stopVAD: () => vadRef.current?.stop() };
}
```

**Wire to useLiveKit**:
```typescript
// In InputArea.tsx
const { publishAudioTrack } = useLiveKit();
const { startVAD } = useVAD((track) => publishAudioTrack(track));

// MicrophoneVAD onClick triggers startVAD()
```

**Key benefit**: Server-side transcription flow unchanged - LiveKit receives audio stream as before, server sends `Transcription` protocol messages back.

#### 4.2 Memory Traces Display

**Add MemoryTraceAddon atom**:
```typescript
// components/atoms/MemoryTraceAddon.tsx
interface MemoryTraceAddonProps {
  traces: MemoryTrace[];
}

export function MemoryTraceAddon({ traces }: MemoryTraceAddonProps) {
  return (
    <div className="flex flex-wrap gap-1">
      {traces.map((trace) => (
        <Tooltip key={trace.id} content={trace.content}>
          <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-purple-100
                          text-purple-800 rounded-full text-xs">
            🧠 {(trace.relevance * 100).toFixed(0)}%
          </span>
        </Tooltip>
      ))}
    </div>
  );
}
```

**Integrate into AssistantMessage**:
```typescript
// In AssistantMessage.tsx
const { memoryTraces } = useMessageContext();
const messageTraces = memoryTraces.filter(t => t.messageId === message.id);

return (
  <ChatBubble
    role="assistant"
    content={message.contents}
    addons={
      <>
        <ComplexAddons tools={messageTools} />
        {messageTraces.length > 0 && <MemoryTraceAddon traces={messageTraces} />}
      </>
    }
  />
);
```

#### 4.3 Remove Legacy Components

Delete after new organisms are working:
1. `components/ToolUsageDisplay.tsx` + CSS + test
2. `components/ProtocolDisplay.tsx` + CSS + test
3. `components/MessageBubble.tsx` (old) + test
4. `components/ChatWindow.tsx` (old)
5. `components/MessageList.tsx` (old)
6. `components/AudioInput.tsx` (replaced by MicrophoneVAD)

**Deliverables**:
- [ ] VADLiveKitBridge adapter working
- [ ] useVAD hook with Silero integration
- [ ] MicrophoneVAD → LiveKit audio streaming
- [ ] Server transcription still works via protocol
- [ ] MemoryTraceAddon displaying per-message traces
- [ ] All legacy components deleted
- [ ] Tests updated for new components

### Phase 5: Testing & Quality

**Goal**: Comprehensive test coverage for new components, integration tests with visual regression, and code quality fixes

#### 5.1 Code Quality Fixes

Fix issues identified in code review before writing tests:

**5.1.1 Remove Dead Code in ChatWindowBridge**
- Remove unused props: `_conversation`, `_onConversationUpdate`, `_isSyncing`, `_lastSyncTime`
- Update `App.tsx` to not pass these dead props
- Replace `console.warn()` stubs with feature flag pattern or `onFeatureUnavailable` callback

**5.1.2 Fix Global State Leakage**
- Move module-level maps (`sentenceAudioMap`, `pendingSentences`, `cleanupTimers`) from `protocolAdapter.ts` and `sileroVAD.ts` to conversation-scoped context
- Pass `SentenceAudioContext` as parameter to `handleAudioChunk` and related functions
- Ensure maps are cleared when conversation changes

**5.1.3 Fix audioRefs Never Populated**
- In `protocolAdapter.ts` `handleAudioChunk`, call `store.addAudioRef()` after storing audio:
```typescript
const audioRef: AudioRef = {
  id: audioRefId,
  sizeBytes: chunk.data.byteLength,
  durationMs: chunk.durationMs,
  sampleRate: parseSampleRate(chunk.format),
};
store.addAudioRef(audioRef);
```

#### 5.2 Unit Tests for New Components

**Critical files requiring tests:**

1. **`adapters/protocolAdapter.test.ts`** (~300 lines)
   - `handleAssistantSentence` transforms protocol → store format
   - `handleAudioChunk` stores audio and associates with sentences
   - `handleToolUseRequest`/`handleToolUseResult` status mapping
   - Race condition: audio arrives before sentence
   - Race condition: sentence arrives before audio
   - Malformed audio format string handling

2. **`stores/conversationStore.test.ts`** (~200 lines)
   - Bidirectional reference integrity (message.sentenceIds ↔ sentences map)
   - Selector `getMessageSentences` returns correct sequence order
   - `loadConversation` clears old state
   - `updateMessageStatus` updates correctly
   - `addAudioRef` populates audioRefs map

3. **`components/molecules/ChatBubble.test.tsx`** (~250 lines)
   - Reasoning block parsing (`<reasoning>` tags)
   - Addon rendering (tools, audio, memory traces)
   - Role-based styling (user vs assistant)

4. **`components/atoms/ComplexAddons.test.tsx`** (~150 lines)
   - Tool status icon rendering (pending/executing/success/error)
   - Expandable details behavior
   - Click interactions

5. **`utils/sileroVAD.test.ts`** (~200 lines)
   - VAD initialization
   - Speech start/end callbacks
   - Cleanup of conversation-scoped state

#### 5.3 Integration Tests with Playwright + Screenshots

**WebSocket Mocking Strategy:**
- Use Playwright's `page.route()` to intercept WebSocket connections
- Fixture files provide canned protocol messages (AssistantSentence, ToolUseRequest, etc.)
- Tests trigger specific message sequences to test UI states

**Screenshot Strategy:**
- Screenshots generated fresh in CI, not committed to repo
- Visual comparison uses Playwright's `toHaveScreenshot()` with threshold
- Baseline images auto-generated on first run, stored in CI artifacts

**New e2e test files:**

1. **`e2e/message-display.spec.ts`** - Message rendering tests
   - Assistant message with tool calls renders correctly + screenshot
   - Streaming message shows typing indicator + screenshot
   - Reasoning blocks render with blue border + screenshot

2. **`e2e/audio-playback.spec.ts`** - Audio feature tests
   - Sentence audio playback controls visible + screenshot

3. **`e2e/memory-traces.spec.ts`** - Memory trace display
   - Memory trace badges show relevance percentage + screenshot

4. **`e2e/visual-regression.spec.ts`** - Screenshot comparison
   - Chat window matches baseline
   - Dark mode chat window matches baseline

**New test infrastructure files:**
```
frontend/
├── e2e/
│   ├── fixtures/
│   │   └── websocket-mock.ts    # Mock WebSocket for protocol messages
│   ├── message-display.spec.ts
│   ├── audio-playback.spec.ts
│   ├── memory-traces.spec.ts
│   └── visual-regression.spec.ts
└── test-results/                 # Generated in CI, gitignored
    └── screenshots/
```

#### 5.4 Deliverables

**Code Quality:**
- [ ] Remove unused props from ChatWindowBridge
- [ ] Update App.tsx to not pass dead props
- [ ] Scope sentenceAudioMap/pendingSentences to conversation
- [ ] Call addAudioRef when storing audio
- [ ] Replace console.warn with feature flags

**Unit Tests:**
- [ ] `protocolAdapter.test.ts` - Protocol transformation tests
- [ ] `conversationStore.test.ts` - State management tests
- [ ] `ChatBubble.test.tsx` - Reasoning/addon rendering
- [ ] `ComplexAddons.test.tsx` - Tool status display
- [ ] `sileroVAD.test.ts` - VAD lifecycle and cleanup

**Integration Tests:**
- [ ] WebSocket mock fixture created
- [ ] `message-display.spec.ts` - Tool calls, streaming, reasoning
- [ ] `audio-playback.spec.ts` - Audio controls
- [ ] `memory-traces.spec.ts` - Memory trace badges
- [ ] `visual-regression.spec.ts` - Screenshot baselines

---

## 6. Feature Implementation Details

### 6.1 Tool Usage Display

**Data Flow**:
```
ToolUseRequest (protocol)
  → handleToolUseRequest (adapter)
  → addToolCall({ status: 'pending' }) (store)
  → AssistantChatMessageInList reads toolCallIds
  → ChatBubble receives addons + tools props
  → ComplexAddons renders icons + expandable details
```

**UI Requirements**:
- Inline emoji icons: 🔧 (pending), ⚡ (executing), ✅ (success), ❌ (error)
- Tooltip on hover with tool name and status
- Click to expand full details (parameters, result)
- Animated indicator for running tools

### 6.2 Reasoning Steps Display

**Data Flow**:
```
ReasoningStep (protocol)
  → handleReasoningStep (adapter)
  → addSentence with sequence=-1 (store) OR
  → Embed as <reasoning> tag in message content
  → ChatBubble parses and renders blue blocks
```

**UI Requirements**:
- Blue-bordered block with "Reasoning" label
- Pre-wrapped whitespace for formatting
- Collapsible for long reasoning
- Sorted by sequence number

### 6.3 Audio Display & Playback

**Data Flow**:
```
AudioChunk (protocol)
  → handleAudioChunk (adapter)
  → audioManager.store(data) (IndexedDB)
  → addAudioRef + update sentence.audioRefId (store)
  → ChatBubble shows AudioAddon
  → Click plays from IndexedDB
```

**UI Requirements**:
- Compact mode: Small play button + duration
- Full mode: Progress bar + time + stop button
- Auto-play option for responses
- Queue management for sequential playback

### 6.4 Transcription Display

**Data Flow**:
```
Transcription (protocol)
  → handleTranscription (adapter)
  → If final: create user message with transcription field
  → If partial: update streaming transcription
  → UserChatMessageInList shows transcription as content
  → AudioAddon with 🎤 if user audio was recorded
```

**UI Requirements**:
- Show transcription as primary content
- Microphone icon indicates voice message
- Tooltip shows "Voice message: {transcription}"
- Option to play original audio

### 6.5 Streaming Responses

**Data Flow**:
```
StartAnswer (protocol)
  → addMessage with status='streaming' (store)
  → Clear previous streaming state

AssistantSentence (protocol)
  → handleAssistantSentence (adapter)
  → addSentence with sequence number
  → If isFinal: completeSentence, update message status='complete'

Message render:
  → Join sentences by sequence
  → Show cursor animation if streaming
  → Status badge in header
```

**UI Requirements**:
- Sentence-by-sentence appearance
- Typing cursor animation
- "Streaming" badge during generation
- Smooth transitions

### 6.6 Memory Traces Display

**Data Flow**:
```
MemoryTrace (protocol)
  → handleMemoryTrace (adapter)
  → addMemoryTrace to store with messageId association
  → AssistantMessage reads traces for this message
  → MemoryTraceAddon renders inline badges with tooltips
```

**UI Requirements**:
- Inline badge per trace with relevance percentage (e.g., "🧠 87%")
- Tooltip on hover shows full memory content
- Purple/indigo color scheme to distinguish from tools
- Expandable detail view for long memories
- Sorted by relevance score (highest first)

### 6.7 VAD Voice Input

**Data Flow**:
```
User clicks MicrophoneVAD
  → useVAD.startVAD()
  → SileroVADManager initializes with microphone
  → VADLiveKitBridge creates AudioWorklet pipeline
  → MediaStreamTrack returned to useLiveKit.publishAudioTrack()
  → LiveKit streams audio to server
  → Server sends Transcription protocol messages (unchanged)
  → MessageContext updates transcription state
  → On final transcription → create user Message
```

**UI Requirements**:
- Animated VAD indicator showing speech probability
- Visual feedback during speech detection
- Smooth transition between listening/speaking states
- Error handling for microphone permissions
- Loading state while Silero model initializes

---


## Appendix A: File Changes Summary

### New Files
```
frontend/src/
├── adapters/
│   ├── protocolAdapter.ts         # Protocol → store mapping
│   └── vadLiveKitBridge.ts        # VAD Float32Array → LiveKit MediaStreamTrack
├── stores/
│   ├── conversationStore.ts       # Zustand normalized store
│   ├── audioStore.ts              # Audio refs and playback state
│   └── connectionStore.ts         # LiveKit connection state
├── hooks/
│   ├── useConversationStore.ts    # Zustand store hook
│   ├── useAudioManager.ts         # IndexedDB audio storage
│   └── useVAD.ts                  # Silero VAD manager
├── components/
│   ├── atoms/
│   │   ├── AudioAddon.tsx
│   │   ├── ComplexAddons.tsx
│   │   ├── MemoryTraceAddon.tsx   # Memory trace display
│   │   ├── MessageBubble.tsx
│   │   ├── InputSendButton.tsx
│   │   ├── LanguageFlag.tsx
│   │   ├── RecordingButtonForInput.tsx
│   │   ├── ResizableBarTextInput.tsx
│   │   └── ToggleSwitch.tsx
│   ├── molecules/
│   │   ├── ChatBubble.tsx
│   │   ├── LanguageSelector.tsx
│   │   └── MicrophoneVAD.tsx
│   └── organisms/
│       ├── AssistantMessage.tsx
│       ├── UserMessage.tsx
│       ├── StreamingMessage.tsx
│       ├── MessageList.tsx
│       ├── InputArea.tsx
│       ├── ResponseControls.tsx
│       └── ChatWindow.tsx
├── utils/
│   ├── cls.ts                     # Class name utility
│   ├── sileroVAD.ts               # Silero VAD manager
│   └── audioWorklet.ts            # AudioWorklet processor
├── types/
│   ├── components.ts              # Component props
│   └── streaming.ts               # Branded types
└── public/
    └── vad-processor.js           # AudioWorklet for VAD bridge
```

### Modified Files
```
frontend/src/
├── hooks/
│   └── useLiveKit.ts              # Add VAD track support
├── App.tsx                        # Add store providers
├── vite.config.ts                 # Add Tailwind plugin
├── main.tsx                       # Import Tailwind CSS
└── index.css                      # Tailwind theme variables
```

### Deleted Files (Phase 4)
```
frontend/src/
├── contexts/
│   └── MessageContext.tsx         # Replaced by Zustand stores
└── components/
    ├── ToolUsageDisplay.tsx       # Replaced by ComplexAddons
    ├── ToolUsageDisplay.css
    ├── ToolUsageDisplay.test.tsx
    ├── ProtocolDisplay.tsx        # Integrated into message components
    ├── ProtocolDisplay.css
    ├── ProtocolDisplay.test.tsx
    ├── MessageBubble.tsx          # Replaced by new MessageBubble atom
    ├── MessageBubble.test.tsx
    ├── MessageList.tsx            # Replaced by new organism
    ├── ChatWindow.tsx             # Replaced by new organism
    └── AudioInput.tsx             # Replaced by MicrophoneVAD
```

---

## Appendix B: Type Conversions Reference

### ID Conversions
```typescript
// Old → New
string → MessageId(string)
string → SentenceId(string)
string → ToolCallId(string)
string → AudioRefId(string)
string → ConversationId(string)
```

### Status Conversions
```typescript
// Tool status
{ request, result: null } → { status: 'pending' }
{ request, result: null, executing: true } → { status: 'executing' }
{ request, result: { success: true } } → { status: 'success' }
{ request, result: { success: false } } → { status: 'error' }

// Message status
isStreaming: true → MessageStatus.Streaming
isStreaming: false → MessageStatus.Complete
error: string → MessageStatus.Error
```

### Audio Conversions
```typescript
// AudioChunk → AudioRef
{
  data: Uint8Array,
  durationMs: number,
  format: string,
  sequence: number,
}
→
{
  id: AudioRefId,
  sizeBytes: data.byteLength,
  durationMs: durationMs,
  sampleRate: parseFormat(format),
}
```

---

*Document Version: 3.0*
*Last Updated: 2025-12-29*
*Strategy: Zustand stores, VAD → LiveKit streaming, full new-components integration*
