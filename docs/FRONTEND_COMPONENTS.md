# Frontend Component Architecture

This document describes the implementation of Alicia's frontend component system, which follows Atomic Design principles with a focus on reusability, composability, and integration with the optimization system.

## Overview

The frontend is structured with components, stores, hooks, and contexts:

**`/home/usr/projects/alicia/frontend/src/`** - Main application structure

## Atomic Design Structure

Components are organized by complexity level:

```
frontend/src/components/
├── atoms/           # Smallest, indivisible components
├── molecules/       # Combinations of atoms
└── organisms/       # Complete, functional UI sections
```

### Design Principles

1. **Progressive Composition** - Atoms combine into molecules, molecules into organisms
2. **Single Responsibility** - Each component does one thing well
3. **Prop-Driven** - Components are controlled via props, not internal state
4. **Type Safety** - Full TypeScript support with strict types
5. **Testing** - Components have corresponding `.test.tsx` files for quality assurance

## Atoms

Atoms are the fundamental building blocks - buttons, inputs, flags, bubbles.

### MessageBubble

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/MessageBubble.tsx`

The core message display component with type-based styling.

**Props**:
```typescript
interface MessageBubbleProps {
  type?: MessageRole;           // 'user' | 'assistant' | 'system'
  content?: React.ReactNode;    // Message content
  state?: MessageState;         // 'idle' | 'typing' | 'sending' | 'streaming' | 'completed' | 'error'
  timestamp?: Date;
  showTyping?: boolean;
  addons?: MessageAddon[];      // Icons, badges, etc.
  hideTimestamp?: boolean;
  className?: string;
}
```

**Features**:
- Type-specific styling (user: right-aligned blue, assistant: left-aligned gray, system: centered)
- Markdown-like formatting (`**bold**`, `*italic*`, `` `code` ``)
- Typing indicator animation
- Error state display
- Addon icon display with tooltips

**Example**:
```tsx
<MessageBubble
  type={MESSAGE_TYPES.ASSISTANT}
  content="Here's the answer you requested."
  timestamp={new Date()}
  addons={[
    { id: '1', emoji: '🔧', tooltip: 'Used search tool', position: 'inline' }
  ]}
/>
```

### InputSendButton

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/InputSendButton.tsx`

Send button with loading and disabled states.

### RecordingButtonForInput

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/RecordingButtonForInput.tsx`

Voice input recording button with visual feedback.

### ResizableBarTextInput

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/ResizableBarTextInput.tsx`

Auto-resizing textarea with max height constraints.

### ToggleSwitch

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/ToggleSwitch.tsx`

Animated toggle switch for settings.

### LanguageFlag

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/LanguageFlag.tsx`

Language flag display using country code emojis.

### AudioAddon

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/AudioAddon.tsx`

Audio playback control addon.

### ComplexAddons

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/ComplexAddons.tsx`

Container for multiple addon types (tools, memories, etc.).

### BranchNavigator

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/BranchNavigator.tsx`

Navigation controls for message branching/versioning.

### FeedbackControls

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/FeedbackControls.tsx`

Vote buttons (up/down/critical) for feedback submission.

### FeedbackPopover

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/FeedbackPopover.tsx`

Popover UI for detailed feedback with quick feedback options.

### Badge Components

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/`

- **ScoreBadge** - Display score values with color coding
- **CountBadge** - Display count indicators
- **StatusBadge** - Display status indicators
- **SyncStatusBadge** - Display synchronization status

### VoiceVisualizer

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/VoiceVisualizer.tsx`

Waveform visualization for voice input/output.

### Collapsible

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/Collapsible.tsx`

Expandable/collapsible content container.

### AlertDialog

**Location**: `/home/usr/projects/alicia/frontend/src/components/atoms/AlertDialog.tsx`

Modal dialog for alerts and confirmations.

## Molecules

Molecules combine atoms into functional units.

### ChatBubble

**Location**: `/home/usr/projects/alicia/frontend/src/components/molecules/ChatBubble.tsx`

Enhanced MessageBubble with voting, tool use displays, and memory references.

**Props**:
```typescript
interface ChatBubbleProps extends MessageBubbleProps {
  showVoting?: boolean;
  onVote?: (vote: VoteType) => void;
  toolCalls?: ToolCall[];
  memories?: MemoryReference[];
}
```

**Features**:
- Includes voting buttons (up/down/critical)
- Displays tool usage badges
- Shows memory retrieval indicators
- Integrates with feedback store

**Example**:
```tsx
<ChatBubble
  type={MESSAGE_TYPES.ASSISTANT}
  content="Based on your previous preferences..."
  showVoting={true}
  onVote={(vote) => feedbackStore.addVote('message', msgId, vote)}
  memories={[
    { id: 'mem_1', content: 'User prefers concise answers', relevance: 0.9 }
  ]}
/>
```

### LanguageSelector

**Location**: `/home/usr/projects/alicia/frontend/src/components/molecules/LanguageSelector.tsx`

Dropdown language selector with flag display.

### MicrophoneVAD

**Location**: `/home/usr/projects/alicia/frontend/src/components/molecules/MicrophoneVAD.tsx`

Voice activity detection microphone with visual feedback.

**Features**:
- Real-time VAD using `@ricky0123/vad-web`
- Waveform visualization
- Auto-stop on silence
- Integrates with audio recording hooks

## Organisms

Organisms are complete, self-contained UI sections that combine molecules and atoms.

### InputArea

**Location**: `/home/usr/projects/alicia/frontend/src/components/organisms/InputArea.tsx`

Complete message input interface with text, voice, and send controls.

**Features**:
- Resizable text input
- Voice recording toggle
- Send button with validation
- Loading states
- Keyboard shortcuts (Enter to send, Shift+Enter for newline)

### UserMessage

**Location**: `/home/usr/projects/alicia/frontend/src/components/organisms/UserMessage.tsx`

User message display in conversation list context.

**Features**:
- Avatar display
- Timestamp
- Edit/delete actions (if enabled)
- Audio playback (if message has audio)

### AssistantMessage

**Location**: `/home/usr/projects/alicia/frontend/src/components/organisms/AssistantMessage.tsx`

Assistant message with full voting, tool displays, and memory indicators.

**Props**:
```typescript
interface AssistantChatMessageProps {
  message: Message;
  showVoting?: boolean;
  showTools?: boolean;
  showMemories?: boolean;
  onVote?: (vote: VoteType, quickFeedback?: string) => void;
  onToolClick?: (toolCall: ToolCall) => void;
  onMemoryClick?: (memory: MemoryReference) => void;
}
```

**Features**:
- Voting interface with quick feedback options
- Tool usage display with parameters
- Memory reference display with relevance scores
- Copy-to-clipboard functionality
- Markdown rendering

### ChatWindow

**Location**: `/home/usr/projects/alicia/frontend/src/components/organisms/ChatWindow.tsx`

Complete chat interface layout.

**Features**:
- Message list with virtualization
- Input component
- Scroll-to-bottom
- Loading states
- Empty state display

### ChatWindowBridge

**Location**: `/home/usr/projects/alicia/frontend/src/components/organisms/ChatWindowBridge.tsx`

ChatWindow integrated with backend API and stores.

**Features**:
- Connects to conversation store
- Handles message sending via API
- Manages loading/error states
- Auto-scrolls on new messages
- Integrates voting with feedback store

## State Management (Stores)

**Location**: `/home/usr/projects/alicia/frontend/src/stores/`

All stores use Zustand with immer middleware for immutable updates.

### feedbackStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/feedbackStore.ts`

Manages user votes and feedback.

**State**:
```typescript
interface FeedbackStoreState {
  votes: Record<string, Vote>;
  aggregates: Record<string, VoteAggregates>;
}

interface Vote {
  id: string;
  targetType: 'message' | 'tool_use' | 'memory' | 'reasoning' | 'memory_usage' | 'memory_extraction';
  targetId: string;
  vote: 'up' | 'down' | 'critical';
  quickFeedback?: string;
  timestamp: number;
}
```

**Actions**:
```typescript
addVote(targetType, targetId, vote, quickFeedback?)
removeVote(targetType, targetId)
getVote(targetType, targetId)
setAggregates(targetType, targetId, aggregates)
getAggregates(targetType, targetId)
clearFeedback()
```

**Usage**:
```typescript
import { useFeedbackStore } from '@/stores/feedbackStore';

const feedbackStore = useFeedbackStore();

// Add vote
feedbackStore.addVote('message', 'msg_123', 'down', 'too_verbose');

// Get vote
const vote = feedbackStore.getVote('message', 'msg_123');

// Get aggregates (from server)
const aggregates = feedbackStore.getAggregates('message', 'msg_123');
// → { upvotes: 5, downvotes: 2, special: { critical: 1 } }
```

### dimensionStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/dimensionStore.ts`

Manages GEPA dimension weights and scores.

**State**:
```typescript
interface DimensionStoreState {
  weights: DimensionWeights;
  currentScores?: DimensionScores;
  presets: PivotPreset[];
  activePresetId?: PresetId;
}

interface DimensionWeights {
  successRate: number;
  quality: number;
  efficiency: number;
  robustness: number;
  generalization: number;
  diversity: number;
  innovation: number;
}
```

**Presets**:
```typescript
const PIVOT_PRESETS = [
  {
    id: 'accuracy',
    label: 'Accurate',
    icon: '✓',
    weights: { successRate: 0.4, quality: 0.25, ... },
    description: 'Prioritize correct answers over speed'
  },
  {
    id: 'speed',
    label: 'Fast',
    icon: '⚡',
    weights: { efficiency: 0.35, successRate: 0.2, ... },
    description: 'Quick responses with reasonable accuracy'
  },
  {
    id: 'reliable',
    label: 'Reliable',
    icon: '🛡️',
    weights: { robustness: 0.3, successRate: 0.25, ... },
    description: 'Consistent results across different inputs'
  },
  {
    id: 'creative',
    label: 'Creative',
    icon: '🎨',
    weights: { diversity: 0.2, innovation: 0.15, ... },
    description: 'Novel approaches and varied solutions'
  },
  {
    id: 'balanced',
    label: 'Balanced',
    icon: '⚖️',
    weights: { successRate: 0.25, quality: 0.2, ... },
    description: 'Moderate emphasis favoring success and quality'
  }
];
```

**Actions**:
```typescript
setCustomWeights(weights: DimensionWeights)
setPreset(presetId: PresetId)
resetToBalanced()
```

**Usage**:
```typescript
import { useDimensionStore } from '@/stores/dimensionStore';

const dimensionStore = useDimensionStore();

// Apply preset
dimensionStore.setPreset('accuracy');

// Custom weights
dimensionStore.setCustomWeights({
  successRate: 0.3,
  quality: 0.25,
  efficiency: 0.15,
  robustness: 0.15,
  generalization: 0.1,
  diversity: 0.05,
  innovation: 0.0
});
```

### memoryStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/memoryStore.ts`

Manages memory creation, retrieval, and display.

**Actions**:
```typescript
createMemory(content, tags, importance)
searchMemories(query, threshold, limit)
updateMemory(id, updates)
deleteMemory(id)
getRelevantMemories(context)
```

### conversationStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/conversationStore.ts`

Manages conversation state and message history.

**Actions**:
```typescript
createConversation(title)
loadConversation(id)
addMessage(conversationId, message)
updateMessage(id, updates)
deleteMessage(id)
archiveConversation(id)
```

### audioStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/audioStore.ts`

Manages audio recording and playback state.

### notesStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/notesStore.ts`

Manages user notes and annotations.

### serverInfoStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/serverInfoStore.ts`

Tracks server capabilities and configuration.

### connectionStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/connectionStore.ts`

Manages WebSocket and SSE connection states.

### branchStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/branchStore.ts`

Manages message branching and versioning.

**Actions**:
```typescript
createBranch(messageId, content)
switchBranch(branchId)
mergeBranch(branchId)
deleteBranch(branchId)
getBranches(messageId)
```

### sidebarStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/sidebarStore.ts`

Manages sidebar state (open/closed, active panel, etc.).

**Actions**:
```typescript
toggleSidebar()
setSidebarOpen(isOpen)
setActivePanel(panelId)
```

### toastStore

**Location**: `/home/usr/projects/alicia/frontend/src/stores/toastStore.ts`

Manages toast notification display.

**Actions**:
```typescript
showToast(message, type, duration)
hideToast(id)
clearAllToasts()
```

## Hooks

**Location**: `/home/usr/projects/alicia/frontend/src/hooks/`

**Note**: The hooks `useSSE`, `useSync`, `useDatabase`, and `useVAD` are internal implementation details and not exported from the hooks module for public use.

### useMessages

Manages message sending and retrieval.

```typescript
const { messages, sendMessage, loading, error } = useMessages(conversationId);
```

### useConversations

Manages conversation list and operations.

```typescript
const { conversations, createConversation, loadConversation } = useConversations();
```

### useSSE

Server-Sent Events connection hook.

```typescript
const { connect, disconnect, lastEvent } = useSSE(url, options);
```

### useSync

WebSocket-based real-time sync.

```typescript
const { syncState, lastUpdate } = useSync(conversationId);
```

### useDatabase (IndexedDB)

Local database operations for offline support.

```typescript
const { db, ready, error } = useDatabase();
```

### useVAD

Voice activity detection hook.

```typescript
const { listening, userSpeaking, start, stop } = useVAD(options);
```

### useFeedback

**Location**: `/home/usr/projects/alicia/frontend/src/hooks/useFeedback.ts`

Handles feedback/voting operations for any votable entity.

```typescript
const {
  vote,
  isVoting,
  submitVote,
  removeVote,
  aggregates
} = useFeedback(targetType, targetId);
```

**Features**:
- Automatic vote submission to backend
- Local state synchronization with feedbackStore
- Vote aggregates fetching and caching
- Optimistic updates

### useBranchStore

Branch navigation and management hook.

```typescript
const {
  currentBranch,
  branches,
  switchBranch,
  createBranch
} = useBranchStore(messageId);
```

### useAudioManager

**Location**: `/home/usr/projects/alicia/frontend/src/hooks/useAudioManager.ts`

Audio playback and recording management.

```typescript
const {
  recording,
  playing,
  currentAudioId,
  startRecording,
  stopRecording,
  playAudio,
  pauseAudio,
  stopAudio
} = useAudioManager();
```

**Features**:
- Recording state management
- Playback controls
- Audio queue management
- Integration with audioStore

## Integration with Optimization System

### Voting Integration

Components integrate with the feedback system:

```tsx
// In AssistantMessage
const handleVote = async (vote: VoteType, quickFeedback?: string) => {
  // Update local store
  feedbackStore.addVote('message', message.id, vote, quickFeedback);

  // Submit to backend
  try {
    const response = await fetch('/api/v1/feedback', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        target_type: 'message',
        target_id: message.id,
        vote,
        quick_feedback: quickFeedback
      })
    });

    const result = await response.json();

    // Update dimension weights in store
    dimensionStore.setWeights(result.new_weights);
  } catch (error) {
    console.error('Failed to submit feedback:', error);
  }
};
```

### Streaming Optimization Progress

```tsx
// Monitor optimization progress
useEffect(() => {
  if (!optimizationRunId) return;

  const eventSource = new EventSource(
    `/api/v1/optimizations/${optimizationRunId}/stream`
  );

  eventSource.addEventListener('message', (event) => {
    const data = JSON.parse(event.data);

    if (data.type === 'progress') {
      dimensionStore.updateScores(data.dimension_scores);
      setProgress({
        iteration: data.iteration,
        maxIterations: data.max_iterations,
        currentScore: data.current_score,
        bestScore: data.best_score
      });
    } else if (data.type === 'completed') {
      eventSource.close();
      setOptimizationComplete(true);
    }
  });

  return () => eventSource.close();
}, [optimizationRunId]);
```

### Memory Display

```tsx
// In AssistantMessage
{message.memories && message.memories.length > 0 && (
  <div className="memory-references">
    {message.memories.map(memory => (
      <div key={memory.id} className="memory-badge">
        <span className="memory-icon">💭</span>
        <span className="memory-content">{memory.content}</span>
        <span className="memory-score">{(memory.relevance * 100).toFixed(0)}%</span>
      </div>
    ))}
  </div>
)}
```

## Styling

All components use Tailwind CSS with a custom design system.

### Utility Functions

**Location**: `/home/usr/projects/alicia/frontend/src/utils/cls.ts`

```typescript
// Class name combiner
export function cls(classes: (string | boolean | undefined)[]): string {
  return classes.filter(Boolean).join(' ');
}

// Usage
const className = cls([
  'base-class',
  isActive && 'active-class',
  isDisabled && 'disabled-class',
  customClass
]);
```

## Type Definitions

**Location**: `/home/usr/projects/alicia/frontend/src/types/components.ts`

```typescript
export type MessageRole = 'user' | 'assistant' | 'system';
export type MessageState = 'idle' | 'typing' | 'sending' | 'streaming' | 'completed' | 'error';

export interface BaseComponentProps {
  className?: string;
}

export interface MessageAddon {
  id: string;
  emoji: string;
  tooltip: string;
  position?: 'inline' | 'block';
  onClick?: () => void;
}

export interface ToolCall {
  id: string;
  name: string;
  parameters: Record<string, any>;
  result?: any;
  status: 'pending' | 'success' | 'error';
}

export interface MemoryReference {
  id: string;
  content: string;
  relevance: number;
  tags?: string[];
}

export interface Message {
  id: string;
  conversationId: string;
  type: MessageRole;
  content: string;
  timestamp: Date;
  toolCalls?: ToolCall[];
  memories?: MemoryReference[];
  audioId?: string;
}
```

## Extending the Component Library

### Adding a New Atom

1. Create component file: `atoms/NewComponent.tsx`
2. Define props interface extending `BaseComponentProps`
3. Implement component with TypeScript
4. Write tests: `atoms/NewComponent.test.tsx`
5. Export from `atoms/index.ts`

### Adding a New Molecule

1. Identify which atoms to compose
2. Create molecule file: `molecules/NewMolecule.tsx`
3. Import required atoms
4. Define combined props interface
5. Implement composition logic
6. Write integration tests if needed

### Adding a New Organism

1. Identify required molecules and atoms
2. Create organism file: `organisms/NewOrganism.tsx`
3. Integrate with stores and hooks
4. Implement business logic
5. Write integration tests if needed
6. Add to ChatWindowBridge if relevant

## Testing Strategy

### Component Tests

Use React Testing Library for component tests:

```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import MessageBubble from './MessageBubble';

describe('MessageBubble', () => {
  it('renders user message correctly', () => {
    render(
      <MessageBubble
        type="user"
        content="Test message"
        timestamp={new Date()}
      />
    );

    expect(screen.getByText('Test message')).toBeInTheDocument();
  });

  it('shows typing indicator when typing', () => {
    render(
      <MessageBubble
        type="assistant"
        state="typing"
        showTyping={true}
      />
    );

    expect(screen.getAllByRole('status')).toHaveLength(3); // 3 dots
  });
});
```

### Hook Tests

Test custom hooks with React Testing Library:

```typescript
import { renderHook, act } from '@testing-library/react';
import { useMessages } from './useMessages';

describe('useMessages', () => {
  it('loads messages correctly', async () => {
    const { result } = renderHook(() => useMessages('conv_1'));

    await act(async () => {
      await result.current.loadMessages();
    });

    expect(result.current.messages).toHaveLength(5);
  });
});
```

## Performance Considerations

### Message List Virtualization

For long conversation histories, use virtualization:

```tsx
import { FixedSizeList } from 'react-window';

<FixedSizeList
  height={600}
  itemCount={messages.length}
  itemSize={100}
  width="100%"
>
  {({ index, style }) => (
    <div style={style}>
      <AssistantMessage message={messages[index]} />
    </div>
  )}
</FixedSizeList>
```

### Memoization

Use React.memo for expensive components:

```tsx
export const AssistantMessage = React.memo(({ message, ...props }) => {
  // Component implementation
}, (prevProps, nextProps) => {
  // Custom comparison
  return prevProps.message.id === nextProps.message.id &&
         prevProps.message.content === nextProps.message.content;
});
```

### Selective State Access

Access only needed state to avoid unnecessary re-renders:

```tsx
// Bad - component re-renders on any state change
const conversationStore = useConversationStore();

// Good - only re-renders when conversations change
const conversations = useConversationStore(state => state.conversations);
```

## Related Documentation

- `/home/usr/projects/alicia/docs/OPTIMIZATION_SYSTEM.md` - Backend optimization system
- `/home/usr/projects/alicia/docs/COMPONENTS.md` - Backend component architecture

## See Also

- [Protocol Specification](protocol/index.md) - Client-server protocol
- [Android App](ANDROID.md) - Native mobile implementation
- [Architecture Overview](ARCHITECTURE.md) - System context
