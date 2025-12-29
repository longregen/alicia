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
│   └── VoteControl.tsx       # Thumbs up/down with count
├── NoteEditor/
│   └── InlineNoteEditor.tsx  # Add/edit notes inline
├── MemoryCard/
│   ├── MemoryCard.tsx        # Single memory display
│   └── MemoryActions.tsx     # Pin, delete, edit actions
├── FeedbackPanel/
│   └── MessageFeedback.tsx   # Vote + note combined
├── ReasoningTrace/
│   └── ReasoningStep.tsx     # Single reasoning step
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

### 1. Response Voting System

#### UX Design
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

#### Component: `VoteControl.tsx`
```typescript
interface VoteControlProps {
  messageId: string;
  upvotes: number;
  downvotes: number;
  userVote: 'up' | 'down' | null;
  onVote: (vote: 'up' | 'down') => void;
  disabled?: boolean;
}
```

#### Interactions
- Click thumbs up/down to vote
- Click again to remove vote
- Clicking opposite thumb switches vote
- Visual feedback: filled icon = voted, outline = not voted
- Animate count change
- Show tooltip with breakdown on hover

#### Protocol Extension Needed
```typescript
// New envelope type
MessageVote (20): {
  messageId: string;
  conversationId: string;
  vote: 'up' | 'down' | 'remove';
  timestamp: number;
}

// Server response
VoteConfirmation (21): {
  messageId: string;
  upvotes: number;
  downvotes: number;
  userVote: 'up' | 'down' | null;
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

#### Complete UX Design
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
│ 🔧 Tools Used                                      [▼]   │
│   └─ read_file: src/index.ts ✓                          │
├─────────────────────────────────────────────────────────┤
│ 🧠 Memories (2)                                    [▼]   │
│   ├─ "User prefers TypeScript" (95%)                    │
│   └─ "Working on Node.js project" (87%)                 │
├─────────────────────────────────────────────────────────┤
│ 💭 Reasoning                                       [▼]   │
│   └─ 3 reasoning steps                                  │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 👍 12  │ 👎 2  │ 📝 2 notes │ [Regenerate] │ [⋮]   │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 1: Foundation (Atoms & Core Infrastructure)
**Priority: Critical**

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

4. **Create React Contexts**
   - [ ] `FeedbackContext`: Votes, notes state
   - [ ] `MemoryContext`: Memory management state
   - [ ] `ServerInfoContext`: Server information state

### Phase 2: Voting System
**Priority: High**

1. **Components**
   - [ ] VoteControl molecule
   - [ ] VoteAnimation (micro-interaction)

2. **Hooks**
   - [ ] `useVoting()`: Vote state and actions
   - [ ] `useOptimisticVote()`: Optimistic UI updates

3. **Integration**
   - [ ] Add VoteControl to MessageBubble
   - [ ] Wire up protocol handlers
   - [ ] Add to sync system

### Phase 3: Notes System
**Priority: High**

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
**Priority: High**

1. **Components**
   - [ ] MemoryCard molecule
   - [ ] MemoryEditor molecule
   - [ ] MemoryList organism
   - [ ] MemorySearch molecule
   - [ ] MemoryManager organism
   - [ ] InlineMemoryTrace (for messages)

2. **Hooks**
   - [ ] `useMemories()`: Memory CRUD
   - [ ] `useMemorySearch()`: Search/filter
   - [ ] `useMemoryFeedback()`: Rate memory relevance

3. **Integration**
   - [ ] Add MemoryManager to sidebar
   - [ ] Enhance ProtocolDisplay with memory actions
   - [ ] Create memory creation flow

### Phase 5: Server Information Panel
**Priority: Medium**

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
**Priority: Medium**

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
**Priority: Medium**

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
   - [ ] Optimistic updates everywhere

---

## API Endpoints Needed (Backend)

```
# Voting
POST   /api/v1/messages/{id}/vote          # Vote on message
DELETE /api/v1/messages/{id}/vote          # Remove vote
GET    /api/v1/messages/{id}/votes         # Get vote counts

# Notes
POST   /api/v1/messages/{id}/notes         # Add note
GET    /api/v1/messages/{id}/notes         # Get notes for message
PUT    /api/v1/notes/{id}                  # Update note
DELETE /api/v1/notes/{id}                  # Delete note
GET    /api/v1/conversations/{id}/notes    # All notes in conversation

# Memories
POST   /api/v1/conversations/{id}/memories # Create memory
GET    /api/v1/conversations/{id}/memories # List memories
PUT    /api/v1/memories/{id}               # Update memory
DELETE /api/v1/memories/{id}               # Delete memory
POST   /api/v1/memories/{id}/pin           # Pin/unpin memory
POST   /api/v1/memories/{id}/archive       # Archive memory
POST   /api/v1/memories/{id}/feedback      # Rate memory relevance

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

## Success Metrics

1. **Engagement**
   - % of messages with votes
   - Average notes per conversation
   - Memory creation rate

2. **Quality**
   - Positive vote ratio over time
   - Note categories distribution
   - Memory relevance scores

3. **Technical**
   - Vote latency < 100ms
   - Note sync < 500ms
   - Memory search < 200ms

---

## Open Questions

1. Should votes be visible to other users (if multi-user)?
2. Should notes be private or shared?
3. Memory scope: conversation vs global?
4. Should we show aggregate feedback to users?
5. Rate limiting for votes/notes?

---

## Next Steps

1. Review and approve this plan
2. Design mockups for key components (Figma)
3. Implement Phase 1 foundation
4. Iterate based on user testing
