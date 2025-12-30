import React from 'react';
import { cls } from '../../utils/cls';
import { CSS, sizeClasses } from '../../utils/constants';
import type { BaseComponentProps, Size } from '../../types/components';

/**
 * GhostButton atom component for subtle actions.
 * Transparent background with minimal styling, shows on hover.
 */

export interface GhostButtonProps extends BaseComponentProps {
  /** Button size */
  size?: Size;
  /** Whether the button is disabled */
  disabled?: boolean;
  /** Whether the button is in loading state */
  loading?: boolean;
  /** Click handler */
  onClick?: (e: React.MouseEvent<HTMLButtonElement>) => void;
  /** Button type */
  type?: 'button' | 'submit' | 'reset';
  /** ARIA label */
  ariaLabel?: string;
  /** Full width button */
  fullWidth?: boolean;
}

const GhostButton: React.FC<GhostButtonProps> = ({
  size = 'md',
  disabled = false,
  loading = false,
  onClick,
  type = 'button',
  ariaLabel,
  fullWidth = false,
  className = '',
  children,
}) => {
  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (!disabled && !loading && onClick) {
      onClick(e);
    }
  };

  const buttonClasses = cls(
    // Base styles
    CSS.flex,
    CSS.itemsCenter,
    CSS.justifyCenter,
    CSS.gap2,
    CSS.rounded,
    CSS.fontMedium,
    CSS.transitionAll,
    CSS.duration200,
    CSS.focusOutlineNone,
    CSS.focusRing2,
    CSS.focusRingAlicia500,

    // Size
    sizeClasses[size].button,

    // Ghost styling - transparent with subtle hover
    'bg-transparent',
    'border border-transparent',
    CSS.textMuted,
    'hover:bg-surface-bg hover:text-primary-text hover:border-gray-300 dark:hover:border-gray-600',

    // States
    disabled || loading ? cls(CSS.cursorNotAllowed, CSS.opacity50) : CSS.cursorPointer,
    fullWidth ? CSS.wFull : '',

    // Custom classes
    className
  );

  return (
    <button
      type={type}
      onClick={handleClick}
      disabled={disabled || loading}
      aria-label={ariaLabel}
      className={buttonClasses}
    >
      {loading && (
        <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
      )}
      {children}
    </button>
  );
};

export default GhostButton;
