/**
 * Common UI patterns and CSS class combinations
 * Reduces duplication of frequently used Tailwind class combinations
 */

import { cls } from './cls';

// Layout patterns
export const uiPatterns = {
  // Flex layouts
  flexCenter: 'flex items-center',
  flexCenterBetween: 'flex items-center justify-between',
  flexCenterGap: (gap: number) => `flex items-center gap-${gap}`,
  flexCol: 'flex flex-col',
  flexColGap: (gap: number) => `flex flex-col gap-${gap}`,

  // Text styles
  textMuted: 'text-sm text-surface-400',
  textPrimary: 'text-sm text-primary-text',
  textSecondary: 'text-xs text-surface-500',
  textLabel: 'text-sm font-medium text-primary-text whitespace-nowrap',

  // Container styles
  cardContainer: 'p-4 bg-surface-900 rounded-lg',
  bubbleContainer: 'max-w-xs rounded-2xl px-4 py-2',

  // Transitions
  transition: 'transition-all duration-200',
  transitionFast: 'transition-all duration-100',

  // Button states
  buttonDisabled: 'opacity-50 cursor-not-allowed',
  buttonHover: 'hover:bg-surface-700',

  // Icon sizes
  iconSm: 'w-4 h-4',
  iconMd: 'w-5 h-5',
  iconLg: 'w-6 h-6',

  // Loading/Animation
  loadingDot: 'w-2 h-2 bg-surface-400 rounded-full animate-bounce',
  pulseAnimation: 'animate-pulse',

  // Badge styles
  badge: 'text-xs px-2 py-0.5 rounded-full',
  badgeSuccess: 'bg-green-500/20 text-green-400',
  badgeError: 'bg-red-500/20 text-red-400',
  badgeWarning: 'bg-yellow-500/20 text-yellow-400',
  badgeInfo: 'bg-blue-500/20 text-blue-400',
};

// Composite patterns
export const compositePatterns = {
  // Message layout
  messageContainer: () => cls(uiPatterns.flexCenterGap(3), 'max-w-4xl'),

  // Input group
  inputGroup: () => cls(uiPatterns.flexCenter, uiPatterns.flexCenterGap(2), 'p-2'),

  // Tool badge
  toolBadge: (status: 'success' | 'error' | 'running') => cls(
    uiPatterns.badge,
    status === 'success' ? uiPatterns.badgeSuccess :
    status === 'error' ? uiPatterns.badgeError :
    uiPatterns.badgeInfo
  ),

  // Loading indicator
  typingIndicator: () => cls(
    uiPatterns.flexCenter,
    uiPatterns.flexCenterGap(1),
    'px-4 py-2'
  ),
};

// Export convenience functions
export const flexCenter = () => uiPatterns.flexCenter;
export const flexCenterBetween = () => uiPatterns.flexCenterBetween;
export const flexCenterGap = (gap: number) => uiPatterns.flexCenterGap(gap);
export const textMuted = () => uiPatterns.textMuted;
export const textPrimary = () => uiPatterns.textPrimary;
export const cardContainer = () => uiPatterns.cardContainer;
