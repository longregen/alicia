/**
 * Standardized z-index values for consistent layering across the application.
 *
 * Usage:
 * - Import the constant you need: import { Z_INDEX } from '@/constants/zIndex'
 * - Use in className: className={`z-[${Z_INDEX.SIDEBAR}]`}
 * - Or use the CSS variable: className="z-[var(--z-sidebar)]"
 *
 * Layer hierarchy (lowest to highest):
 * 1. OVERLAY (40) - Background overlays, dimmed backdrops
 * 2. SIDEBAR (50) - Slide-out sidebars, drawers
 * 3. HAMBURGER (60) - Mobile hamburger menu button
 * 4. DROPDOWN (70) - Dropdown menus, popovers
 * 5. MODAL (80) - Modal dialogs
 * 6. TOAST (90) - Toast notifications
 * 7. TOOLTIP (100) - Tooltips (always on top)
 * 8. ERROR_BANNER (1000) - Critical error banners (highest priority)
 */

export const Z_INDEX = {
  /** Background overlays and dimmed backdrops */
  OVERLAY: 40,
  /** Slide-out sidebars and drawers */
  SIDEBAR: 50,
  /** Mobile hamburger menu button */
  HAMBURGER: 60,
  /** Dropdown menus and popovers */
  DROPDOWN: 70,
  /** Modal dialogs */
  MODAL: 80,
  /** Toast notifications */
  TOAST: 90,
  /** Tooltips */
  TOOLTIP: 100,
  /** Critical error banners - highest priority */
  ERROR_BANNER: 1000,
} as const;
