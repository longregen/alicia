# Frontend Mockup Migration Plan

## Overview

Migrate the current frontend to match the mockup's visual design while:
- Keeping atomic design structure (atoms/molecules/organisms)
- Adopting shadcn/ui-like patterns (CVA variants, cn() utility)
- Migrating to mockup's OKLCH color system
- Preserving current feedback behavior (just updating looks)

## Phase 1: Foundation (Color System & Utilities)

### 1.1 Color System Migration
**File:** `frontend/src/index.css`

Replace current `@theme` block with mockup's OKLCH variables:

```css
:root {
  --background: oklch(0.13 0.005 250);
  --foreground: oklch(0.95 0 0);
  --card: oklch(0.16 0.005 250);
  --primary: oklch(0.65 0.2 250);
  --secondary: oklch(0.22 0.005 250);
  --muted: oklch(0.25 0.005 250);
  --muted-foreground: oklch(0.65 0 0);
  --accent: oklch(0.55 0.15 160);
  --destructive: oklch(0.55 0.2 25);
  --border: oklch(0.25 0.005 250);
  --input: oklch(0.2 0.005 250);
  --ring: oklch(0.65 0.2 250);
  --radius: 0.75rem;

  /* Sidebar-specific */
  --sidebar: oklch(0.11 0.005 250);
  --sidebar-border: oklch(0.2 0.005 250);
  --sidebar-accent: oklch(0.18 0.005 250);

  /* Voice states */
  --voice-active: oklch(0.65 0.2 160);
  --voice-listening: oklch(0.7 0.2 250);
  --voice-processing: oklch(0.7 0.15 80);

  /* Status */
  --success: oklch(0.55 0.15 160);
  --warning: oklch(0.7 0.15 80);

  /* Charts */
  --chart-1: oklch(0.65 0.2 250);
  --chart-2: oklch(0.55 0.15 160);
  --chart-3: oklch(0.7 0.15 80);
  --chart-4: oklch(0.6 0.2 300);
  --chart-5: oklch(0.55 0.2 25);
}
```

Map existing semantic tokens to new variables for backwards compatibility during migration.

### 1.2 Utility Functions
**File:** `frontend/src/lib/utils.ts` (new)

```typescript
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

Add dependencies: `clsx`, `tailwind-merge`

### 1.3 Update STYLE.md
**File:** `docs/STYLE.md`

Document new color system, cn() usage, CVA patterns, component conventions.

---

## Phase 2: UI Primitives (Atoms)

Port/update these atoms to shadcn/ui patterns with CVA:

| Current | Action | Notes |
|---------|--------|-------|
| Button | Update | Add CVA variants (default, destructive, outline, secondary, ghost, link), sizes (default, sm, lg, icon) |
| Badge | Update | Add CVA variants, keep status mapping |
| IconButton | Merge | Consolidate into Button with `size="icon"` |
| GhostButton | Merge | Consolidate into Button with `variant="ghost"` |
| PrimaryButton | Remove | Use `<Button variant="default">` |
| Toast | Update | Match mockup styling |
| Tooltip | Update | Use Radix pattern, add arrow |
| ToggleSwitch | Update | Match Switch component style |

### New Atoms to Add:
| Component | Source |
|-----------|--------|
| Input | `frontend-mockup/components/ui/input.tsx` |
| Textarea | `frontend-mockup/components/ui/textarea.tsx` |
| Label | `frontend-mockup/components/ui/label.tsx` |
| Card | `frontend-mockup/components/ui/card.tsx` |
| Separator | `frontend-mockup/components/ui/separator.tsx` |
| ScrollArea | `frontend-mockup/components/ui/scroll-area.tsx` |
| Skeleton | `frontend-mockup/components/ui/skeleton.tsx` |
| Spinner | `frontend-mockup/components/ui/spinner.tsx` |
| Progress | `frontend-mockup/components/ui/progress.tsx` |
| Kbd | `frontend-mockup/components/ui/kbd.tsx` |
| Avatar | `frontend-mockup/components/ui/avatar.tsx` |
| Slider | `frontend-mockup/components/ui/slider.tsx` |
| Checkbox | `frontend-mockup/components/ui/checkbox.tsx` |
| RadioGroup | `frontend-mockup/components/ui/radio-group.tsx` |
| Select | `frontend-mockup/components/ui/select.tsx` |

---

## Phase 3: Overlay Components (Atoms/Molecules)

| Component | Source | Notes |
|-----------|--------|-------|
| Dialog | mockup | Modal dialogs |
| Popover | mockup | For feedback popovers |
| DropdownMenu | mockup | Context menus |
| Sheet | mockup | Side panels |
| AlertDialog | mockup | Confirmations |
| Command | mockup | Command palette |
| Collapsible | mockup | For reasoning blocks |

---

## Phase 4: Layout Components

### 4.1 Resizable Sidebar
**File:** `frontend/src/components/organisms/Sidebar.tsx`

**State Integration:** Use existing `conversationStore` from `frontend/src/stores/conversationStore.ts`

Features to add:
- Collapsible (64px collapsed, 200-480px expanded)
- Resize handle with drag
- Keyboard shortcut (⌘B toggle)
- Search with ⌘K shortcut
- Conversation list with:
  - Active/Archived sections
  - Inline rename
  - Context menu (archive, delete)
  - Time formatting ("5m ago")
- Bottom navigation icons (Memory, Server, Settings)
- Connection status indicator

### 4.2 Panel Navigation
**File:** `frontend/src/components/organisms/AliciaApp.tsx` (new or update App.tsx)

```typescript
type Panel = "chat" | "memory" | "server" | "settings";
const [activePanel, setActivePanel] = useState<Panel>("chat");
```

Layout structure:
```
┌─────────────────────────────────────────────┐
│ Sidebar │ Main Content (panel-based)        │
│         │                                   │
│ [convs] │ ChatArea | MemoryPanel |          │
│         │ ServerPanel | SettingsPanel       │
│         │                                   │
│ [nav]   │                                   │
└─────────────────────────────────────────────┘
```

---

## Phase 5: Chat Components

### 5.1 Message Branching
**Files:** `ChatBubble.tsx`, `MessageBubble.tsx`

**Backend Status:** NOT currently supported. Has `previous_id` for linear chains but no sibling/branch support. Would need schema changes for full persistence.

**Implementation:** UI-only (local React state) for now. Structure ready for backend integration later.

Add to local state (not persisted):
```typescript
// Local state in ChatArea or parent component
const [messageBranches, setMessageBranches] = useState<Map<string, {
  siblings: string[];      // IDs of alternative versions
  currentIndex: number;    // Current position
  versions: Message[];     // Local versions array
}>>(new Map());
```

UI additions:
- Branch navigator: `< 1/3 >` with chevron buttons
- Edit button on messages
- For user messages: creates new local branch (not sent to backend)
- For assistant messages: treated as correction feedback via existing voting system

**Future backend work needed:**
- Add `parent_id` column to `alicia_messages`
- New `alicia_message_branches` table
- API endpoints: `/regenerate`, `/correct`, `/branches`, `/promote`

### 5.2 Inline Editing
- Click edit icon → Textarea replaces content
- Save/Cancel buttons
- User edits create new sibling branch
- Assistant edits submit as correction Commentary

### 5.3 Feedback Popover (visual update only)
**Current behavior kept**, update visuals:
- Use Popover component instead of inline
- Radio group for feedback type
- Optional comment textarea
- Match mockup styling

### 5.4 Tool/Memory Badges
Update `ToolUseCard` and `MemoryTraceAddon`:
- Pill badge style (rounded-full)
- Status dot indicator
- Consistent icon + name + status pattern

### 5.5 Chat Header
Add header bar with:
- Conversation title
- Conversation ID badge
- Audio toggle button

### 5.6 Voice Visualizer
Port `frontend-mockup/components/voice-visualizer.tsx`:
- Wave animation bars
- State-based colors (listening, processing, speaking)
- Pulse ring animation

---

## Phase 6: Panel Components

### 6.1 Memory Panel
**File:** `frontend/src/components/organisms/MemoryManager/` (update)

Structure:
- Header: Icon + "Memory Management" + count badge + "Add Memory" button
- Search bar
- Type filter buttons (preference, fact, instruction, context)
- 2-column card grid with:
  - Type badge with color coding
  - Pin indicator
  - Content text
  - Tags
  - Importance score + usage count
  - Dropdown menu (pin, edit, archive, delete)

### 6.2 Settings Panel
**File:** `frontend/src/components/Settings.tsx` (update)

Structure:
- Header: Icon + "Settings" + category count + "Reset All" button
- 2-column card grid:
  - Voice & Audio (switches, slider)
  - Memory (switches)
  - Appearance (theme select, switches)
  - Notifications (switches)
  - Keyboard Shortcuts (kbd display)
  - Privacy & Data (buttons)

### 6.3 Server Panel
**File:** `frontend/src/components/organisms/ServerPanel/` (update)

Port mockup patterns for server info display.

---

## Phase 7: Update Exports & Tests

### 7.1 Update atoms/index.ts
- Remove deprecated components (PrimaryButton, etc.)
- Add new atoms
- Update type exports

### 7.2 Update molecules/index.ts
- Add new molecules
- Update type exports

### 7.3 Update Tests
- Update component tests for new props/variants
- Add tests for new components
- Update snapshot tests

---

## Implementation Order

1. **Foundation** - Color system, cn() utility, STYLE.md
2. **Core Atoms** - Button (with CVA), Badge, Input, Card, Label
3. **More Atoms** - Tooltip, Switch, Separator, ScrollArea, etc.
4. **Overlays** - Dialog, Popover, DropdownMenu
5. **Sidebar** - Resizable, collapsible, conversation list
6. **Panel Navigation** - AliciaApp layout restructure
7. **Chat Updates** - Branching, editing, header, voice viz
8. **Memory Panel** - Full port
9. **Settings Panel** - Full port
10. **Server Panel** - Full port
11. **Cleanup** - Remove deprecated, update exports, tests

---

## Files to Modify

### Core Files
- `frontend/src/index.css` - Color system
- `frontend/src/lib/utils.ts` - New cn() utility
- `docs/STYLE.md` - Documentation update

### Atoms (~15 files)
- `frontend/src/components/atoms/Button.tsx` - CVA refactor
- `frontend/src/components/atoms/Badge.tsx` - CVA refactor
- `frontend/src/components/atoms/Toast.tsx` - Style update
- `frontend/src/components/atoms/Tooltip.tsx` - Radix pattern
- `frontend/src/components/atoms/ToggleSwitch.tsx` - Style update
- `frontend/src/components/atoms/MessageBubble.tsx` - Branching UI
- New: Input, Textarea, Label, Card, Separator, ScrollArea, Skeleton, Spinner, Progress, Kbd, Avatar, Slider

### Molecules (~5 files)
- `frontend/src/components/molecules/ChatBubble.tsx` - Branching, editing
- New: Popover-based feedback, DropdownMenu items

### Organisms (~8 files)
- `frontend/src/components/Sidebar.tsx` - Full refactor
- `frontend/src/components/organisms/ChatWindow.tsx` - Header, voice viz
- `frontend/src/components/Settings.tsx` - Full refactor
- `frontend/src/components/organisms/MemoryManager/` - Full refactor
- `frontend/src/components/organisms/ServerPanel/` - Update
- New: AliciaApp.tsx (or update App.tsx)

### Overlays (~6 new files)
- Dialog, Popover, DropdownMenu, Sheet, AlertDialog, Collapsible

---

## Dependencies to Add (Fresh Install)

All Radix dependencies will be installed fresh. Run:

```bash
pnpm add clsx tailwind-merge class-variance-authority \
  @radix-ui/react-dialog @radix-ui/react-popover \
  @radix-ui/react-dropdown-menu @radix-ui/react-tooltip \
  @radix-ui/react-scroll-area @radix-ui/react-collapsible \
  @radix-ui/react-switch @radix-ui/react-slider \
  @radix-ui/react-checkbox @radix-ui/react-radio-group \
  @radix-ui/react-select @radix-ui/react-separator \
  @radix-ui/react-avatar @radix-ui/react-label \
  @radix-ui/react-progress
```

```json
{
  "clsx": "^2.x",
  "tailwind-merge": "^2.x",
  "class-variance-authority": "^0.7.x",
  "@radix-ui/react-dialog": "^1.x",
  "@radix-ui/react-popover": "^1.x",
  "@radix-ui/react-dropdown-menu": "^2.x",
  "@radix-ui/react-tooltip": "^1.x",
  "@radix-ui/react-scroll-area": "^1.x",
  "@radix-ui/react-collapsible": "^1.x",
  "@radix-ui/react-switch": "^1.x",
  "@radix-ui/react-slider": "^1.x",
  "@radix-ui/react-checkbox": "^1.x",
  "@radix-ui/react-radio-group": "^1.x",
  "@radix-ui/react-select": "^2.x",
  "@radix-ui/react-separator": "^1.x",
  "@radix-ui/react-avatar": "^1.x",
  "@radix-ui/react-label": "^2.x",
  "@radix-ui/react-progress": "^1.x"
}
```

---

## Estimated Component Count

| Category | New | Updated | Removed | Total |
|----------|-----|---------|---------|-------|
| Atoms | 15 | 6 | 3 | 18 net new |
| Molecules | 2 | 3 | 0 | 5 changes |
| Organisms | 1 | 6 | 0 | 7 changes |
| Overlays | 6 | 0 | 0 | 6 new |
| **Total** | **24** | **15** | **3** | **36 changes** |
