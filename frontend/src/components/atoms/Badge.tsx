import React from 'react';
import { cls } from '../../utils/cls';
import type { BaseComponentProps, Variant, Size } from '../../types/components';

/**
 * Base Badge component for displaying status, scores, and counts.
 * Provides the foundation for StatusBadge, ScoreBadge, and CountBadge.
 */

export interface BadgeProps extends BaseComponentProps {
  /** Badge variant style */
  variant?: Variant;
  /** Badge size */
  size?: Size;
  /** Optional icon or prefix */
  icon?: React.ReactNode;
  /** Optional dot indicator */
  showDot?: boolean;
  /** Dot color (only shown if showDot is true) */
  dotColor?: string;
}

const Badge: React.FC<BadgeProps> = ({
  variant = 'default',
  size = 'md',
  icon,
  showDot = false,
  dotColor,
  className = '',
  children,
}) => {
  const sizeClasses = {
    sm: 'px-1.5 py-0.5 text-xs',
    md: 'px-2 py-1 text-sm',
    lg: 'px-3 py-1.5 text-base',
  };

  const variantStyles = {
    default: 'bg-surface text-default border',
    primary: 'bg-accent-subtle text-accent border-accent',
    success: 'bg-success-subtle text-success border-success',
    warning: 'bg-warning-subtle text-warning border-warning',
    error: 'bg-error-subtle text-error border-error',
  };

  const badgeClasses = cls(
    // Base styles
    'inline-flex items-center gap-1',
    'rounded-full border font-medium whitespace-nowrap',

    // Size
    sizeClasses[size],

    // Variant
    variantStyles[variant],

    // Custom classes
    className
  );

  const dotColorClass = dotColor || 'bg-current';

  return (
    <span className={badgeClasses}>
      {showDot && (
        <span className={cls('w-1.5 h-1.5 rounded-full', dotColorClass)} />
      )}
      {icon && <span className="flex items-center">{icon}</span>}
      {children}
    </span>
  );
};

export default Badge;
