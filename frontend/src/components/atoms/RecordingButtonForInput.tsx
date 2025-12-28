import React, { useState } from 'react';
import { RECORDING_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import { css } from '../../utils/constants';
import { uiPatterns } from '../../utils/uiPatterns';
import type { BaseComponentProps, RecordingState, Size } from '../../types/components';

// Component-specific constants
const BUTTON_BASE_CLASSES = [
  'rounded-full',
  'border-2',
  css.transitionAll,
  css.duration200,
  'ease-in-out',
  css.flex,
  css.itemsCenter,
  css.justifyCenter,
  css.fontMedium,
  'relative',
  'overflow-hidden',
  css.focusOutlineNone,
  'active:transform',
  'active:scale-95',
] as const;

const SIZE_MAP = {
  sm: 'w-8 h-8 text-sm',
  md: 'w-10 h-10 text-base',
  lg: 'w-12 h-12 text-lg',
} as const;

const DISABLED_CLASSES = [
  css.bgInactiveDisabled,
  'border-inactive-disabled',
  css.textMuted,
  css.cursorNotAllowed,
] as const;

const STATE_CLASSES = {
  [RECORDING_STATES.RECORDING]: [
    css.bgActiveSpeaking,
    'hover:bg-active-speaking',
    'border-active-speaking',
    css.textWhite,
    'glow-recording',
    'animate-recording-pulse',
  ],
  [RECORDING_STATES.PROCESSING]: [
    css.bgPrimaryBlue,
    'border-primary-blue',
    css.textWhite,
    'cursor-wait',
    css.animatePulse,
    'glow-primary',
  ],
  [RECORDING_STATES.ERROR]: [
    css.bgError,
    'border-error',
    css.textWhite,
    'hover:bg-error',
    'glow-error',
  ],
  [RECORDING_STATES.COMPLETED]: [
    css.bgTranslationComplete,
    'border-translation-complete',
    css.textWhite,
    'hover:bg-translation-complete',
    'glow-success',
  ],
  [RECORDING_STATES.IDLE]: [
    css.bgSurfaceBg,
    'border-primary-blue-glow',
    css.textPrimary,
    'hover:bg-primary-blue-glow',
    'hover:border-primary-blue',
    'hover:text-white-text',
    'hover:glow-primary',
  ],
} as const;

const TOOLTIP_CLASSES = [
  'absolute',
  'bottom-full',
  'left-1/2',
  'transform',
  '-translate-x-1/2',
  'mb-2',
  css.px3,
  'py-1',
  css.bgMainBg,
  css.textPrimary,
  css.textXs,
  css.rounded,
  'whitespace-nowrap',
  css.opacity50,
  'group-hover:opacity-100',
  css.transitionAll,
  css.duration200,
  'pointer-events-none',
  'z-10',
] as const;

// Component props interface
export interface RecordingButtonForInputProps extends BaseComponentProps {
  /** Current recording state */
  state?: RecordingState;
  /** Callback fired when recording state should change */
  onToggleRecording?: (newState: RecordingState) => void;
  /** Click handler for the button */
  onClick?: () => void;
  /** Whether the button is disabled */
  disabled?: boolean;
  /** Size variant of the button */
  size?: Size;
  /** Show a tooltip */
  showTooltip?: boolean;
}

const RecordingButtonForInput: React.FC<RecordingButtonForInputProps> = ({
  state = RECORDING_STATES.IDLE,
  onToggleRecording,
  onClick,
  showTooltip = false,
  disabled = false,
  size = 'md',
  className = ''
}) => {
  const [isPressed, setIsPressed] = useState<boolean>(false);

  const getButtonClasses = (): string => {
    const baseClasses = [
      BUTTON_BASE_CLASSES,
      SIZE_MAP[size],
    ];

    if (disabled) {
      return cls(baseClasses, DISABLED_CLASSES);
    }

    return cls(baseClasses, STATE_CLASSES[state]);
  };

  const getIcon = (): React.JSX.Element => {
    switch (state) {
      case RECORDING_STATES.RECORDING:
        return (
          <div className={cls('w-3', 'h-3', 'bg-white', 'rounded-full', css.animatePulse)} />
        );

      case RECORDING_STATES.PROCESSING:
        return (
          <div className={cls(css.flex, 'space-x-0.5')}>
            <div
              className={cls('w-1', 'h-1', 'bg-white', 'rounded-full', 'animate-bounce')}
              style={{ animationDelay: '0ms' }}
            />
            <div
              className={cls('w-1', 'h-1', 'bg-white', 'rounded-full', 'animate-bounce')}
              style={{ animationDelay: '150ms' }}
            />
            <div
              className={cls('w-1', 'h-1', 'bg-white', 'rounded-full', 'animate-bounce')}
              style={{ animationDelay: '300ms' }}
            />
          </div>
        );

      case RECORDING_STATES.ERROR:
        return (
          <svg className={uiPatterns.iconSm} fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
          </svg>
        );

      case RECORDING_STATES.COMPLETED:
        return (
          <svg className={uiPatterns.iconSm} fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
          </svg>
        );

      default: // IDLE
        return (
          <svg className={uiPatterns.iconSm} fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M7 4a3 3 0 016 0v4a3 3 0 11-6 0V4zm4 10.93A7.001 7.001 0 0017 8a1 1 0 10-2 0A5 5 0 015 8a1 1 0 00-2 0 7.001 7.001 0 006 6.93V17H6a1 1 0 100 2h8a1 1 0 100-2h-3v-2.07z" clipRule="evenodd" />
          </svg>
        );
    }
  };

  const handleMouseDown = (): void => {
    if (!disabled) {
      setIsPressed(true);
    }
  };

  const handleMouseUp = (): void => {
    setIsPressed(false);
  };

  const handleClick = (): void => {
    if (!disabled) {
      if (onClick) {
        onClick();
      } else if (onToggleRecording) {
        const newState = state === RECORDING_STATES.IDLE ? RECORDING_STATES.RECORDING : RECORDING_STATES.IDLE;
        onToggleRecording(newState);
      }
    }
  };

  const getTooltipText = (): string => {
    switch (state) {
      case RECORDING_STATES.RECORDING:
        return 'Stop recording';
      case RECORDING_STATES.PROCESSING:
        return 'Processing audio...';
      case RECORDING_STATES.ERROR:
        return 'Recording failed - click to retry';
      case RECORDING_STATES.COMPLETED:
        return 'Recording completed';
      default:
        return 'Start recording';
    }
  };

  const tooltipText = getTooltipText();
  const isProcessing = state === RECORDING_STATES.PROCESSING;
  const isRecording = state === RECORDING_STATES.RECORDING;

  return (
    <div className="relative group">
      <button
        className={cls(
          getButtonClasses(),
          className,
          isPressed ? 'transform scale-95' : ''
        )}
        onClick={handleClick}
        onMouseDown={handleMouseDown}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        disabled={disabled || isProcessing}
        title={tooltipText}
        aria-label={tooltipText}
        aria-pressed={isRecording}
        type="button"
      >
        {getIcon()}

        {/* Ripple effect for active states */}
        {isRecording && (
          <div className="absolute inset-0 rounded-full border-2 border-active-speaking animate-ping opacity-30" />
        )}
      </button>

      {
        showTooltip
        ? <div className={cls(TOOLTIP_CLASSES)}>
            {tooltipText}
            <div className="absolute top-full left-1/2 transform -translate-x-1/2 border-2 border-transparent border-t-main-bg" />
          </div>
        :''
      }
    </div>
  );
};

export default RecordingButtonForInput;
