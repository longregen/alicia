import React from 'react';
import { cls } from '../../utils/cls';
import type { BaseComponentProps, Size, Variant } from '../../types/components';

/**
 * IconButton atom component for icon-only actions.
 * Provides a circular or square button with icon content.
 */

export interface IconButtonProps extends BaseComponentProps {
  /** Button variant style */
  variant?: Variant;
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
  /** ARIA label (required for accessibility) */
  ariaLabel: string;
  /** Circular shape (default: square with rounded corners) */
  circular?: boolean;
  /** Icon element or SVG */
  icon: React.ReactNode;
}

const IconButton: React.FC<IconButtonProps> = ({
  variant = 'default',
  size = 'md',
  disabled = false,
  loading = false,
  onClick,
  type = 'button',
  ariaLabel,
  circular = false,
  icon,
  className = '',
}) => {
  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (!disabled && !loading && onClick) {
      onClick(e);
    }
  };

  const sizeClasses = {
    sm: 'p-1.5',
    md: 'p-2',
    lg: 'p-3',
  };

  const iconSize = size === 'sm' ? 'w-3 h-3' : size === 'md' ? 'w-4 h-4' : 'w-5 h-5';

  const variantClasses = {
    default: 'bg-surface text-default border hover:bg-sunken',
    primary: 'bg-accent text-on-emphasis hover:bg-accent-hover active:bg-accent-active border-accent',
    success: 'bg-success text-on-emphasis hover:bg-success/90 border-success',
    warning: 'bg-warning text-on-emphasis hover:bg-warning/90 border-warning',
    error: 'bg-error text-on-emphasis hover:bg-error/90 border-error',
  };

  const buttonClasses = cls(
    // Base styles
    'inline-flex items-center justify-center',
    'border transition-all duration-200',
    'focus:outline-none focus:ring-2 focus:ring-accent',

    // Padding
    sizeClasses[size],

    // Shape
    circular ? 'rounded-full' : 'rounded-lg',

    // Variant
    variantClasses[variant],

    // States
    disabled || loading ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',

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
      {loading ? (
        <div className={cls(iconSize, 'border-2 border-current border-t-transparent rounded-full animate-spin')} />
      ) : (
        <div className={iconSize}>{icon}</div>
      )}
    </button>
  );
};

export default IconButton;
