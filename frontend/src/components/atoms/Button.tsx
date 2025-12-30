import React from 'react';
import { cls } from '../../utils/cls';
import { CSS, sizeClasses, variantClasses } from '../../utils/constants';
import type { BaseComponentProps, Size, Variant } from '../../types/components';

/**
 * Base button component with support for different variants and sizes.
 * Provides the foundation for IconButton, PrimaryButton, and GhostButton.
 */

export interface ButtonProps extends BaseComponentProps {
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
  /** ARIA label */
  ariaLabel?: string;
  /** Full width button */
  fullWidth?: boolean;
}

const Button: React.FC<ButtonProps> = ({
  variant = 'default',
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
    CSS.border,
    CSS.rounded,
    CSS.fontMedium,
    CSS.transitionAll,
    CSS.duration200,
    CSS.focusOutlineNone,
    CSS.focusRing2,
    CSS.focusRingAlicia500,

    // Size
    sizeClasses[size].button,

    // Variant
    variantClasses[variant].background,
    variantClasses[variant].text,
    variantClasses[variant].border,
    variantClasses[variant].hover,

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

export default Button;
