# Frontend Component Architecture

This document describes the implementation of Alicia's frontend component system, which follows Atomic Design principles with a focus on reusability, composability, and integration with the optimization system.

## Overview

The frontend is structured in two main directories:

1. **`/home/usr/projects/alicia/frontend/src/`** - Main application with stores, hooks, and contexts
2. **`/home/usr/projects/alicia/frontend/new-components/src/`** - Atomic/Molecular/Organism component library

## Atomic Design Structure

Components are organized by complexity level:

```
frontend/new-components/src/components/
├── atoms/           # Smallest, indivisible components
├── molecules/       # Combinations of atoms
└── organisms/       # Complete, functional UI sections
```

### Design Principles

1. **Progressive Composition** - Atoms combine into molecules, molecules into organisms
2. **Single Responsibility** - Each component does one thing well
3. **Prop-Driven** - Components are controlled via props, not internal state
4. **Type Safety** - Full TypeScript support with strict types
5. **Storybook Integration** - Every component has a `.stories.tsx` file

## Atoms

Atoms are the fundamental building blocks - buttons, inputs, flags, bubbles.

### MessageBubble

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/MessageBubble.tsx`

The core message display component with type-based styling.

**Props**:
```typescript
interface MessageBubbleProps {
  type?: MessageType;           // 'user' | 'assistant' | 'system'
  content?: React.ReactNode;    // Message content
  state?: MessageState;         // 'completed' | 'typing' | 'error'
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

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/InputSendButton.tsx`

Send button with loading and disabled states.

### RecordingButtonForInput

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/RecordingButtonForInput.tsx`

Voice input recording button with visual feedback.

### ResizableBarTextInput

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/ResizableBarTextInput.tsx`

Auto-resizing textarea with max height constraints.

### ToggleSwitch

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/ToggleSwitch.tsx`

Animated toggle switch for settings.

### LanguageFlag

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/LanguageFlag.tsx`

Language flag display using country code emojis.

### AudioAddon

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/AudioAddon.tsx`

Audio playback control addon.

### ComplexAddons

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/ComplexAddons.tsx`

Container for multiple addon types (tools, memories, etc.).

## Molecules

Molecules combine atoms into functional units.

### ChatBubble

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/molecules/ChatBubble.tsx`

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

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/molecules/LanguageSelector.tsx`

Dropdown language selector with flag display.

### MicrophoneVAD

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/molecules/MicrophoneVAD.tsx`

Voice activity detection microphone with visual feedback.

**Features**:
- Real-time VAD using `@ricky0123/vad-web`
- Waveform visualization
- Auto-stop on silence
- Integrates with audio recording hooks

## Organisms

Organisms are complete, self-contained UI sections that combine molecules and atoms.

### InputMessageChatComponent

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/organisms/InputMessageChatComponent.tsx`

Complete message input interface with text, voice, and send controls.

**Props**:
```typescript
interface InputMessageChatComponentProps {
  onSendMessage: (message: string) => void;
  onVoiceInput?: (audio: Blob) => void;
  placeholder?: string;
  disabled?: boolean;
  showVoiceButton?: boolean;
}
```

**Features**:
- Resizable text input
- Voice recording toggle
- Send button with validation
- Loading states
- Keyboard shortcuts (Enter to send, Shift+Enter for newline)

### UserChatMessageInList

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/organisms/UserChatMessageInList.tsx`

User message display in conversation list context.

**Features**:
- Avatar display
- Timestamp
- Edit/delete actions (if enabled)
- Audio playback (if message has audio)

### AssistantChatMessageInList

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/organisms/AssistantChatMessageInList.tsx`

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

### ChatPage

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/organisms/ChatPage.tsx`

Complete chat interface layout.

**Features**:
- Message list with virtualization
- Input component
- Scroll-to-bottom
- Loading states
- Empty state display

### ChatPageWithAPI

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/components/organisms/ChatPageWithAPI.tsx`

ChatPage integrated with backend API and stores.

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
  targetType: 'message' | 'tool_use' | 'memory' | 'reasoning';
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
    description: 'Equal emphasis on all dimensions'
  }
];
```

**Actions**:
```typescript
setWeights(weights: DimensionWeights)
updateScores(scores: DimensionScores)
applyPreset(presetId: PresetId)
resetToDefaults()
```

**Usage**:
```typescript
import { useDimensionStore } from '@/stores/dimensionStore';

const dimensionStore = useDimensionStore();

// Apply preset
dimensionStore.applyPreset('accuracy');

// Custom weights
dimensionStore.setWeights({
  successRate: 0.3,
  quality: 0.25,
  efficiency: 0.15,
  robustness: 0.15,
  generalization: 0.1,
  diversity: 0.05,
  innovation: 0.0
});

// Update scores from SSE stream
dimensionStore.updateScores(event.dimension_scores);
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

## Hooks

**Location**: `/home/usr/projects/alicia/frontend/src/hooks/`

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

### Custom Component Hooks

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/hooks/`

#### useChat

Complete chat functionality hook.

```typescript
const {
  messages,
  sendMessage,
  sendVoiceMessage,
  loading,
  error,
  scrollToBottom
} = useChat(conversationId);
```

#### useRealtimeChat

Real-time chat with SSE integration.

```typescript
const {
  messages,
  sendMessage,
  streamingMessage,
  isStreaming
} = useRealtimeChat(conversationId);
```

#### useAudioPlayback

Audio playback control.

```typescript
const {
  playing,
  progress,
  play,
  pause,
  seek
} = useAudioPlayback(audioUrl);
```

## Integration with Optimization System

### Voting Integration

Components integrate with the feedback system:

```tsx
// In AssistantChatMessageInList
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
// In AssistantChatMessageInList
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

### Design Tokens

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/utils/constants.ts`

```typescript
export const CSS = {
  // Colors
  bgPrimary: 'bg-primary-dark',
  bgSecondary: 'bg-secondary-darker',
  bgMessageSent: 'bg-message-sent',
  bgMessageReceived: 'bg-message-received',
  textPrimary: 'text-primary-light',
  textMuted: 'text-primary-muted',

  // Spacing
  px4: 'px-4',
  py3: 'py-3',
  gap2: 'gap-2',

  // Typography
  textSm: 'text-sm',
  textXs: 'text-xs',

  // Layout
  flex: 'flex',
  flexCol: 'flex-col',
  itemsCenter: 'items-center',
  justifyBetween: 'justify-between',

  // Animations
  transitionAll: 'transition-all',
  duration300: 'duration-300',
  animateBounce: 'animate-bounce',
};
```

### Utility Functions

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/utils/cls.ts`

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

## Storybook Development

Every component has a Storybook story for isolated development and testing.

**Example**: `/home/usr/projects/alicia/frontend/new-components/src/components/atoms/MessageBubble.stories.tsx`

```tsx
import type { Meta, StoryObj } from '@storybook/react';
import MessageBubble from './MessageBubble';
import { MESSAGE_TYPES, MESSAGE_STATES } from '~/mockData';

const meta: Meta<typeof MessageBubble> = {
  title: 'Atoms/MessageBubble',
  component: MessageBubble,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof MessageBubble>;

export const UserMessage: Story = {
  args: {
    type: MESSAGE_TYPES.USER,
    content: 'Hello, how are you?',
    timestamp: new Date(),
  },
};

export const AssistantMessage: Story = {
  args: {
    type: MESSAGE_TYPES.ASSISTANT,
    content: 'I\'m doing well, thank you!',
    timestamp: new Date(),
  },
};

export const Typing: Story = {
  args: {
    type: MESSAGE_TYPES.ASSISTANT,
    state: MESSAGE_STATES.TYPING,
    showTyping: true,
  },
};
```

### Running Storybook

```bash
cd frontend/new-components
npm run storybook
```

Access at `http://localhost:6006`

## Type Definitions

**Location**: `/home/usr/projects/alicia/frontend/new-components/src/types/components.ts`

```typescript
export type MessageType = 'user' | 'assistant' | 'system';
export type MessageState = 'completed' | 'typing' | 'error' | 'sending';

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
  type: MessageType;
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
4. Create Storybook story: `atoms/NewComponent.stories.tsx`
5. Export from `atoms/index.ts`

### Adding a New Molecule

1. Identify which atoms to compose
2. Create molecule file: `molecules/NewMolecule.tsx`
3. Import required atoms
4. Define combined props interface
5. Implement composition logic
6. Create Storybook story
7. Write integration tests if needed

### Adding a New Organism

1. Identify required molecules and atoms
2. Create organism file: `organisms/NewOrganism.tsx`
3. Integrate with stores and hooks
4. Implement business logic
5. Create Storybook story with mock data
6. Add to ChatPageWithAPI if relevant

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

### Store Tests

Test Zustand stores:

```typescript
import { renderHook, act } from '@testing-library/react';
import { useFeedbackStore } from './feedbackStore';

describe('feedbackStore', () => {
  it('adds vote correctly', () => {
    const { result } = renderHook(() => useFeedbackStore());

    act(() => {
      result.current.addVote('message', 'msg_1', 'up');
    });

    const vote = result.current.getVote('message', 'msg_1');
    expect(vote.vote).toBe('up');
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
      <AssistantChatMessageInList message={messages[index]} />
    </div>
  )}
</FixedSizeList>
```

### Memoization

Use React.memo for expensive components:

```tsx
export const AssistantChatMessageInList = React.memo(({ message, ...props }) => {
  // Component implementation
}, (prevProps, nextProps) => {
  // Custom comparison
  return prevProps.message.id === nextProps.message.id &&
         prevProps.message.content === nextProps.message.content;
});
```

### Store Selectors

Use Zustand selectors to avoid unnecessary re-renders:

```tsx
// Bad - component re-renders on any store change
const store = useFeedbackStore();

// Good - only re-renders when votes change
const votes = useFeedbackStore(state => state.votes);
```

## Related Documentation

- `/home/usr/projects/alicia/docs/OPTIMIZATION_SYSTEM.md` - Backend optimization system
- `/home/usr/projects/alicia/docs/PHASE_6_INTEGRATION.md` - Phase 6 integration details
- `/home/usr/projects/alicia/docs/COMPONENTS.md` - Backend component architecture
- Storybook documentation at `http://localhost:6006`

## See Also

- [Protocol Specification](protocol/index.md) - Client-server protocol
- [Android App](ANDROID.md) - Native mobile implementation
- [Architecture Overview](ARCHITECTURE.md) - System context
