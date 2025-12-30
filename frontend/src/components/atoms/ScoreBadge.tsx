import React from 'react';
import Badge, { type BadgeProps } from './Badge';
import type { Variant } from '../../types/components';

/**
 * ScoreBadge atom component for displaying numerical scores.
 * Automatically determines variant based on score thresholds.
 */

export interface ScoreBadgeProps extends Omit<BadgeProps, 'variant'> {
  /** Numerical score value (0-100 or 0-1) */
  score: number;
  /** Maximum score value (default: 100) */
  max?: number;
  /** Show percentage symbol */
  showPercent?: boolean;
  /** Custom thresholds for variant colors */
  thresholds?: {
    error: number;
    warning: number;
    success: number;
  };
}

const ScoreBadge: React.FC<ScoreBadgeProps> = ({
  score,
  max = 100,
  showPercent = false,
  thresholds = {
    error: 40,
    warning: 70,
    success: 85,
  },
  children,
  ...props
}) => {
  // Normalize score to percentage
  const percentage = max === 1 ? score * 100 : (score / max) * 100;

  // Determine variant based on thresholds
  let variant: Variant = 'default';
  if (percentage >= thresholds.success) {
    variant = 'success';
  } else if (percentage >= thresholds.warning) {
    variant = 'warning';
  } else if (percentage < thresholds.error) {
    variant = 'error';
  } else {
    variant = 'warning';
  }

  // Format display value
  const displayValue = children || (
    showPercent ? `${percentage.toFixed(0)}%` : score.toFixed(max === 1 ? 2 : 0)
  );

  return (
    <Badge {...props} variant={variant}>
      {displayValue}
    </Badge>
  );
};

export default ScoreBadge;
