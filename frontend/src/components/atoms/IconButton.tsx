import React from 'react';
import { cls } from '../../utils/cls';
import { CSS, variantClasses } from '../../utils/constants';
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

  const iconSize = size === 'sm' ? 'w-3 h-3' : size === 'md' ? 'w-4 h-4' : 'w-5 h-5';
  const paddingSize = size === 'sm' ? 'p-1.5' : size === 'md' ? 'p-2' : 'p-3';

  const buttonClasses = cls(
    // Base styles
    CSS.flex,
    CSS.itemsCenter,
    CSS.justifyCenter,
    CSS.border,
    CSS.transitionAll,
    CSS.duration200,
    CSS.focusOutlineNone,
    CSS.focusRing2,
    CSS.focusRingAlicia500,

    // Padding
    paddingSize,

    // Shape
    circular ? CSS.roundedFull : CSS.rounded,

    // Variant
    variantClasses[variant].background,
    variantClasses[variant].text,
    variantClasses[variant].border,
    variantClasses[variant].hover,

    // States
    disabled || loading ? cls(CSS.cursorNotAllowed, CSS.opacity50) : CSS.cursorPointer,

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
