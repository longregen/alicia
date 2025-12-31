# Alicia Frontend Style Guide

> **Visual Design Reference**: Use the look and feel of `frontend-mockup/` to guide the visual design. The mockup contains the canonical visual reference for spacing, colors, and component appearance.

## Design System Overview

This project uses **Tailwind CSS 4** with a semantic color system based on OKLCH color space. Colors automatically adapt to dark/light mode via `prefers-color-scheme`.

### Critical: Component CSS Classes

The following CSS component classes are defined in `src/index.css` and **must not be removed**. These provide consistent styling across the application:

| Class | Purpose |
|-------|---------|
| `.btn`, `.btn-primary`, `.btn-secondary`, `.btn-ghost`, `.btn-destructive` | Button styles |
| `.input` | Form input styling |
| `.tab`, `.tab.active` | Tab navigation |
| `.card`, `.card-hover` | Card containers |
| `.badge`, `.badge-success`, `.badge-error`, `.badge-warning`, `.badge-neutral` | Status badges |

**Always use these classes** instead of writing inline Tailwind for common components. If you need to modify them, update `src/index.css` rather than overriding with inline styles.

## Color Tokens

### Backgrounds
| Class | Usage |
|-------|-------|
| `bg-app` | Page/app background |
| `bg-surface` | Cards, panels |
| `bg-elevated` | Modals, dropdowns, popovers |
| `bg-sunken` | Inset areas, disabled states |
| `bg-overlay` | Modal backdrops |

### Text
| Class | Usage |
|-------|-------|
| `text-default` | Primary text |
| `text-muted` | Secondary text, labels |
| `text-subtle` | Tertiary text, placeholders |
| `text-on-emphasis` | Text on accent backgrounds |

### Accent (Interactive)
| Class | Usage |
|-------|-------|
| `bg-accent` | Primary buttons, active states |
| `bg-accent-hover` | Hover state for accent |
| `bg-accent-active` | Pressed state |
| `bg-accent-subtle` | Light accent background (selected items) |
| `text-accent` | Accent colored text, links |
| `border-accent` | Accent borders, focus rings |

### Status Colors
| Class | Usage |
|-------|-------|
| `bg-success` / `text-success` | Success states, connected, complete |
| `bg-success-subtle` | Light success background |
| `bg-error` / `text-error` | Errors, destructive actions |
| `bg-error-subtle` | Light error background |
| `bg-warning` / `text-warning` | Warnings, caution states |
| `bg-warning-subtle` | Light warning background |

### Special Indicators
| Class | Usage |
|-------|-------|
| `bg-reasoning` | AI reasoning blocks |
| `bg-tool-use` | Tool invocation indicators |
| `bg-tool-result` | Tool result indicators |

### Borders
| Class | Usage |
|-------|-------|
| `border` | Default border (uses `--color-border`) |
| `border-muted` | Subtle borders |
| `border-emphasis` | Strong borders |
| `border-accent` | Accent colored borders |

## Component Patterns

### Buttons (use `.btn` classes)

```tsx
// Primary button - use for main actions
<button className="btn btn-primary">Submit</button>

// Secondary button - use for secondary actions
<button className="btn btn-secondary">Cancel</button>

// Ghost button - use for subtle/text-like buttons
<button className="btn-ghost">Close</button>

// Destructive button - use for dangerous actions
<button className="btn btn-destructive">Delete</button>
```

### Inputs (use `.input` class)

```tsx
<input className="input" placeholder="Enter text..." />
<select className="input">...</select>
<textarea className="input" rows={3} />
```

### Tabs (use `.tab` class)

```tsx
<div className="flex gap-1">
  <button className={`tab ${activeTab === 'one' ? 'active' : ''}`}>Tab One</button>
  <button className={`tab ${activeTab === 'two' ? 'active' : ''}`}>Tab Two</button>
</div>
```

### Cards (use `.card` class)

```tsx
// Basic card
<div className="card p-4">
  <h3 className="text-foreground font-medium">Title</h3>
  <p className="text-muted-foreground text-sm">Description</p>
</div>

// Interactive card with hover effect
<div className="card card-hover p-4">...</div>

// Card without border (for nested sections)
<div className="bg-card rounded-lg p-4">...</div>
```

### Status Badges (use `.badge` classes)

```tsx
<span className="badge badge-success">Connected</span>
<span className="badge badge-error">Failed</span>
<span className="badge badge-warning">Pending</span>
<span className="badge badge-neutral">Draft</span>
```

### Modals

```tsx
// Backdrop
<div className="fixed inset-0 bg-overlay" />

// Modal content
<div className="bg-elevated rounded-xl shadow-xl p-6">
  {/* content */}
</div>
```

### Form Inputs

```tsx
// Use the .input class for consistent styling
<input className="input" placeholder="Enter value..." />

// For custom styling needs, extend rather than replace:
<input className="input w-64" />  // adds width constraint
```

## Animations

Available animation classes:
- `animate-fade-in` - Fade in (0.2s)
- `animate-slide-up` - Slide up with fade (0.3s)
- `animate-slide-in` - Slide in from right (0.3s)
- `animate-pulse-recording` - Pulsing effect for recording states

## Shadows

| Class | Usage |
|-------|-------|
| `shadow-sm` | Subtle elevation |
| `shadow-md` | Cards, buttons |
| `shadow-lg` | Dropdowns, popovers |
| `shadow-xl` | Modals |

## Best Practices

1. **Use component CSS classes** - Use `.btn`, `.input`, `.tab`, `.card`, `.badge` classes defined in `index.css`
2. **Reference frontend-mockup** - Check `frontend-mockup/` for visual design guidance before implementing new UI
3. **Always use semantic tokens** - Use `bg-surface` not `bg-gray-100`
4. **No hardcoded colors** - Never use hex values or `bg-blue-500` directly
5. **Dark mode is automatic** - Colors adapt via CSS custom properties
6. **Avoid harsh borders** - Use `bg-card` or `bg-secondary` for visual separation instead of `border border-default`
7. **Use the `cls()` utility** - For conditional class concatenation:
   ```tsx
   import { cls } from '../utils/cls';

   <div className={cls('bg-surface', isActive && 'bg-accent-subtle')} />
   ```
8. **Don't remove component classes** - The classes in `@layer components` of `index.css` are used throughout the app

## File Structure

- `src/index.css` - Design tokens in `@theme` block AND component classes in `@layer components`
- `tailwind.config.js` - Animations and spacing extensions
- `src/utils/cls.ts` - Class name utility
- `frontend-mockup/` - Visual design reference (canonical look and feel)

## Adding New Colors

Add to `@theme` block in `src/index.css`:

```css
@theme {
  /* Add new color scale */
  --color-purple-500: oklch(0.55 0.20 300);

  /* Or semantic color referencing existing scale */
  --color-info: var(--color-primary-400);
}
```

Then use as `bg-purple-500` or `bg-info` in components.
