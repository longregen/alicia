import React from 'react';
import { cls } from '../../utils/cls';
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

  const sizeClasses = {
    sm: 'px-3 py-1.5 text-xs',
    md: 'px-4 py-2 text-sm',
    lg: 'px-6 py-3 text-base',
  };

  const buttonClasses = cls(
    // Base styles
    'inline-flex items-center justify-center gap-2',
    'rounded-lg font-medium',
    'transition-all duration-200',
    'focus:outline-none focus:ring-2 focus:ring-accent',

    // Size
    sizeClasses[size],

    // Ghost styling - transparent with subtle hover
    'bg-transparent border border-transparent',
    'text-muted hover:bg-surface hover:text-default hover:border',

    // States
    disabled || loading ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
    fullWidth ? 'w-full' : '',

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
