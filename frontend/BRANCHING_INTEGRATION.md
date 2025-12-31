# Message Branching Integration

## Overview

Message branching support has been successfully integrated into the message list, allowing users to create and navigate between alternative versions of their messages.

## Architecture

### Components

1. **BranchNavigator** (`frontend/src/components/atoms/BranchNavigator.tsx`)
   - UI component showing "< 1/3 >" style navigation
   - Only renders when `totalBranches > 1`
   - Provides prev/next navigation buttons
   - Disables buttons at boundaries (first/last branch)

2. **ChatBubble** (`frontend/src/components/molecules/ChatBubble.tsx`)
   - Integrates BranchNavigator for user messages (lines 549-556)
   - Manages branch content display via `useBranchStore`
   - Shows edit button on hover for creating new branches
   - Initializes branches on mount for user messages

3. **branchStore** (`frontend/src/stores/branchStore.ts`)
   - Zustand store managing branch state
   - Stores branches as `Map<MessageId, MessageBranch[]>`
   - Tracks current version index per message
   - UI-only state (not persisted to backend)

### Data Flow

```
User edits message
  ↓
ChatBubble.handleSaveEdit()
  ↓
branchStore.createBranch(messageId, newContent)
  ↓
Branch count increases, current index updates
  ↓
ChatBubble re-renders with new branch content
  ↓
BranchNavigator shows "2/2" (if 2 branches)
```

## Key Features

### Branch Creation
- User messages are automatically initialized with a single branch on mount
- Editing a user message creates a new branch (via `createBranch`)
- Each branch stores: `{ content: string, createdAt: Date }`

### Branch Navigation
- BranchNavigator appears when `branchCount > 1`
- Previous/Next buttons navigate through branches
- Current branch content is displayed in the message bubble
- Branch counter shows position (e.g., "2/3")

### Branch Display
- Only **user messages** support branching (assistant messages do not)
- Current branch content overrides the original message content
- Branches are ordered by creation time
- Navigation is cyclical-aware (buttons disable at boundaries)

## State Management

### Branch Store State
```typescript
{
  branches: Map<MessageId, MessageBranch[]>,
  currentVersionIndex: Map<MessageId, number>,
}
```

### Key Methods
- `initializeBranch(messageId, content)` - Create initial branch
- `createBranch(messageId, content)` - Add new branch version
- `navigateToBranch(messageId, direction)` - Move between branches
- `getCurrentBranch(messageId)` - Get active branch content
- `getBranchCount(messageId)` - Get total number of branches
- `getCurrentIndex(messageId)` - Get current position (0-based)

## UI Behavior

### Branch Navigator Visibility
- **Shows**: User messages with 2+ branches
- **Hidden**: Assistant messages, messages with ≤1 branch

### Edit Button
- Shows on hover for completed messages (not streaming)
- Located in top corner of message bubble
- Triggers edit mode with textarea
- Save creates new branch, Cancel discards changes

### Branch Counter Format
- Displays as "current/total" (e.g., "1/3")
- Uses 1-based indexing for display (0-based internally)
- Monospace font for alignment

## Testing

### Component Tests
- `BranchNavigator.test.tsx` - 8 tests for UI component
- `ChatBubble.branching.test.tsx` - 16 tests for integration
- `branchStore.test.ts` - 19 tests for state management

### Test Coverage
- Branch creation and navigation
- Content display and switching
- Button enable/disable states
- State persistence across re-renders
- Counter display accuracy

### Total Tests: 1346 passing (including 43 branching-specific)

## Limitations

### Current Implementation
1. **UI-only**: Branches not persisted to backend
2. **User messages only**: Assistant messages don't support branching
3. **No merge**: Can't combine branches
4. **Linear history**: No tree visualization
5. **Memory only**: Branches lost on page refresh

### Future Enhancements (Phase 5.2+)
- Backend persistence
- Branch labels/descriptions
- Tree visualization for complex histories
- Branch comparison view
- Merge/cherry-pick operations

## Files Modified

- `frontend/src/components/atoms/BranchNavigator.tsx` - UI component (already existed)
- `frontend/src/components/molecules/ChatBubble.tsx` - Integration (lines 173-194, 549-556)
- `frontend/src/stores/branchStore.ts` - State management (already existed)
- `frontend/src/components/molecules/ChatBubble.branching.test.tsx` - Integration tests (new)

## Usage Example

```typescript
// User edits message "Hello" to "Hi there"
const { createBranch } = useBranchStore();
createBranch(messageId, "Hi there");

// Navigate between versions
const { navigateToBranch } = useBranchStore();
navigateToBranch(messageId, 'prev'); // Back to "Hello"
navigateToBranch(messageId, 'next'); // Forward to "Hi there"

// Get current branch
const { getCurrentBranch } = useBranchStore();
const current = getCurrentBranch(messageId);
console.log(current.content); // "Hi there"
```

## Implementation Notes

1. **Initialization**: ChatBubble automatically calls `initializeBranch` for user messages on mount (useEffect on line 184-188)

2. **Content Priority**: Effective content = `currentBranch?.content || props.content` (line 196)

3. **Navigator Position**: Rendered in the footer alongside timestamp and addons (lines 549-556)

4. **Edit Mode**: Controlled by `isEditing` state, toggled by edit button click (lines 303-323)

5. **Type Safety**: Uses branded types (`MessageId`) for type-safe message identification

## Performance Considerations

- Branch state is memoized in Zustand store
- Only re-renders affected messages on branch changes
- Navigator conditionally renders (null when ≤1 branch)
- Minimal overhead for messages without branches

## Accessibility

- Edit button has `aria-label="Edit message"`
- Branch buttons have `aria-label="Previous/Next branch"`
- Disabled state clearly indicated
- Keyboard navigation supported (buttons are focusable)

## Next Steps

Per FRONTEND_PLAN.md Phase 5.2:
- Backend API for branch persistence
- Server-side branch storage in conversations table
- Sync branches across devices
- Add branch metadata (timestamps, labels)
