# Alicia Frontend Style Guide

## Design System Overview

This project uses **Tailwind CSS 4** with a semantic color system based on OKLCH color space. Colors automatically adapt to dark/light mode via `prefers-color-scheme`.

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

### Buttons

```tsx
// Primary button
<button className="bg-accent hover:bg-accent-hover text-on-emphasis px-4 py-2 rounded-lg">
  Submit
</button>

// Secondary button
<button className="bg-surface hover:bg-elevated text-default border px-4 py-2 rounded-lg">
  Cancel
</button>

// Danger button
<button className="bg-error hover:bg-error-600 text-on-emphasis px-4 py-2 rounded-lg">
  Delete
</button>
```

### Cards

```tsx
<div className="bg-surface rounded-lg p-4 shadow-md">
  <h3 className="text-default font-medium">Title</h3>
  <p className="text-muted text-sm">Description</p>
</div>
```

### Status Badges

```tsx
// Success
<span className="bg-success-subtle text-success px-2 py-1 rounded text-sm">
  Connected
</span>

// Error
<span className="bg-error-subtle text-error px-2 py-1 rounded text-sm">
  Failed
</span>

// Warning
<span className="bg-warning-subtle text-warning px-2 py-1 rounded text-sm">
  Pending
</span>
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
<input
  className="bg-surface border rounded-lg px-3 py-2 text-default
             placeholder:text-subtle
             focus:border-accent focus:ring-2 focus:ring-accent focus:outline-none"
/>
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

1. **Always use semantic tokens** - Use `bg-surface` not `bg-gray-100`
2. **No hardcoded colors** - Never use hex values or `bg-blue-500` directly
3. **Dark mode is automatic** - Colors adapt via CSS custom properties
4. **Use the `cls()` utility** - For conditional class concatenation:
   ```tsx
   import { cls } from '../utils/cls';

   <div className={cls('bg-surface', isActive && 'bg-accent-subtle')} />
   ```

## File Structure

- `src/index.css` - Design tokens defined in `@theme` block
- `tailwind.config.js` - Animations and spacing extensions
- `src/utils/cls.ts` - Class name utility

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
