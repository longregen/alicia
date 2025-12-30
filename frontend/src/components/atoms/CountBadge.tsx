import React from 'react';
import Badge, { type BadgeProps } from './Badge';

/**
 * CountBadge atom component for displaying numerical counts.
 * Used for notifications, vote counts, and other numerical indicators.
 */

export interface CountBadgeProps extends Omit<BadgeProps, 'children'> {
  /** Count value to display */
  count: number;
  /** Maximum count to display before showing "+" (e.g., "99+") */
  max?: number;
  /** Show zero counts (default: false) */
  showZero?: boolean;
}

const CountBadge: React.FC<CountBadgeProps> = ({
  count,
  max = 99,
  showZero = false,
  ...props
}) => {
  // Don't render if count is 0 and showZero is false
  if (count === 0 && !showZero) {
    return null;
  }

  // Format count with max threshold
  const displayCount = count > max ? `${max}+` : count.toString();

  return (
    <Badge {...props} size={props.size || 'sm'}>
      {displayCount}
    </Badge>
  );
};

export default CountBadge;
