// Common CSS class constants in camelCase
export const CSS = {
  // Layout
  flex: 'flex',
  flexCol: 'flex-col',
  flexRow: 'flex-row',
  itemsCenter: 'items-center',
  itemsStart: 'items-start',
  justifyCenter: 'justify-center',
  justifyBetween: 'justify-between',
  gap1: 'gap-1',
  gap2: 'gap-2',
  gap3: 'gap-3',
  gap4: 'gap-4',
  gap6: 'gap-6',
  spaceY2: 'space-y-2',
  spaceY3: 'space-y-3',
  spaceY4: 'space-y-4',
  spaceY6: 'space-y-6',
  spaceY8: 'space-y-8',

  // Spacing
  p2: 'p-2',
  p3: 'p-3',
  p4: 'p-4',
  p6: 'p-6',
  px2: 'px-2',
  px3: 'px-3',
  px4: 'px-4',
  py1: 'py-1',
  py2: 'py-2',
  py3: 'py-3',
  m2: 'm-2',
  mb2: 'mb-2',
  mt1: 'mt-1',
  mt2: 'mt-2',
  mt3: 'mt-3',
  mt5: 'mt-5',
  pl3: 'pl-3',

  // Sizing
  wFull: 'w-full',
  hFull: 'h-full',
  minW0: 'min-w-0',

  // Colors - Backgrounds (from STYLE.md)
  bgMainBg: 'bg-main-bg',
  bgContainerBg: 'bg-container-bg',
  bgSurfaceBg: 'bg-surface-bg',
  bgMessageReceived: 'bg-message-received-bg',
  bgMessageSent: 'bg-message-sent-bg',

  // Colors - Brand (from STYLE.md)
  bgPrimaryBlue: 'bg-primary-blue',
  bgPrimaryBlueHover: 'bg-primary-blue-hover',
  bgPrimaryBlueActive: 'bg-primary-blue-active',
  bgPrimaryBlueGlow: 'bg-primary-blue-glow',

  // Colors - States (from STYLE.md)
  bgActiveSpeaking: 'bg-active-speaking',
  bgInactiveDisabled: 'bg-inactive-disabled',
  bgError: 'bg-error',

  // Colors - Text (from STYLE.md)
  textPrimary: 'text-primary-text',
  textWhite: 'text-white-text',
  textMuted: 'text-muted-text',

  // Colors - Special (from STYLE.md)
  bgReasoning: 'bg-reasoning',
  bgToolUse: 'bg-tool-use',
  bgToolResult: 'bg-tool-result',
  bgTranslationComplete: 'bg-translation-complete',

  textReasoning: 'text-reasoning',
  textToolUse: 'text-tool-use',
  textToolResult: 'text-tool-result',
  textTranslationComplete: 'text-translation-complete',

  textActiveSpeaking: 'text-active-speaking',
  textPrimaryBlue: 'text-primary-blue',
  textError: 'text-error',

  // Legacy color mappings (for backward compatibility)
  bgWhite: 'bg-white',
  bgSurface50: 'bg-surface-50',
  bgSurface100: 'bg-surface-100',
  bgSurface400: 'bg-surface-400',
  bgSurface800: 'bg-surface-800',
  bgSurface900: 'bg-surface-900',

  textSurface100: 'text-surface-100',
  textSurface400: 'text-surface-400',
  textSurface500: 'text-surface-500',
  textSurface600: 'text-surface-600',
  textSurface700: 'text-surface-700',
  textSurface900: 'text-surface-900',

  borderSurface200: 'border-surface-200',
  borderSurface300: 'border-surface-300',
  borderSurface600: 'border-surface-600',
  borderSurface700: 'border-surface-700',

  bgAlicia50: 'bg-alicia-50',
  bgAlicia500: 'bg-alicia-500',
  bgAlicia600: 'bg-alicia-600',
  textAlicia400: 'text-alicia-400',
  textAlicia500: 'text-alicia-500',
  textAlicia600: 'text-alicia-600',

  bgSuccess100: 'bg-success-100',
  bgSuccess500: 'bg-success-500',
  bgWarning100: 'bg-warning-100',
  bgError100: 'bg-error-100',
  bgError500: 'bg-error-500',

  textSuccess300: 'text-success-300',
  textSuccess700: 'text-success-700',
  textWarning300: 'text-warning-300',
  textWarning700: 'text-warning-700',
  textError300: 'text-error-300',
  textError700: 'text-error-700',

  // Dark mode variants
  darkBgSurface700: 'dark:bg-surface-700',
  darkBgSurface800: 'dark:bg-surface-800',
  darkBgSurface900: 'dark:bg-surface-900',

  darkTextSurface100: 'dark:text-surface-100',
  darkTextSurface300: 'dark:text-surface-300',
  darkTextSurface400: 'dark:text-surface-400',
  darkTextAlicia400: 'dark:text-alicia-400',

  darkBorderSurface600: 'dark:border-surface-600',
  darkBorderSurface700: 'dark:border-surface-700',

  // Interactive states
  hoverBgSurface100: 'hover:bg-surface-100',
  hoverBgAlicia600: 'hover:bg-alicia-600',
  hoverTextAlicia600: 'hover:text-alicia-600',
  hoverBgPrimaryBlue: 'hover:bg-primary-blue-hover',

  focusOutlineNone: 'focus:outline-none',
  focusRing2: 'focus:ring-2',
  focusRingAlicia500: 'focus:ring-alicia-500',

  // Borders and shapes
  border: 'border',
  border2: 'border-2',
  rounded: 'rounded',
  roundedLg: 'rounded-lg',
  roundedFull: 'rounded-full',

  // Typography
  textXs: 'text-xs',
  textSm: 'text-sm',
  textBase: 'text-base',
  textLg: 'text-lg',
  fontMedium: 'font-medium',
  fontSemibold: 'font-semibold',
  textCenter: 'text-center',

  // Transitions and animations
  transitionColors: 'transition-colors',
  transitionAll: 'transition-all',
  transitionTransform: 'transition-transform',
  duration200: 'duration-200',
  duration300: 'duration-300',

  // Animations
  animatePulse: 'animate-pulse',
  animateSpin: 'animate-spin',
  animateBounce: 'animate-bounce',
  animateBounceOnce: 'animate-bounce-once',

  // Utilities
  cursorPointer: 'cursor-pointer',
  cursorNotAllowed: 'cursor-not-allowed',
  selectNone: 'select-none',
  opacity50: 'opacity-50',
  opacity70: 'opacity-70',
  disabledOpacity50: 'disabled:opacity-50',
  disabledCursorNotAllowed: 'disabled:cursor-not-allowed',
} as const;

export const css = CSS;

// Size-based class maps
export const sizeClasses = {
  sm: {
    padding: 'px-2 py-1',
    text: 'text-xs',
    icon: 'w-3 h-3',
    button: 'px-3 py-1.5 text-xs',
  },
  md: {
    padding: 'px-3 py-2',
    text: 'text-sm',
    icon: 'w-4 h-4',
    button: 'px-4 py-2 text-sm',
  },
  lg: {
    padding: 'px-4 py-3',
    text: 'text-base',
    icon: 'w-5 h-5',
    button: 'px-6 py-3 text-base',
  },
} as const;

// Variant-based class maps
export const variantClasses = {
  default: {
    background: css.bgSurfaceBg,
    text: css.textPrimary,
    border: 'border-primary-blue-glow',
    hover: css.hoverBgPrimaryBlue,
  },
  primary: {
    background: css.bgPrimaryBlue,
    text: css.textWhite,
    border: 'border-primary-blue',
    hover: css.hoverBgPrimaryBlue,
  },
  success: {
    background: css.bgActiveSpeaking,
    text: css.textWhite,
    border: 'border-active-speaking',
    hover: 'hover:bg-active-speaking',
  },
  warning: {
    background: 'bg-warning-500',
    text: css.textWhite,
    border: 'border-warning-500',
    hover: 'hover:bg-warning-600',
  },
  error: {
    background: css.bgError,
    text: css.textWhite,
    border: 'border-red-500',
    hover: 'hover:bg-error',
  },
} as const;
