# Frontend UX Enhancement Plan

## Vision

Transform the Alicia frontend into a comprehensive AI interaction platform that empowers users to actively participate in improving AI responses through memory management, feedback mechanisms, and annotation capabilities.

---

## Current State Analysis

### Existing Atoms & Molecules
- **MessageBubble**: Individual message display with tool usage
- **InputBar**: Text input component
- **AudioInput/Output**: Voice interaction atoms
- **ToolUsageDisplay**: Expandable tool details
- **ProtocolDisplay**: Shows reasoning, memories, tools, errors
- **ResponseControls**: Stop/Regenerate controls

### Current Gaps
- No voting/rating system
- No user notes/annotations
- Memory traces are read-only (display only)
- No message-level feedback mechanisms
- Limited server information exposure

---

## Proposed Component Architecture

### Atomic Design System

```
atoms/
├── Button/
│   ├── IconButton.tsx        # Compact icon-only buttons
│   ├── PrimaryButton.tsx     # Main action buttons
│   └── GhostButton.tsx       # Subtle action buttons
├── Badge/
│   ├── StatusBadge.tsx       # Connection, sync status
│   ├── ScoreBadge.tsx        # Relevance scores, ratings
│   └── CountBadge.tsx        # Vote counts, note counts
├── Input/
│   ├── TextInput.tsx         # Single line input
│   ├── TextArea.tsx          # Multi-line for notes
│   └── SearchInput.tsx       # With search icon
├── Card/
│   ├── BaseCard.tsx          # Container with shadow/border
│   └── CollapsibleCard.tsx   # Expandable content
├── Icon/
│   ├── ThumbsUp.tsx
│   ├── ThumbsDown.tsx
│   ├── Note.tsx
│   ├── Memory.tsx
│   ├── Star.tsx
│   └── Edit.tsx
└── Feedback/
    ├── Tooltip.tsx           # Hover information
    ├── Toast.tsx             # Notifications
    └── ProgressBar.tsx       # Loading states

molecules/
├── VoteControl/
│   ├── VoteControl.tsx       # Unified voting component (all target types)
│   ├── VoteButton.tsx        # Single vote button with animation
│   ├── VoteCount.tsx         # Animated count display
│   └── QuickFeedback.tsx     # Quick feedback chip selector
├── ToolFeedback/
│   ├── ToolUseVoting.tsx     # Vote on tool usage decisions
│   ├── ToolUseCard.tsx       # Enhanced tool display with voting
│   └── ToolFeedbackChips.tsx # Wrong tool, wrong params, etc.
├── MemoryFeedback/
│   ├── MemoryVoting.tsx      # Vote on memory relevance
│   ├── MemoryCard.tsx        # Single memory with voting
│   ├── MemoryActions.tsx     # Pin, delete, edit actions
│   ├── MissingMemory.tsx     # "Add memory for next time" prompt
│   └── IrrelevanceReason.tsx # Why wasn't this relevant?
├── ReasoningFeedback/
│   ├── ReasoningVoting.tsx   # Vote on reasoning steps
│   ├── ReasoningStep.tsx     # Single step with voting
│   ├── ReasoningChain.tsx    # Full chain with summary
│   └── ReasoningIssues.tsx   # Issue type selector
├── NoteEditor/
│   └── InlineNoteEditor.tsx  # Add/edit notes inline
├── FeedbackPanel/
│   └── MessageFeedback.tsx   # Vote + note combined
└── ServerInfo/
    ├── ConnectionStatus.tsx  # Server connection details
    ├── SyncStatus.tsx        # Sync state indicator
    └── ModelInfo.tsx         # Current model/config

organisms/
├── MessageBubble/
│   └── EnhancedMessageBubble.tsx  # Message + all feedback
├── MemoryManager/
│   ├── MemoryList.tsx        # All memories for conversation
│   ├── MemorySearch.tsx      # Search/filter memories
│   └── MemoryEditor.tsx      # Create/edit memory
├── FeedbackDashboard/
│   └── ResponseFeedback.tsx  # Aggregate feedback view
├── ServerPanel/
│   └── ServerInfoPanel.tsx   # All server information
└── NotesPanel/
    └── UserNotesPanel.tsx    # All notes for conversation
```

---

## Feature Specifications

### 1. Granular Voting System

The voting system provides feedback at multiple levels of AI decision-making, enabling fine-grained improvement signals.

#### Votable Elements

| Element | Purpose | Feedback Value |
|---------|---------|----------------|
| **Message** | Overall response quality | General satisfaction |
| **Tool Use** | Was the right tool used correctly? | Tool selection & parameter tuning |
| **Memory Selection** | Was this memory relevant? | Memory retrieval improvement |
| **Reasoning Step** | Was this reasoning helpful? | Reasoning chain optimization |

---

#### 1.1 Message-Level Voting

##### UX Design
```
┌─────────────────────────────────────────────────────────┐
│  Assistant Message                                       │
│  ─────────────────────────────────────────────────────  │
│  Lorem ipsum dolor sit amet, consectetur adipiscing...   │
│                                                          │
│  ┌─────────────────────────────────────────────────────┐│
│  │ 👍 12  │ 👎 2  │ 📝 Add Note  │ ⋮ More              ││
│  └─────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────┘
```

---

#### 1.2 Tool Use Voting

##### UX Design
```
┌─────────────────────────────────────────────────────────┐
│ 🔧 Tools Used                                      [▼]   │
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 📁 read_file                                    ✓   │ │
│ │ ─────────────────────────────────────────────────── │ │
│ │ Path: src/components/Button.tsx                     │ │
│ │ Result: 45 lines read                               │ │
│ │                                                     │ │
│ │ Was this tool use helpful?                          │ │
│ │ [👍 Good choice] [👎 Wrong tool] [📝 Note]          │ │
│ │                                                     │ │
│ │ Quick feedback:                                     │ │
│ │ [Should have used different file]                   │ │
│ │ [Parameters were wrong]                             │ │
│ │ [Unnecessary tool call]                             │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                          │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 🔍 grep                                         ✓   │ │
│ │ ─────────────────────────────────────────────────── │ │
│ │ Pattern: "useState"                                 │ │
│ │ Result: 12 matches                                  │ │
│ │                                                     │ │
│ │ [👍 3] [👎 0] [📝 1 note]                           │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

##### Component: `ToolUseVoting.tsx`
```typescript
interface ToolUseFeedback {
  id: string;
  toolUseId: string;
  messageId: string;
  vote: 'up' | 'down' | null;
  quickFeedback?: 'wrong_tool' | 'wrong_params' | 'unnecessary' | 'missing_context';
  note?: string;
  timestamp: number;
}

interface ToolUseVotingProps {
  toolUseId: string;
  toolName: string;
  parameters: Record<string, unknown>;
  result: unknown;
  status: 'running' | 'success' | 'failed';
  feedback: ToolUseFeedback | null;
  onVote: (vote: 'up' | 'down') => void;
  onQuickFeedback: (type: ToolUseFeedback['quickFeedback']) => void;
  onAddNote: (note: string) => void;
}
```

##### Quick Feedback Options for Tools
- **Wrong tool**: "A different tool would have been better"
- **Wrong parameters**: "Right tool, wrong arguments"
- **Unnecessary**: "This tool call wasn't needed"
- **Missing context**: "Needed more context before calling"
- **Perfect**: "Exactly the right choice"

---

#### 1.3 Memory Selection Voting

##### UX Design
```
┌─────────────────────────────────────────────────────────┐
│ 🧠 Retrieved Memories                              [▼]   │
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────┐ │
│ │ "User prefers TypeScript over JavaScript"          │ │
│ │ ─────────────────────────────────────────────────── │ │
│ │ Relevance: 95%  •  Used 12 times  •  📌 Pinned     │ │
│ │                                                     │ │
│ │ Was this memory relevant to your question?          │ │
│ │ [👍 Relevant] [👎 Not relevant] [🎯 Critical]       │ │
│ │                                                     │ │
│ │ [Edit memory] [Unpin]                               │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                          │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ "User works on React projects"                      │ │
│ │ ─────────────────────────────────────────────────── │ │
│ │ Relevance: 67%  •  Used 5 times                     │ │
│ │                                                     │ │
│ │ [👍 2] [👎 1] [🎯 0]                                │ │
│ │                                                     │ │
│ │ Why wasn't this relevant?                           │ │
│ │ [Outdated info] [Wrong context] [Too generic]       │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                          │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ ➕ Missing memory?                                  │ │
│ │ "Was there context I should have remembered?"       │ │
│ │ [Add memory for next time...]                       │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

##### Component: `MemoryVoting.tsx`
```typescript
interface MemoryFeedback {
  id: string;
  memoryId: string;
  messageId: string;
  relevance: 'relevant' | 'not_relevant' | 'critical';
  irrelevanceReason?: 'outdated' | 'wrong_context' | 'too_generic' | 'incorrect';
  shouldUpdate?: string; // Suggested update to memory
  timestamp: number;
}

interface MemoryVotingProps {
  memoryId: string;
  content: string;
  relevanceScore: number;
  usageCount: number;
  pinned: boolean;
  feedback: MemoryFeedback | null;
  aggregateFeedback: {
    relevant: number;
    notRelevant: number;
    critical: number;
  };
  onVote: (relevance: MemoryFeedback['relevance']) => void;
  onIrrelevanceReason: (reason: MemoryFeedback['irrelevanceReason']) => void;
  onSuggestUpdate: (suggestion: string) => void;
  onEdit: () => void;
  onPin: (pinned: boolean) => void;
}
```

##### Memory Voting Options
- **Relevant** (👍): Memory was helpful for this response
- **Not relevant** (👎): Memory shouldn't have been retrieved
- **Critical** (🎯): This memory was essential - always use it

##### Irrelevance Reasons
- **Outdated**: Information is no longer accurate
- **Wrong context**: Doesn't apply to this situation
- **Too generic**: Not specific enough to be useful
- **Incorrect**: The memory contains wrong information

---

#### 1.4 Reasoning Step Voting

##### UX Design
```
┌─────────────────────────────────────────────────────────┐
│ 💭 Reasoning Process                               [▼]   │
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Step 1: Understanding the request                   │ │
│ │ ─────────────────────────────────────────────────── │ │
│ │ "The user wants to refactor the Button component    │ │
│ │ to use TypeScript generics for better type safety"  │ │
│ │                                                     │ │
│ │ [👍 Correct understanding] [👎 Misunderstood]       │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                          │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Step 2: Analyzing current implementation            │ │
│ │ ─────────────────────────────────────────────────── │ │
│ │ "Looking at the existing Button.tsx, I see it uses  │ │
│ │ React.FC with inline prop types..."                 │ │
│ │                                                     │ │
│ │ [👍 1] [👎 0]                                       │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                          │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Step 3: Planning the refactor                       │ │
│ │ ─────────────────────────────────────────────────── │ │
│ │ "I'll create a generic ButtonProps<T> interface..." │ │
│ │                                                     │ │
│ │ Was this reasoning step helpful?                    │ │
│ │ [👍 Good logic] [👎 Flawed reasoning] [📝 Note]     │ │
│ │                                                     │ │
│ │ What was wrong?                                     │ │
│ │ [Incorrect assumption] [Missed consideration]       │ │
│ │ [Overcomplicated] [Wrong direction]                 │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                          │
│ Overall Reasoning Quality                                │
│ ────────────────────────                                │
│ Steps: 3  •  👍 2  •  👎 0                               │
│ [Rate overall reasoning chain...]                        │
└─────────────────────────────────────────────────────────┘
```

##### Component: `ReasoningVoting.tsx`
```typescript
interface ReasoningFeedback {
  id: string;
  reasoningStepId: string;
  messageId: string;
  stepNumber: number;
  vote: 'up' | 'down' | null;
  issue?: 'incorrect_assumption' | 'missed_consideration' | 'overcomplicated' | 'wrong_direction';
  note?: string;
  timestamp: number;
}

interface ReasoningVotingProps {
  stepId: string;
  stepNumber: number;
  content: string;
  feedback: ReasoningFeedback | null;
  onVote: (vote: 'up' | 'down') => void;
  onIssue: (issue: ReasoningFeedback['issue']) => void;
  onAddNote: (note: string) => void;
}

interface ReasoningChainSummaryProps {
  messageId: string;
  steps: Array<{
    id: string;
    content: string;
    feedback: ReasoningFeedback | null;
  }>;
  overallRating: number | null; // 1-5 stars
  onRateOverall: (rating: number) => void;
}
```

##### Reasoning Issues
- **Incorrect assumption**: Started from a wrong premise
- **Missed consideration**: Didn't account for something important
- **Overcomplicated**: Made it harder than necessary
- **Wrong direction**: Went down an unproductive path

---

#### Unified Vote Control Component

##### Component: `VoteControl.tsx`
```typescript
type VotableType = 'message' | 'tool_use' | 'memory' | 'reasoning';

interface VoteControlProps {
  // Identification
  targetType: VotableType;
  targetId: string;
  messageId: string;

  // State
  upvotes: number;
  downvotes: number;
  userVote: 'up' | 'down' | null;

  // Type-specific props
  specialVote?: 'critical'; // For memories
  quickFeedbackOptions?: QuickFeedbackOption[];

  // Callbacks
  onVote: (vote: 'up' | 'down') => void;
  onSpecialVote?: (vote: string) => void;
  onQuickFeedback?: (feedback: string) => void;
  onAddNote?: () => void;

  // Display
  size?: 'sm' | 'md' | 'lg';
  showCounts?: boolean;
  showNote?: boolean;
  disabled?: boolean;
}

interface QuickFeedbackOption {
  id: string;
  label: string;
  icon?: string;
}
```

---

#### Protocol Extensions for Granular Voting

```typescript
// Unified feedback envelope
Feedback (20): {
  id: string;
  conversationId: string;
  messageId: string;
  targetType: 'message' | 'tool_use' | 'memory' | 'reasoning';
  targetId: string;
  vote: 'up' | 'down' | 'critical' | 'remove';
  quickFeedback?: string;
  note?: string;
  timestamp: number;
}

// Server response
FeedbackConfirmation (21): {
  feedbackId: string;
  targetType: string;
  targetId: string;
  aggregates: {
    upvotes: number;
    downvotes: number;
    specialVotes?: Record<string, number>;
  };
  userVote: 'up' | 'down' | 'critical' | null;
}

// Batch feedback for multiple items
BatchFeedback (28): {
  conversationId: string;
  messageId: string;
  items: Array<{
    targetType: string;
    targetId: string;
    vote: string;
    quickFeedback?: string;
  }>;
  timestamp: number;
}
```

---

### 2. User Notes System

#### UX Design
```
┌─────────────────────────────────────────────────────────┐
│  Assistant Message                                       │
│  ─────────────────────────────────────────────────────  │
│  Lorem ipsum dolor sit amet...                           │
│                                                          │
│  📝 Notes (2)                                      [+]   │
│  ┌─────────────────────────────────────────────────────┐│
│  │ "This response could include more context about..." ││
│  │ — You, 2 hours ago                        [Edit][×] ││
│  ├─────────────────────────────────────────────────────┤│
│  │ "Consider mentioning the alternative approach"      ││
│  │ — You, 1 day ago                          [Edit][×] ││
│  └─────────────────────────────────────────────────────┘│
│                                                          │
│  ┌─────────────────────────────────────────────────────┐│
│  │ Add a note for improvement...                    [→]││
│  └─────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────┘
```

#### Component: `InlineNoteEditor.tsx`
```typescript
interface UserNote {
  id: string;
  messageId: string;
  content: string;
  category: 'improvement' | 'correction' | 'context' | 'general';
  createdAt: number;
  updatedAt: number;
}

interface NoteEditorProps {
  messageId: string;
  notes: UserNote[];
  onAddNote: (note: Omit<UserNote, 'id' | 'createdAt' | 'updatedAt'>) => void;
  onEditNote: (id: string, content: string) => void;
  onDeleteNote: (id: string) => void;
  collapsed?: boolean;
}
```

#### Note Categories
- **Improvement**: "Response could be better if..."
- **Correction**: "This is factually incorrect because..."
- **Context**: "User meant X, not Y..."
- **General**: Freeform notes

#### Quick Note Templates
```
[Template chips for fast input]
[ ] Could use more detail
[ ] Too verbose
[ ] Incorrect information
[ ] Missing context
[ ] Great response!
```

---

### 3. Memory Management System

#### UX Design - Memory Panel (Sidebar)
```
┌─────────────────────────────────────┐
│ 🧠 Memories                    [+]  │
├─────────────────────────────────────┤
│ 🔍 Search memories...               │
├─────────────────────────────────────┤
│ Filter: [All ▾] [Relevance ▾]       │
├─────────────────────────────────────┤
│ ┌─────────────────────────────────┐ │
│ │ 📌 User prefers TypeScript      │ │
│ │ ────────────────────────────    │ │
│ │ Relevance: 95%                  │ │
│ │ Used: 12 times                  │ │
│ │ Created: 3 days ago             │ │
│ │ [Edit] [Archive] [Delete]       │ │
│ └─────────────────────────────────┘ │
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ User works on React projects    │ │
│ │ ────────────────────────────    │ │
│ │ Relevance: 87%                  │ │
│ │ Used: 8 times                   │ │
│ │ [Edit] [Archive] [Delete]       │ │
│ └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

#### UX Design - In-Message Memory Traces
```
┌─────────────────────────────────────────────────────────┐
│  Assistant Message                                       │
│  ─────────────────────────────────────────────────────  │
│  Based on your preference for TypeScript, here's...      │
│                                                          │
│  🧠 Used Memories                                  [▼]   │
│  ┌─────────────────────────────────────────────────────┐│
│  │ "User prefers TypeScript"              95% match    ││
│  │ [👍 Helpful] [👎 Not relevant] [📌 Pin]            ││
│  └─────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────┘
```

#### Component: `MemoryManager.tsx`
```typescript
interface Memory {
  id: string;
  content: string;
  category: 'preference' | 'fact' | 'context' | 'instruction';
  relevanceScore?: number;
  usageCount: number;
  pinned: boolean;
  archived: boolean;
  createdAt: number;
  updatedAt: number;
  lastUsedAt?: number;
}

interface MemoryManagerProps {
  conversationId: string;
  memories: Memory[];
  onCreateMemory: (memory: Omit<Memory, 'id' | 'createdAt' | 'updatedAt' | 'usageCount'>) => void;
  onUpdateMemory: (id: string, updates: Partial<Memory>) => void;
  onDeleteMemory: (id: string) => void;
  onPinMemory: (id: string, pinned: boolean) => void;
  onArchiveMemory: (id: string) => void;
}
```

#### Memory Actions
- **Create**: Add new memory manually
- **Edit**: Modify existing memory content
- **Pin**: Keep memory always active
- **Archive**: Hide but don't delete
- **Delete**: Remove permanently
- **Rate**: Was this memory helpful in context?

#### Memory Categories
- **Preference**: User preferences and styles
- **Fact**: Information about user/project
- **Context**: Situational context
- **Instruction**: How to respond/behave

---

### 4. Server Information Panel

#### UX Design
```
┌─────────────────────────────────────────────────────────┐
│ 🖥️ Server Information                            [⟳]    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ Connection                                               │
│ ├─ Status: 🟢 Connected                                 │
│ ├─ Latency: 42ms                                        │
│ ├─ Protocol: LiveKit WebSocket                          │
│ └─ Last heartbeat: 2s ago                               │
│                                                          │
│ Sync Status                                              │
│ ├─ State: ✓ Synced                                      │
│ ├─ Last sync: 5 seconds ago                             │
│ ├─ Pending: 0 messages                                  │
│ └─ Conflicts: 0                                         │
│                                                          │
│ Model Configuration                                      │
│ ├─ Model: claude-3-opus                                 │
│ ├─ Context window: 200K                                 │
│ ├─ Temperature: 0.7                                     │
│ └─ Max tokens: 4096                                     │
│                                                          │
│ Voice Settings                                           │
│ ├─ TTS Provider: ElevenLabs                             │
│ ├─ Voice: Aria                                          │
│ ├─ Speed: 1.0x                                          │
│ └─ Auto-play: On                                        │
│                                                          │
│ Active MCP Servers (2)                                   │
│ ├─ filesystem (stdio) 🟢                                │
│ └─ github (SSE) 🟢                                      │
│                                                          │
│ Session Stats                                            │
│ ├─ Messages: 24                                         │
│ ├─ Tool calls: 8                                        │
│ ├─ Memories used: 3                                     │
│ └─ Session duration: 45 min                             │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

#### Component: `ServerInfoPanel.tsx`
```typescript
interface ServerInfo {
  connection: {
    status: 'connected' | 'connecting' | 'disconnected' | 'reconnecting';
    latency: number;
    protocol: string;
    lastHeartbeat: number;
  };
  sync: {
    state: 'synced' | 'syncing' | 'error';
    lastSync: number;
    pendingCount: number;
    conflictCount: number;
  };
  model: {
    name: string;
    contextWindow: number;
    temperature: number;
    maxTokens: number;
  };
  voice: {
    provider: string;
    voice: string;
    speed: number;
    autoPlay: boolean;
  };
  mcpServers: Array<{
    name: string;
    transport: string;
    status: 'connected' | 'disconnected' | 'error';
  }>;
  stats: {
    messageCount: number;
    toolCallCount: number;
    memoriesUsed: number;
    sessionDuration: number;
  };
}
```

---

### 5. Enhanced Message Bubble

#### Complete UX Design with Granular Voting
```
┌─────────────────────────────────────────────────────────┐
│ 🤖 Assistant                                   12:34 PM │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ Here's the TypeScript implementation you requested...    │
│                                                          │
│ ```typescript                                            │
│ function example(): void {                               │
│   console.log("Hello");                                  │
│ }                                                        │
│ ```                                                      │
│                                                          │
├─────────────────────────────────────────────────────────┤
│ 🔧 Tools Used (2)                                  [▼]   │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 📁 read_file: src/index.ts                     ✓   │ │
│ │ [👍 Good] [👎 Wrong] [Unnecessary] [Wrong params]   │ │
│ ├─────────────────────────────────────────────────────┤ │
│ │ 🔍 grep: "useState" in src/                    ✓   │ │
│ │ [👍 2] [👎 0]                            [📝 Note]  │ │
│ └─────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────┤
│ 🧠 Memories (2)                                    [▼]   │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ "User prefers TypeScript" (95%) 📌                  │ │
│ │ [👍 Relevant] [👎 Not relevant] [🎯 Critical]       │ │
│ ├─────────────────────────────────────────────────────┤ │
│ │ "Working on Node.js project" (87%)                  │ │
│ │ [👍 1] [👎 0] [🎯 0]                     [Edit]     │ │
│ ├─────────────────────────────────────────────────────┤ │
│ │ ➕ Missing memory? [Add for next time...]           │ │
│ └─────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────┤
│ 💭 Reasoning (3 steps)                             [▼]   │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 1. Understanding request          [👍] [👎]        │ │
│ │ 2. Analyzing implementation       [👍 1] [👎 0]    │ │
│ │ 3. Planning the solution          [👍] [👎]        │ │
│ ├─────────────────────────────────────────────────────┤ │
│ │ Overall reasoning: ⭐⭐⭐⭐☆ (4/5)                  │ │
│ └─────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ 📝 Notes (1)                                       [+]   │
│ └─ "Consider using async/await pattern" - 2h ago        │
│                                                          │
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 👍 12  │ 👎 2  │ [Regenerate] │ [Copy] │ [⋮]       │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

#### Collapsed State (Default)
```
┌─────────────────────────────────────────────────────────┐
│ 🤖 Assistant                                   12:34 PM │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ Here's the TypeScript implementation you requested...    │
│                                                          │
│ ```typescript                                            │
│ function example(): void { ... }                         │
│ ```                                                      │
│                                                          │
├─────────────────────────────────────────────────────────┤
│ 🔧 2 tools  │  🧠 2 memories  │  💭 3 steps        [▼]   │
├─────────────────────────────────────────────────────────┤
│ 👍 12  │ 👎 2  │ 📝 1  │ [Regenerate] │ [⋮]              │
└─────────────────────────────────────────────────────────┘
```

#### Feedback Summary Indicators
The collapsed state shows aggregate feedback with visual indicators:
- Green dot: Mostly positive feedback
- Yellow dot: Mixed feedback
- Red dot: Mostly negative feedback
- Number badges show counts

```
🔧 2 tools 🟢  │  🧠 2 memories 🟡  │  💭 3 steps 🟢
```

---

## Implementation Phases

### Phase 1: Foundation (Atoms & Core Infrastructure)

1. **Create Atomic Components**
   - [ ] IconButton, PrimaryButton, GhostButton
   - [ ] StatusBadge, ScoreBadge, CountBadge
   - [ ] TextInput, TextArea
   - [ ] BaseCard, CollapsibleCard
   - [ ] Icon set (ThumbsUp, ThumbsDown, Note, Memory, etc.)
   - [ ] Tooltip, Toast components

2. **Extend Protocol Types**
   - [ ] Add MessageVote envelope type
   - [ ] Add UserNote envelope type
   - [ ] Add MemoryAction envelope type
   - [ ] Add ServerInfo envelope type

3. **Extend Local Database**
   - [ ] Add `votes` table
   - [ ] Add `notes` table
   - [ ] Add `memories` table with full CRUD
   - [ ] Add `session_stats` table

4. **Create Zustand Stores**
   - [ ] `feedbackStore`: Votes, notes state
   - [ ] `memoryStore`: Global memory management state
   - [ ] `serverInfoStore`: Server information state

### Phase 2: Granular Voting System

1. **Core Voting Components**
   - [ ] VoteControl (unified component)
   - [ ] VoteButton with animations
   - [ ] VoteCount with animated transitions
   - [ ] QuickFeedback chip selector

2. **Tool Use Voting**
   - [ ] ToolUseVoting molecule
   - [ ] ToolUseCard with integrated voting
   - [ ] ToolFeedbackChips (wrong tool, wrong params, etc.)

3. **Memory Voting**
   - [ ] MemoryVoting molecule
   - [ ] MemoryCard with relevance voting
   - [ ] IrrelevanceReason selector
   - [ ] MissingMemory prompt ("Add for next time")

4. **Reasoning Voting**
   - [ ] ReasoningVoting molecule
   - [ ] ReasoningStep with voting
   - [ ] ReasoningChain with summary
   - [ ] ReasoningIssues selector
   - [ ] Overall reasoning rating (5-star)

5. **Hooks**
   - [ ] `useFeedback()`: Unified feedback state and actions
   - [ ] `useFeedbackAggregates()`: Compute summary indicators

6. **Integration**
   - [ ] Integrate voting into ToolUsageDisplay
   - [ ] Integrate voting into ProtocolDisplay (memories)
   - [ ] Integrate voting into ReasoningSteps
   - [ ] Wire up protocol handlers
   - [ ] Add to sync system

### Phase 3: Notes System

1. **Components**
   - [ ] InlineNoteEditor molecule
   - [ ] NoteCard atom
   - [ ] NoteTemplates molecule
   - [ ] NotesPanel organism

2. **Hooks**
   - [ ] `useNotes()`: Notes CRUD
   - [ ] `useNoteSync()`: Sync with server

3. **Integration**
   - [ ] Add notes section to MessageBubble
   - [ ] Create NotesPanel in sidebar
   - [ ] Wire up protocol handlers

### Phase 4: Memory Management

1. **Components**
   - [ ] MemoryCard molecule
   - [ ] MemoryEditor molecule
   - [ ] MemoryList organism
   - [ ] MemorySearch molecule
   - [ ] MemoryManager organism
   - [ ] InlineMemoryTrace (for messages)

2. **Hooks**
   - [ ] `useMemories()`: Memory CRUD (global scope)
   - [ ] `useMemorySearch()`: Search/filter
   - [ ] `useMemoryFeedback()`: Rate memory relevance

3. **Integration**
   - [ ] Add MemoryManager to sidebar
   - [ ] Enhance ProtocolDisplay with memory actions
   - [ ] Create memory creation flow

### Phase 5: Server Information Panel

1. **Components**
   - [ ] ConnectionStatus molecule
   - [ ] SyncStatus molecule
   - [ ] ModelInfo molecule
   - [ ] MCPServerStatus molecule
   - [ ] SessionStats molecule
   - [ ] ServerInfoPanel organism

2. **Hooks**
   - [ ] `useServerInfo()`: Aggregate server state
   - [ ] `useLatencyMonitor()`: Connection quality

3. **Integration**
   - [ ] Add panel to settings or sidebar
   - [ ] Real-time updates from protocol

### Phase 6: Enhanced Message Bubble

1. **Refactor MessageBubble**
   - [ ] Integrate VoteControl
   - [ ] Integrate InlineNoteEditor
   - [ ] Integrate InlineMemoryTrace
   - [ ] Add collapsible sections
   - [ ] Add timestamp and metadata

2. **Micro-interactions**
   - [ ] Hover states
   - [ ] Expand/collapse animations
   - [ ] Vote animations
   - [ ] Copy feedback

### Phase 7: Polish & Accessibility

1. **Accessibility**
   - [ ] ARIA labels for all interactive elements
   - [ ] Keyboard navigation
   - [ ] Screen reader support
   - [ ] Focus management

2. **Responsive Design**
   - [ ] Mobile-friendly memory panel
   - [ ] Touch-friendly vote buttons
   - [ ] Adaptive layouts

3. **Performance**
   - [ ] Virtualized lists for notes/memories
   - [ ] Lazy loading for collapsed sections

---

## API Endpoints (Backend)

```
# Message Voting
POST   /api/v1/messages/{id}/vote          # Vote on message
DELETE /api/v1/messages/{id}/vote          # Remove vote
GET    /api/v1/messages/{id}/votes         # Get votes for message

# Tool Use Voting
POST   /api/v1/tool-uses/{id}/vote         # Vote on tool use
DELETE /api/v1/tool-uses/{id}/vote         # Remove vote
POST   /api/v1/tool-uses/{id}/quick-feedback    # Wrong tool, wrong params, etc.

# Memory Voting
POST   /api/v1/memories/{id}/vote          # Vote on memory relevance
DELETE /api/v1/memories/{id}/vote          # Remove vote
POST   /api/v1/memories/{id}/irrelevance-reason # Why wasn't this relevant

# Reasoning Voting
POST   /api/v1/reasoning/{id}/vote         # Vote on reasoning step
DELETE /api/v1/reasoning/{id}/vote         # Remove vote
POST   /api/v1/reasoning/{id}/issue        # Reasoning issue type

# Notes
POST   /api/v1/messages/{id}/notes         # Add note to message
GET    /api/v1/messages/{id}/notes         # Get notes for message
PUT    /api/v1/notes/{id}                  # Update note
DELETE /api/v1/notes/{id}                  # Delete note
POST   /api/v1/tool-uses/{id}/notes        # Add note to tool use
POST   /api/v1/reasoning/{id}/notes        # Add note to reasoning step

# Memories (Global Scope)
POST   /api/v1/memories                    # Create memory
GET    /api/v1/memories                    # List all memories
GET    /api/v1/memories/{id}               # Get single memory
PUT    /api/v1/memories/{id}               # Update memory
DELETE /api/v1/memories/{id}               # Delete memory
POST   /api/v1/memories/{id}/pin           # Pin/unpin memory
POST   /api/v1/memories/{id}/archive       # Archive memory

# Server Info
GET    /api/v1/server/info                 # Server configuration
GET    /api/v1/session/stats               # Session statistics
```

---

## Protocol Extensions (LiveKit)

```typescript
// New envelope types
enum EnvelopeType {
  // Existing...

  // New types
  MessageVote = 20,
  VoteConfirmation = 21,
  UserNote = 22,
  NoteConfirmation = 23,
  MemoryAction = 24,
  MemoryConfirmation = 25,
  ServerInfo = 26,
  SessionStats = 27,
}
```

---

## Design Tokens

```css
:root {
  /* Colors */
  --color-upvote: #22c55e;
  --color-upvote-hover: #16a34a;
  --color-downvote: #ef4444;
  --color-downvote-hover: #dc2626;
  --color-note: #3b82f6;
  --color-memory: #8b5cf6;
  --color-memory-high: #22c55e;
  --color-memory-medium: #eab308;
  --color-memory-low: #6b7280;

  /* Spacing */
  --space-feedback-gap: 8px;
  --space-section-padding: 12px;

  /* Animation */
  --transition-vote: 150ms ease-out;
  --transition-expand: 200ms ease-in-out;
}
```

---

## State Management (Zustand)

Use Zustand for lightweight, performant state management instead of React Context.

### Feedback Store

```typescript
// frontend/src/stores/feedbackStore.ts
import { create } from 'zustand';

interface Vote {
  id: string;
  targetType: 'message' | 'tool_use' | 'memory' | 'reasoning';
  targetId: string;
  vote: 'up' | 'down' | 'critical';
  quickFeedback?: string;
  timestamp: number;
}

interface FeedbackStore {
  votes: Map<string, Vote>;

  // Actions
  addVote: (targetType: Vote['targetType'], targetId: string, vote: Vote['vote']) => void;
  removeVote: (targetType: Vote['targetType'], targetId: string) => void;
  getVote: (targetType: Vote['targetType'], targetId: string) => Vote | undefined;

  // Aggregates
  getVoteCounts: (targetType: Vote['targetType'], targetId: string) => { up: number; down: number };
}

export const useFeedbackStore = create<FeedbackStore>((set, get) => ({
  votes: new Map(),

  addVote: (targetType, targetId, vote) => {
    const key = `${targetType}:${targetId}`;
    set((state) => {
      const newVotes = new Map(state.votes);
      newVotes.set(key, {
        id: crypto.randomUUID(),
        targetType,
        targetId,
        vote,
        timestamp: Date.now(),
      });
      return { votes: newVotes };
    });
  },

  removeVote: (targetType, targetId) => {
    const key = `${targetType}:${targetId}`;
    set((state) => {
      const newVotes = new Map(state.votes);
      newVotes.delete(key);
      return { votes: newVotes };
    });
  },

  getVote: (targetType, targetId) => {
    const key = `${targetType}:${targetId}`;
    return get().votes.get(key);
  },

  getVoteCounts: (targetType, targetId) => {
    // For single-user, just return current vote state
    const vote = get().getVote(targetType, targetId);
    return {
      up: vote?.vote === 'up' ? 1 : 0,
      down: vote?.vote === 'down' ? 1 : 0,
    };
  },
}));
```

### Memory Store (Global Scope)

```typescript
// frontend/src/stores/memoryStore.ts
import { create } from 'zustand';

interface Memory {
  id: string;
  content: string;
  category: 'preference' | 'fact' | 'context' | 'instruction';
  pinned: boolean;
  archived: boolean;
  createdAt: number;
  updatedAt: number;
}

interface MemoryStore {
  memories: Memory[];

  // Actions
  createMemory: (memory: Omit<Memory, 'id' | 'createdAt' | 'updatedAt'>) => void;
  updateMemory: (id: string, updates: Partial<Memory>) => void;
  deleteMemory: (id: string) => void;
  pinMemory: (id: string, pinned: boolean) => void;
  archiveMemory: (id: string) => void;

  // Queries
  searchMemories: (query: string) => Memory[];
  getPinnedMemories: () => Memory[];
}

export const useMemoryStore = create<MemoryStore>((set, get) => ({
  memories: [],

  createMemory: (memory) => {
    const now = Date.now();
    set((state) => ({
      memories: [...state.memories, {
        ...memory,
        id: crypto.randomUUID(),
        createdAt: now,
        updatedAt: now,
      }],
    }));
  },

  updateMemory: (id, updates) => {
    set((state) => ({
      memories: state.memories.map((m) =>
        m.id === id ? { ...m, ...updates, updatedAt: Date.now() } : m
      ),
    }));
  },

  deleteMemory: (id) => {
    set((state) => ({
      memories: state.memories.filter((m) => m.id !== id),
    }));
  },

  pinMemory: (id, pinned) => {
    get().updateMemory(id, { pinned });
  },

  archiveMemory: (id) => {
    get().updateMemory(id, { archived: true });
  },

  searchMemories: (query) => {
    const q = query.toLowerCase();
    return get().memories.filter((m) =>
      !m.archived && m.content.toLowerCase().includes(q)
    );
  },

  getPinnedMemories: () => {
    return get().memories.filter((m) => m.pinned && !m.archived);
  },
}));
```

### Notes Store

```typescript
// frontend/src/stores/notesStore.ts
import { create } from 'zustand';

interface UserNote {
  id: string;
  messageId: string;
  content: string;
  category: 'improvement' | 'correction' | 'context' | 'general';
  createdAt: number;
  updatedAt: number;
}

interface NotesStore {
  notes: Map<string, UserNote[]>; // messageId -> notes

  addNote: (messageId: string, content: string, category: UserNote['category']) => void;
  updateNote: (noteId: string, content: string) => void;
  deleteNote: (noteId: string) => void;
  getNotesForMessage: (messageId: string) => UserNote[];
}

export const useNotesStore = create<NotesStore>((set, get) => ({
  notes: new Map(),

  addNote: (messageId, content, category) => {
    const now = Date.now();
    const note: UserNote = {
      id: crypto.randomUUID(),
      messageId,
      content,
      category,
      createdAt: now,
      updatedAt: now,
    };

    set((state) => {
      const newNotes = new Map(state.notes);
      const existing = newNotes.get(messageId) || [];
      newNotes.set(messageId, [...existing, note]);
      return { notes: newNotes };
    });
  },

  updateNote: (noteId, content) => {
    set((state) => {
      const newNotes = new Map(state.notes);
      for (const [messageId, notes] of newNotes) {
        const updated = notes.map((n) =>
          n.id === noteId ? { ...n, content, updatedAt: Date.now() } : n
        );
        newNotes.set(messageId, updated);
      }
      return { notes: newNotes };
    });
  },

  deleteNote: (noteId) => {
    set((state) => {
      const newNotes = new Map(state.notes);
      for (const [messageId, notes] of newNotes) {
        newNotes.set(messageId, notes.filter((n) => n.id !== noteId));
      }
      return { notes: newNotes };
    });
  },

  getNotesForMessage: (messageId) => {
    return get().notes.get(messageId) || [];
  },
}));
```

---

## Feedback Visibility (Modifier Key)

Votes and notes are hidden by default to keep the UI clean. Hold **Alt** (or **Cmd** on Mac) to reveal feedback controls.

### Implementation

```typescript
// frontend/src/hooks/useFeedbackVisibility.ts
import { useState, useEffect } from 'react';

export function useFeedbackVisibility() {
  const [showFeedback, setShowFeedback] = useState(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.altKey || e.metaKey) {
        setShowFeedback(true);
      }
    };

    const handleKeyUp = (e: KeyboardEvent) => {
      if (!e.altKey && !e.metaKey) {
        setShowFeedback(false);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
    };
  }, []);

  return showFeedback;
}
```

### Usage in Components

```tsx
// In MessageBubble.tsx
import { useFeedbackVisibility } from '../hooks/useFeedbackVisibility';

function MessageBubble({ message }: Props) {
  const showFeedback = useFeedbackVisibility();

  return (
    <div className="message-bubble">
      <div className="message-content">{message.content}</div>

      {showFeedback && (
        <div className="feedback-controls">
          <VoteControl targetType="message" targetId={message.id} />
          <NoteButton messageId={message.id} />
        </div>
      )}
    </div>
  );
}
```

### Visual Indicator

```tsx
// Show a subtle hint that feedback mode is active
{showFeedback && (
  <div className="feedback-mode-indicator">
    Press Alt/Cmd to show feedback controls
  </div>
)}
```

---

## Protocol Files

### Frontend Protocol Types

```typescript
// frontend/src/types/feedback-protocol.ts

export enum FeedbackEnvelopeType {
  Feedback = 20,
  FeedbackConfirmation = 21,
  UserNote = 22,
  NoteConfirmation = 23,
  MemoryAction = 24,
  MemoryConfirmation = 25,
  ServerInfo = 26,
  SessionStats = 27,
}

// Vote message sent from client to server
export interface FeedbackMessage {
  id: string;
  conversationId: string;
  messageId: string;
  targetType: 'message' | 'tool_use' | 'memory' | 'reasoning';
  targetId: string;
  vote: 'up' | 'down' | 'critical' | 'remove';
  quickFeedback?: string;
  note?: string;
  timestamp: number;
}

// Server confirmation with aggregate counts
export interface FeedbackConfirmation {
  feedbackId: string;
  targetType: string;
  targetId: string;
  aggregates: {
    upvotes: number;
    downvotes: number;
    specialVotes?: Record<string, number>;
  };
  userVote: 'up' | 'down' | 'critical' | null;
}

// Note message
export interface UserNoteMessage {
  id: string;
  messageId: string;
  content: string;
  category: 'improvement' | 'correction' | 'context' | 'general';
  action: 'create' | 'update' | 'delete';
  timestamp: number;
}

// Note confirmation
export interface NoteConfirmation {
  noteId: string;
  messageId: string;
  success: boolean;
}

// Memory action (global scope)
export interface MemoryActionMessage {
  id: string;
  action: 'create' | 'update' | 'delete' | 'pin' | 'archive';
  memory?: {
    content: string;
    category: 'preference' | 'fact' | 'context' | 'instruction';
    pinned?: boolean;
  };
  timestamp: number;
}

// Memory confirmation
export interface MemoryConfirmation {
  memoryId: string;
  action: string;
  success: boolean;
}

// Server info broadcast
export interface ServerInfoMessage {
  connection: {
    status: 'connected' | 'connecting' | 'disconnected';
    latency: number;
  };
  model: {
    name: string;
    provider: string;
  };
  mcpServers: Array<{
    name: string;
    status: 'connected' | 'disconnected' | 'error';
  }>;
}

// Session statistics
export interface SessionStatsMessage {
  messageCount: number;
  toolCallCount: number;
  memoriesUsed: number;
  sessionDuration: number;
}
```

---

## Backend Planning

### Go Handler Specifications

```go
// internal/adapters/http/handlers/votes.go
package handlers

import (
    "net/http"
    "github.com/go-chi/chi/v5"
)

type VoteHandler struct {
    voteService ports.VoteService
}

// POST /api/v1/messages/{id}/vote
func (h *VoteHandler) VoteOnMessage(w http.ResponseWriter, r *http.Request) {
    messageID := chi.URLParam(r, "id")
    var req struct {
        Vote          string  `json:"vote"` // "up", "down", "critical"
        QuickFeedback *string `json:"quickFeedback,omitempty"`
    }
    // Decode, validate, save
}

// POST /api/v1/tool-uses/{id}/vote
func (h *VoteHandler) VoteOnToolUse(w http.ResponseWriter, r *http.Request) {
    toolUseID := chi.URLParam(r, "id")
    // Similar implementation
}

// POST /api/v1/memories/{id}/vote
func (h *VoteHandler) VoteOnMemory(w http.ResponseWriter, r *http.Request) {
    memoryID := chi.URLParam(r, "id")
    // Similar implementation
}

// POST /api/v1/reasoning/{id}/vote
func (h *VoteHandler) VoteOnReasoning(w http.ResponseWriter, r *http.Request) {
    reasoningID := chi.URLParam(r, "id")
    // Similar implementation
}
```

```go
// internal/adapters/http/handlers/notes.go
package handlers

type NoteHandler struct {
    noteService ports.NoteService
}

// POST /api/v1/messages/{id}/notes
func (h *NoteHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
    messageID := chi.URLParam(r, "id")
    var req struct {
        Content  string `json:"content"`
        Category string `json:"category"`
    }
    // Create note
}

// PUT /api/v1/notes/{id}
func (h *NoteHandler) UpdateNote(w http.ResponseWriter, r *http.Request) {
    noteID := chi.URLParam(r, "id")
    // Update note
}

// DELETE /api/v1/notes/{id}
func (h *NoteHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
    noteID := chi.URLParam(r, "id")
    // Delete note
}
```

```go
// internal/adapters/http/handlers/memories.go
package handlers

type MemoryHandler struct {
    memoryService ports.MemoryService
}

// POST /api/v1/memories (global scope)
func (h *MemoryHandler) CreateMemory(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Content  string `json:"content"`
        Category string `json:"category"`
    }
    // Create memory
}

// GET /api/v1/memories
func (h *MemoryHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
    // Return all memories (with pagination)
}

// PUT /api/v1/memories/{id}
func (h *MemoryHandler) UpdateMemory(w http.ResponseWriter, r *http.Request) {
    memoryID := chi.URLParam(r, "id")
    // Update memory
}

// DELETE /api/v1/memories/{id}
func (h *MemoryHandler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
    memoryID := chi.URLParam(r, "id")
    // Delete memory
}

// POST /api/v1/memories/{id}/pin
func (h *MemoryHandler) PinMemory(w http.ResponseWriter, r *http.Request) {
    memoryID := chi.URLParam(r, "id")
    // Toggle pin status
}
```

### LiveKit Protocol Handlers

```go
// internal/adapters/livekit/feedback_handlers.go
package livekit

import (
    "context"
)

// HandleFeedback processes vote messages from clients
func (h *ProtocolHandler) HandleFeedback(ctx context.Context, msg *FeedbackMessage) error {
    // 1. Validate vote
    // 2. Store in database
    // 3. Send confirmation back to client
    confirmation := &FeedbackConfirmation{
        FeedbackID: msg.ID,
        TargetType: msg.TargetType,
        TargetID:   msg.TargetID,
        UserVote:   msg.Vote,
    }
    return h.sendMessage(ctx, FeedbackConfirmation, confirmation)
}

// HandleUserNote processes note messages from clients
func (h *ProtocolHandler) HandleUserNote(ctx context.Context, msg *UserNoteMessage) error {
    switch msg.Action {
    case "create":
        return h.noteService.Create(ctx, msg)
    case "update":
        return h.noteService.Update(ctx, msg.ID, msg.Content)
    case "delete":
        return h.noteService.Delete(ctx, msg.ID)
    }
    return nil
}

// HandleMemoryAction processes memory CRUD from clients
func (h *ProtocolHandler) HandleMemoryAction(ctx context.Context, msg *MemoryActionMessage) error {
    switch msg.Action {
    case "create":
        return h.memoryService.Create(ctx, msg.Memory)
    case "update":
        return h.memoryService.Update(ctx, msg.ID, msg.Memory)
    case "delete":
        return h.memoryService.Delete(ctx, msg.ID)
    case "pin":
        return h.memoryService.Pin(ctx, msg.ID)
    case "archive":
        return h.memoryService.Archive(ctx, msg.ID)
    }
    return nil
}
```

### Router Setup

```go
// internal/adapters/http/router.go
func SetupRoutes(r chi.Router, handlers *Handlers) {
    r.Route("/api/v1", func(r chi.Router) {
        // Message voting
        r.Post("/messages/{id}/vote", handlers.Vote.VoteOnMessage)
        r.Delete("/messages/{id}/vote", handlers.Vote.RemoveMessageVote)
        r.Get("/messages/{id}/votes", handlers.Vote.GetMessageVotes)

        // Tool use voting
        r.Post("/tool-uses/{id}/vote", handlers.Vote.VoteOnToolUse)
        r.Delete("/tool-uses/{id}/vote", handlers.Vote.RemoveToolUseVote)
        r.Post("/tool-uses/{id}/quick-feedback", handlers.Vote.ToolUseQuickFeedback)

        // Memory voting
        r.Post("/memories/{id}/vote", handlers.Vote.VoteOnMemory)
        r.Delete("/memories/{id}/vote", handlers.Vote.RemoveMemoryVote)

        // Reasoning voting
        r.Post("/reasoning/{id}/vote", handlers.Vote.VoteOnReasoning)
        r.Delete("/reasoning/{id}/vote", handlers.Vote.RemoveReasoningVote)

        // Notes
        r.Post("/messages/{id}/notes", handlers.Note.CreateNote)
        r.Get("/messages/{id}/notes", handlers.Note.GetNotesForMessage)
        r.Put("/notes/{id}", handlers.Note.UpdateNote)
        r.Delete("/notes/{id}", handlers.Note.DeleteNote)

        // Memories (global scope)
        r.Post("/memories", handlers.Memory.CreateMemory)
        r.Get("/memories", handlers.Memory.ListMemories)
        r.Get("/memories/{id}", handlers.Memory.GetMemory)
        r.Put("/memories/{id}", handlers.Memory.UpdateMemory)
        r.Delete("/memories/{id}", handlers.Memory.DeleteMemory)
        r.Post("/memories/{id}/pin", handlers.Memory.PinMemory)
        r.Post("/memories/{id}/archive", handlers.Memory.ArchiveMemory)

        // Server info
        r.Get("/server/info", handlers.Server.GetInfo)
        r.Get("/session/stats", handlers.Server.GetSessionStats)
    })
}
```

---

## Integration with DSPy/GEPA Optimization

The feedback collected through this UX system serves as the foundation for automatic prompt improvement via DSPy and GEPA optimization. See the [DSPy + GEPA Implementation Plan](dspy-gepa-implementation-plan.md) for full details.

### Feedback Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                      User Feedback Loop                          │
├─────────────────────────────────────────────────────────────────┤
│  User Actions           →    Stored Data     →    Optimization  │
│  ─────────────────────────────────────────────────────────────  │
│  Message upvote/downvote →  alicia_votes    →  Response quality │
│  Tool use voting        →  tool_feedback    →  Tool selection   │
│  Memory relevance vote  →  memory_feedback  →  Memory retrieval │
│  Reasoning step vote    →  reasoning_votes  →  Chain-of-thought │
│  User notes             →  alicia_notes     →  Instruction tuning│
└─────────────────────────────────────────────────────────────────┘
```

### How Feedback Improves the System

1. **Message Votes**: Positive votes on responses become training examples for GEPA's prompt optimization. Negative votes with notes trigger reflective analysis.

2. **Tool Use Feedback**: Votes on tool calls improve tool descriptions and parameter generation. "Wrong tool" feedback helps the model learn when to use which tool.

3. **Memory Relevance**: Voting on whether retrieved memories were helpful directly improves the memory retrieval system. Critical votes ensure important memories are always included.

4. **Reasoning Votes**: Feedback on reasoning steps helps optimize chain-of-thought prompts and identify common reasoning failures.

5. **User Notes**: Free-form notes provide rich feedback for GEPA's reflective mutation, enabling the system to understand why responses failed.

---

## Accessibility Considerations

All feedback components must be fully accessible:

### Keyboard Navigation
- All voting buttons accessible via Tab key
- Enter/Space to activate buttons
- Arrow keys for quick feedback chips
- Escape to close modals/menus

### Screen Reader Support
```tsx
<VoteButton
  aria-label={`Upvote this ${targetType}. Current count: ${upvotes}`}
  aria-pressed={userVote === 'up'}
  role="button"
/>

<MemoryCard
  aria-label={`Memory: ${content}. Relevance score: ${relevanceScore}%. ${pinned ? 'Pinned.' : ''}`}
/>
```

### Color Contrast
- All text meets WCAG AA contrast ratios
- Vote states indicated by both color AND icon changes
- Status indicators use icons in addition to colors

### Reduced Motion
```css
@media (prefers-reduced-motion: reduce) {
  .vote-animation {
    transition: none;
  }
  .count-animation {
    animation: none;
  }
}
```

---

## Testing Strategy

### Unit Tests
- Test each atom/molecule in isolation
- Verify accessibility requirements
- Test keyboard navigation
- Mock API responses for feedback submission

### Integration Tests
- Test feedback flow from UI to database
- Verify real-time sync across tabs/devices
- Test offline feedback queueing

### E2E Tests
- Complete user flows for voting
- Memory management workflows
- Note creation and editing
- Reconnection scenarios

### Performance Tests
- Vote latency under load (< 100ms)
- Memory search response time (< 200ms)
- UI responsiveness with 1000+ messages

---

## Mobile Support

The feedback system is designed to work across all Alicia clients:

### Web (Mobile Responsive)
- Touch-friendly voting buttons (minimum 44x44px)
- Swipe gestures for quick feedback
- Collapsible sections for small screens
- Bottom sheet modals for memory management

### Android App Integration
- Same protocol messages work via LiveKit
- Native UI components matching Material Design 3
- Haptic feedback on vote actions
- Offline feedback queue with Room database

### Component Sizing
```css
/* Mobile-first responsive design */
.vote-button {
  min-width: 44px;
  min-height: 44px;
  padding: 12px;
}

@media (min-width: 768px) {
  .vote-button {
    min-width: 32px;
    min-height: 32px;
    padding: 8px;
  }
}
```

---

## Related Documentation

- [Architecture Overview](ARCHITECTURE.md) - System architecture and component interaction
- [DSPy + GEPA Implementation Plan](dspy-gepa-implementation-plan.md) - How feedback improves the system
- [Protocol Specification](protocol/index.md) - Message format details
- [Database Schema](DATABASE.md) - Storage for votes, notes, and memories
- [Offline Sync](OFFLINE_SYNC.md) - How feedback syncs across devices

---

## Next Steps

1. Review and approve this plan
2. Design mockups for key components
3. Implement Phase 1 foundation (atoms and core infrastructure)
4. Integrate with DSPy/GEPA optimization pipeline
5. Iterate based on user testing
