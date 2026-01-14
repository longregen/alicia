import React, { useState } from 'react';
import { cls } from '../../utils/cls';
import type { BaseComponentProps } from '../../types/components';

// Component-specific constants
const BUTTON_BASE_CLASSES = [
  'w-10 h-10', // Medium size only
  'rounded-full',
  'transition-all',
  'duration-200',
  'ease-in-out',
  'flex',
  'items-center',
  'justify-center',
  'font-medium',
  'relative',
  'overflow-hidden',
  'focus:outline-none',
  'active:transform',
  'active:scale-95',
  'group',
] as const;

const DISABLED_CLASSES = [
  'bg-muted',
  'text-muted-foreground',
  'cursor-not-allowed',
] as const;

const CANNOT_SEND_CLASSES = [
  'bg-card',
  'text-muted-foreground',
  'hover:bg-muted',
  'hover:text-accent-foreground',
] as const;

// Primary variant styling only
const PRIMARY_CLASSES = [
  'bg-accent',
  'hover:bg-accent/90',
  'active:bg-accent/80',
  'text-accent-foreground',
] as const;

// Tooltip classes for "up" position only
const TOOLTIP_BASE_CLASSES = [
  'absolute',
  'bottom-full',
  'left-1/2',
  'transform',
  '-translate-x-1/2',
  'mb-2',
  'origin-bottom',
  'px-3',
  'py-1',
  'bg-popover',
  'text-popover-foreground',
  'text-xs',
  'rounded',
  'whitespace-nowrap',
  'opacity-0',
  'scale-50',
  'group-hover:opacity-100',
  'group-hover:scale-100',
  'transition-all',
  'duration-100',
  'ease-out',
  'pointer-events-none',
  'z-10',
] as const;

const TOOLTIP_ARROW_CLASSES = [
  'absolute',
  'top-full',
  'left-1/2',
  'transform',
  '-translate-x-1/2',
  'border-2',
  'border-transparent',
  'border-t-popover'
] as const;

// Simplified component props interface
export interface InputSendButtonProps extends BaseComponentProps {
  /** Callback fired when send button is clicked */
  onSend?: () => void;
  /** Whether the button is disabled */
  disabled?: boolean;
  /** Whether the button can send (typically based on input content) */
  canSend?: boolean;
  /** Tooltip text to display. If null/undefined, no tooltip is shown */
  tooltipText?: string | null;
}

const InputSendButton: React.FC<InputSendButtonProps> = ({
  onSend,
  disabled = false,
  canSend = false,
  className = '',
  tooltipText
}) => {
  const [isPressed, setIsPressed] = useState<boolean>(false);

  const getButtonClasses = (): string => {
    if (disabled) {
      return cls(BUTTON_BASE_CLASSES, DISABLED_CLASSES);
    }

    if (!canSend) {
      return cls(BUTTON_BASE_CLASSES, CANNOT_SEND_CLASSES);
    }

    return cls(BUTTON_BASE_CLASSES, PRIMARY_CLASSES);
  };

  const getIcon = (): React.JSX.Element => {
    // Always show paper airplane, just with different opacity/styling
    return (
      <svg
        className={cls(
          "w-5 h-5 transition-transform duration-200 rotate-90 translate-x-0.5",
          canSend ? "group-hover:scale-110" : "opacity-50"
        )}
        fill="currentColor"
        viewBox="0 0 20 20"
      >
        <path d="M10.894 2.553a1 1 0 00-1.788 0l-7 14a1 1 0 001.169 1.409l5-1.429A1 1 0 009 15.571V11a1 1 0 112 0v4.571a1 1 0 00.725.962l5 1.428a1 1 0 001.17-1.408l-7-14z" />
      </svg>
    );
  };

  const handleMouseDown = (): void => {
    if (!disabled && canSend) {
      setIsPressed(true);
    }
  };

  const handleMouseUp = (): void => {
    setIsPressed(false);
  };

  const handleClick = (): void => {
    if (!disabled && canSend && onSend) {
      onSend();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent): void => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleClick();
    }
  };

  const getAriaLabel = (): string => {
    if (!canSend) return 'Send message (disabled - no message to send)';
    return 'Send message';
  };

  const ariaLabel = getAriaLabel();

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
        onKeyDown={handleKeyDown}
        disabled={disabled || !canSend}
        title={tooltipText || undefined}
        aria-label={ariaLabel}
        type="submit"
      >
        {getIcon()}

        {/* Shine effect on hover - always present but different intensity */}
        <div className={cls(
          "absolute inset-0 rounded-full transition-opacity duration-300",
          canSend
            ? "bg-accent/15 opacity-0 group-hover:opacity-20"
            : "bg-accent/15 opacity-0 group-hover:opacity-10"
        )} />
      </button>

      {/* Tooltip (always positioned up) */}
      {tooltipText && (
        <div className={cls(TOOLTIP_BASE_CLASSES)}>
          {tooltipText}
          <div className={cls(TOOLTIP_ARROW_CLASSES)} />
        </div>
      )}
    </div>
  );
};

export default InputSendButton;
