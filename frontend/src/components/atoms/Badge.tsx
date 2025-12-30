import React from 'react';
import { cls } from '../../utils/cls';
import { CSS } from '../../utils/constants';
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
    default: 'bg-surface-bg text-primary-text border-gray-300 dark:border-gray-600',
    primary: 'bg-primary-blue-glow text-primary-blue border-primary-blue',
    success: 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 border-green-300 dark:border-green-700',
    warning: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400 border-yellow-300 dark:border-yellow-700',
    error: 'bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 border-red-300 dark:border-red-700',
  };

  const badgeClasses = cls(
    // Base styles
    'inline-flex',
    CSS.itemsCenter,
    CSS.gap1,
    'rounded-full',
    'border',
    CSS.fontMedium,
    'whitespace-nowrap',

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
